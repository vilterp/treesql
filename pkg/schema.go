package treesql

import (
	"encoding/binary"
	"fmt"
	"strconv"

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

func (column *columnDescriptor) toRecord(tableName string, db *Database) *record {
	columnsTable := db.schema.tables["__columns__"]
	record := columnsTable.NewRecord()
	record.setString("id", fmt.Sprintf("%d", column.id))
	record.setString("name", column.name)
	record.setString("table_name", tableName)
	record.setString("type", column.typ.Format().String())
	if column.referencesColumn != nil {
		record.setString("references", column.referencesColumn.tableName)
	}
	return record
}

func columnFromRecord(record *record) *columnDescriptor {
	idInt, _ := strconv.Atoi(record.GetField("id").stringVal)
	references := record.GetField("references").stringVal
	var colRef *columnReference
	if len(references) > 0 { // should things be nullable? idk
		colRef = &columnReference{
			tableName: references,
		}
	}
	typ, err := lang.ParseType(record.GetField("type").stringVal)
	if err != nil {
		// TODO: something other than panic
		panic(fmt.Sprintf("error parsing type: %v", err))
	}
	return &columnDescriptor{
		id:               idInt,
		name:             record.GetField("name").stringVal,
		typ:              typ,
		referencesColumn: colRef,
	}
}

func (table *tableDescriptor) toRecord(db *Database) *record {
	record := db.schema.tables["__tables__"].NewRecord()
	record.setString("name", table.name)
	record.setString("primary_key", table.primaryKey)
	return record
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
	tablesTable := db.schema.tables["__tables__"]
	columnsTable := db.schema.tables["__columns__"]
	db.boltDB.View(func(tx *bolt.Tx) error {
		tablesDescs := map[string]*tableDescriptor{}
		// Load all table descriptors.
		if err := tx.Bucket([]byte("__tables__")).ForEach(func(_ []byte, tableBytes []byte) error {
			tableRecord := tablesTable.RecordFromBytes(tableBytes)
			tableDesc := &tableDescriptor{
				name:       tableRecord.GetField("name").stringVal,
				primaryKey: tableRecord.GetField("primary_key").stringVal,
				columns:    make([]*columnDescriptor, 0),
			}
			tablesDescs[tableDesc.name] = tableDesc
			return nil
		}); err != nil {
			return err
		}
		// Load all column descriptors; stick them on table descriptors.
		if err := tx.Bucket([]byte("__columns__")).ForEach(func(key []byte, columnBytes []byte) error {
			columnRecord := columnsTable.RecordFromBytes(columnBytes)
			columnSpec := columnFromRecord(columnRecord)
			tableDesc := tablesDescs[columnRecord.GetField("table_name").stringVal]
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
