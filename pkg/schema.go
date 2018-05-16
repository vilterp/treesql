package treesql

import (
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/vilterp/treesql/pkg/lang"
)

type schema struct {
	tables       map[string]*tableDescriptor
	nextColumnID int
}

// TODO: better name, or refactor. not just a descriptor, since
// it also holds live query info.
type tableDescriptor struct {
	name          string
	columns       []*columnDescriptor
	primaryKey    string
	liveQueryInfo *liveQueryInfo
	isBuiltin     bool
}

func (table *tableDescriptor) getType() *lang.TRecord {
	types := map[string]lang.Type{}
	for _, col := range table.columns {
		types[col.name] = col.typ
	}
	return lang.NewTRecord(types)
}

func (table *tableDescriptor) colIDForName(name string) (int, error) {
	desc, err := table.getColDesc(name)
	if err != nil {
		return 0, err
	}
	return desc.id, nil
}

func (table *tableDescriptor) colReferencingTable(otherName string) *string {
	for _, col := range table.columns {
		if col.referencesColumn != nil && col.referencesColumn.tableName == otherName {
			return &col.name
		}
	}
	return nil
}

func (table *tableDescriptor) getColDesc(name string) (*columnDescriptor, error) {
	for _, col := range table.columns {
		if col.name == name {
			return col, nil
		}
	}
	return nil, fmt.Errorf("col not found: %s", name)
}

func (table *tableDescriptor) pkBucketKey() []byte {
	// Find id of PK column.
	var pkID int
	for _, col := range table.columns {
		if col.name == table.primaryKey {
			pkID = col.id
			break
		}
	}

	// TODO: factor this out to an "encoding" or "keys" file
	pkIDBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(pkIDBytes, uint32(pkID))

	return pkIDBytes
}

func (table *tableDescriptor) getPKType() lang.Type {
	colDesc, err := table.getColDesc(table.primaryKey)
	if err != nil {
		panic(fmt.Sprintf("pk doesn't exist on descriptor: %s.%s", table.name, table.primaryKey))
	}
	return colDesc.typ
}

type columnName string
type columnDescriptor struct {
	id               int
	name             string
	typ              lang.Type
	referencesColumn *columnReference
}

type columnReference struct {
	tableName string // we're gonna assume for now that you can only reference the primary key
}

func (column *columnDescriptor) toRecord(tableName string, db *Database) *lang.VRecord {
	values := map[string]lang.Value{
		"id":         lang.NewVInt(column.id),
		"name":       lang.NewVString(column.name),
		"table_name": lang.NewVString(tableName),
		"type":       lang.NewVString(column.typ.Format().String()),
	}

	if column.referencesColumn != nil {
		values["references"] = lang.NewVString(column.referencesColumn.tableName)
	}
	return lang.NewVRecord(values)
}

func columnFromVal(record *lang.VRecord) (*columnDescriptor, error) {
	idInt := record.GetValue("id").(*lang.VInt)
	name := record.GetValue("name").(*lang.VString)

	maybeReferences := record.GetValue("references")
	var references *lang.VString
	if maybeReferences != nil {
		references = maybeReferences.(*lang.VString)
	}

	var colRef *columnReference
	if references != nil {
		colRef = &columnReference{
			tableName: string(*references),
		}
	}
	typStr := record.GetValue("type").(*lang.VString)
	typ, err := lang.ParseType(string(*typStr))
	if err != nil {
		return nil, fmt.Errorf("error parsing type: %v", err)
	}
	return &columnDescriptor{
		id:               int(*idInt),
		name:             string(*name),
		typ:              typ,
		referencesColumn: colRef,
	}, nil
}

func (table *tableDescriptor) toRecord(db *Database) *lang.VRecord {
	return lang.NewVRecord(map[string]lang.Value{
		"name":        lang.NewVString(table.name),
		"primary_key": lang.NewVString(table.primaryKey),
	})
}

func (db *Database) ensureBuiltinSchema() {
	db.boltDB.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("__tables__"))
		tx.CreateBucketIfNotExists([]byte("__columns__"))
		sequencesBucket, _ := tx.CreateBucketIfNotExists([]byte("__sequences__"))
		// sync next column id
		nextColumnIDBytes := sequencesBucket.Get([]byte("__next_column_id__"))
		if nextColumnIDBytes == nil {
			// write it
			nextColumnIDBytes = make([]byte, 4)
			binary.BigEndian.PutUint32(nextColumnIDBytes, uint32(db.schema.nextColumnID))
			sequencesBucket.Put([]byte("__next_column_id__"), nextColumnIDBytes)
		} else {
			// read it
			nextColumnID := binary.BigEndian.Uint32(nextColumnIDBytes)
			db.schema.nextColumnID = int(nextColumnID)
		}
		return nil
	})
}

func (db *Database) loadUserSchema() {
	db.boltDB.View(func(tx *bolt.Tx) error {
		tablesDescs := map[string]*tableDescriptor{}
		// Load all table descriptors.
		if err := tx.Bucket([]byte("__tables__")).ForEach(func(_ []byte, tableBytes []byte) error {
			// Parse table record.
			tableVal, err := lang.Decode(tableBytes)
			if err != nil {
				return err
			}
			tableRecord, ok := tableVal.(*lang.VRecord)
			if !ok {
				return fmt.Errorf("table descriptor not a record but a %T", tableVal)
			}
			// Parse its fields.
			name := tableRecord.GetValue("name").(*lang.VString)
			primaryKey := tableRecord.GetValue("primary_key").(*lang.VString)

			tableDesc := &tableDescriptor{
				name:       string(*name),
				primaryKey: string(*primaryKey),
				columns:    make([]*columnDescriptor, 0),
			}
			tablesDescs[tableDesc.name] = tableDesc
			return nil
		}); err != nil {
			return err
		}
		// Load all column descriptors; stick them on table descriptors.
		if err := tx.Bucket([]byte("__columns__")).ForEach(func(key []byte, columnBytes []byte) error {
			columnVal, err := lang.Decode(columnBytes)
			if err != nil {
				return err
			}
			columnRecord, ok := columnVal.(*lang.VRecord)
			if !ok {
				return fmt.Errorf("column descriptor not a record but a %T", columnVal)
			}

			columnSpec, err := columnFromVal(columnRecord)
			if err != nil {
				return err
			}
			tableName := columnRecord.GetValue("table_name").(*lang.VString)
			tableDesc := tablesDescs[string(*tableName)]
			tableDesc.columns = append(tableDesc.columns, columnSpec)
			return nil
		}); err != nil {
			return err
		}
		// Add them to the in-memory schema.
		for _, tableDesc := range tablesDescs {
			db.addTableDescriptor(tableDesc)
		}
		return nil
	})
}

// TODO: move to schema? idk
// buildTableDescriptor converts a CREATE TABLE AST node into a tableDescriptor.
// It also assigns column ids.
func (db *Database) buildTableDescriptor(create *CreateTable) (*tableDescriptor, error) {
	// find primary key
	var primaryKey string
	for _, column := range create.Columns {
		if column.PrimaryKey {
			primaryKey = column.Name
			break
		}
	}
	// Create table descriptor
	tableDesc := &tableDescriptor{
		name:       create.Name,
		primaryKey: primaryKey,
		columns:    make([]*columnDescriptor, len(create.Columns)),
	}
	// Create column descriptors
	for idx, parsedColumn := range create.Columns {
		// extract reference
		var reference *columnReference
		if parsedColumn.References != nil {
			reference = &columnReference{
				tableName: *parsedColumn.References,
			}
		}
		// parse type
		typ, err := lang.ParseType(parsedColumn.TypeName)
		if err != nil {
			return nil, fmt.Errorf("error parsing type: %v", err)
		}
		// build column spec
		columnSpec := &columnDescriptor{
			id:               db.schema.nextColumnID,
			name:             parsedColumn.Name,
			referencesColumn: reference,
			typ:              typ,
		}
		// TODO: synchronize access to this
		db.schema.nextColumnID++
		tableDesc.columns[idx] = columnSpec
	}

	return tableDesc, nil
}

// addTableDescriptor initializes the table's liveQueryInfo
// and adds it to the schema.
func (db *Database) addTableDescriptor(table *tableDescriptor) {
	table.liveQueryInfo = table.newLiveQueryInfo() // def something weird about this
	go table.handleEvents()
	// TODO: synchronize access to this
	db.schema.tables[table.name] = table
}

func emptySchema() *schema {
	return &schema{
		tables: map[string]*tableDescriptor{},
	}
}
