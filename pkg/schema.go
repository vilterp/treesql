package treesql

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/vilterp/treesql/package/lang"
)

type Schema struct {
	Tables       map[string]*TableDescriptor
	NextColumnID int
}

// TODO: better name, or refactor. not just a descriptor, since
// it also holds live query info.
type TableDescriptor struct {
	Name          string
	Columns       []*ColumnDescriptor
	PrimaryKey    string
	LiveQueryInfo *LiveQueryInfo
	IsBuiltin     bool
}

func (table *TableDescriptor) getType() *lang.TRecord {
	types := map[string]lang.Type{}
	for _, col := range table.Columns {
		types[col.Name] = col.Type
	}
	return &lang.TRecord{Types: types}
}

func (table *TableDescriptor) colIDForName(name string) (int, error) {
	for _, col := range table.Columns {
		if col.Name == name {
			return col.ID, nil
		}
	}
	return 0, fmt.Errorf("col not found: %s", name)
}

type ColumnName string
type ColumnDescriptor struct {
	ID               int
	Name             string
	Type             lang.Type
	ReferencesColumn *ColumnReference
}

type ColumnReference struct {
	TableName string // we're gonna assume for now that you can only reference the primary key
}

func (column *ColumnDescriptor) ToRecord(tableName string, db *Database) *Record {
	columnsTable := db.Schema.Tables["__columns__"]
	record := columnsTable.NewRecord()
	record.SetString("id", fmt.Sprintf("%d", column.ID))
	record.SetString("name", column.Name)
	record.SetString("table_name", tableName)
	record.SetString("type", column.Type.Format().String())
	if column.ReferencesColumn != nil {
		record.SetString("references", column.ReferencesColumn.TableName)
	}
	return record
}

func ColumnFromRecord(record *Record) *ColumnDescriptor {
	idInt, _ := strconv.Atoi(record.GetField("id").StringVal)
	references := record.GetField("references").StringVal
	var columnReference *ColumnReference
	if len(references) > 0 { // should things be nullable? idk
		columnReference = &ColumnReference{
			TableName: references,
		}
	}
	typ, err := lang.ParseType(record.GetField("type").StringVal)
	if err != nil {
		// TODO: something other than panic
		panic(fmt.Sprintf("error parsing type: %v", err))
	}
	return &ColumnDescriptor{
		ID:               idInt,
		Name:             record.GetField("name").StringVal,
		Type:             typ,
		ReferencesColumn: columnReference,
	}
}

func (table *TableDescriptor) ToRecord(db *Database) *Record {
	record := db.Schema.Tables["__tables__"].NewRecord()
	record.SetString("name", table.Name)
	record.SetString("primary_key", table.PrimaryKey)
	return record
}

func TableFromRecord(record *Record) *TableDescriptor {
	return &TableDescriptor{
		Columns:    make([]*ColumnDescriptor, 0),
		Name:       record.GetField("name").StringVal,
		PrimaryKey: record.GetField("primary_key").StringVal,
	}
}

func (db *Database) EnsureBuiltinSchema() {
	db.BoltDB.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte("__tables__"))
		tx.CreateBucketIfNotExists([]byte("__columns__"))
		sequencesBucket, _ := tx.CreateBucketIfNotExists([]byte("__sequences__"))
		// sync next column id
		nextColumnIDBytes := sequencesBucket.Get([]byte("__next_column_id__"))
		if nextColumnIDBytes == nil {
			// write it
			nextColumnIDBytes = make([]byte, 4)
			binary.BigEndian.PutUint32(nextColumnIDBytes, uint32(db.Schema.NextColumnID))
			sequencesBucket.Put([]byte("__next_column_id__"), nextColumnIDBytes)
		} else {
			// read it
			nextColumnID := binary.BigEndian.Uint32(nextColumnIDBytes)
			db.Schema.NextColumnID = int(nextColumnID)
		}
		return nil
	})
}

func (db *Database) LoadUserSchema() {
	tablesTable := db.Schema.Tables["__tables__"]
	columnsTable := db.Schema.Tables["__columns__"]
	db.BoltDB.View(func(tx *bolt.Tx) error {
		tablesDescs := map[string]*TableDescriptor{}
		// Load all table descriptors.
		if err := tx.Bucket([]byte("__tables__")).ForEach(func(_ []byte, tableBytes []byte) error {
			tableRecord := tablesTable.RecordFromBytes(tableBytes)
			tableDesc := &TableDescriptor{
				Name:       tableRecord.GetField("name").StringVal,
				PrimaryKey: tableRecord.GetField("primary_key").StringVal,
				Columns:    make([]*ColumnDescriptor, 0),
			}
			tablesDescs[tableDesc.Name] = tableDesc
			return nil
		}); err != nil {
			return err
		}
		// Load all column descriptors; stick them on table descriptors.
		if err := tx.Bucket([]byte("__columns__")).ForEach(func(key []byte, columnBytes []byte) error {
			columnRecord := columnsTable.RecordFromBytes(columnBytes)
			columnSpec := ColumnFromRecord(columnRecord)
			tableDesc := tablesDescs[columnRecord.GetField("table_name").StringVal]
			tableDesc.Columns = append(tableDesc.Columns, columnSpec)
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
// buildTableDescriptor converts a CREATE TABLE AST node into a TableDescriptor.
// It also assigns column ids.
func (db *Database) buildTableDescriptor(create *CreateTable) (*TableDescriptor, error) {
	// find primary key
	var primaryKey string
	for _, column := range create.Columns {
		if column.PrimaryKey {
			primaryKey = column.Name
			break
		}
	}
	// Create table descriptor
	tableDesc := &TableDescriptor{
		Name:       create.Name,
		PrimaryKey: primaryKey,
		Columns:    make([]*ColumnDescriptor, len(create.Columns)),
	}
	// Create column descriptors
	for idx, parsedColumn := range create.Columns {
		// extract reference
		var reference *ColumnReference
		if parsedColumn.References != nil {
			reference = &ColumnReference{
				TableName: *parsedColumn.References,
			}
		}
		// parse type
		typ, err := lang.ParseType(parsedColumn.TypeName)
		if err != nil {
			return nil, fmt.Errorf("error parsing type: %v", err)
		}
		// build column spec
		columnSpec := &ColumnDescriptor{
			ID:               db.Schema.NextColumnID,
			Name:             parsedColumn.Name,
			ReferencesColumn: reference,
			Type:             typ,
		}
		// TODO: synchronize access to this
		db.Schema.NextColumnID++
		tableDesc.Columns[idx] = columnSpec
	}

	return tableDesc, nil
}

// addTableDescriptor initializes the table's LiveQueryInfo
// and adds it to the schema.
func (db *Database) addTableDescriptor(table *TableDescriptor) {
	table.LiveQueryInfo = table.NewLiveQueryInfo() // def something weird about this
	go table.HandleEvents()
	// TODO: synchronize access to this
	db.Schema.Tables[table.Name] = table
}

func EmptySchema() *Schema {
	return &Schema{
		Tables: map[string]*TableDescriptor{},
	}
}
