package treesql

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
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
}

type ColumnName string
type ColumnDescriptor struct {
	ID               int
	Name             string
	Type             ColumnType
	ReferencesColumn *ColumnReference
}

type ColumnReference struct {
	TableName string // we're gonna assume for now that you can only reference the primary key
}

// maybe I should use that iota weirdness
type ColumnType byte

const TypeString ColumnType = 0
const TypeInt ColumnType = 1

var TypeToName = map[ColumnType]string{
	TypeString: "string",
	TypeInt:    "int",
}

var NameToType = map[string]ColumnType{
	"string": TypeString,
	"int":    TypeInt,
}

func (column *ColumnDescriptor) ToRecord(tableName string, db *Database) *Record {
	columnsTable := db.Schema.Tables["__columns__"]
	record := columnsTable.NewRecord()
	record.SetString("id", fmt.Sprintf("%d", column.ID))
	record.SetString("name", column.Name)
	record.SetString("table_name", tableName)
	record.SetString("type", TypeToName[column.Type])
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
	return &ColumnDescriptor{
		ID:               idInt,
		Name:             record.GetField("name").StringVal,
		Type:             NameToType[record.GetField("type").StringVal],
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
		tables := map[string]*TableDescriptor{}
		tx.Bucket([]byte("__tables__")).ForEach(func(_ []byte, tableBytes []byte) error {
			tableRecord := tablesTable.RecordFromBytes(tableBytes)
			tableSpec := db.AddTable(
				tableRecord.GetField("name").StringVal,
				tableRecord.GetField("primary_key").StringVal,
				make([]*ColumnDescriptor, 0),
			)
			tables[tableSpec.Name] = tableSpec
			return nil
		})
		tx.Bucket([]byte("__columns__")).ForEach(func(key []byte, columnBytes []byte) error {
			columnRecord := columnsTable.RecordFromBytes(columnBytes)
			columnSpec := ColumnFromRecord(columnRecord)
			tableSpec := tables[columnRecord.GetField("table_name").StringVal]
			tableSpec.Columns = append(tableSpec.Columns, columnSpec)
			return nil
		})
		return nil
	})
}

func (db *Database) AddTable(name string, primaryKey string, columns []*ColumnDescriptor) *TableDescriptor {
	table := &TableDescriptor{
		Name:       name,
		PrimaryKey: primaryKey,
		Columns:    columns,
	}
	table.LiveQueryInfo = table.NewLiveQueryInfo() // def something weird about this
	db.Schema.Tables[name] = table
	go table.HandleEvents()
	return table
}

func EmptySchema() *Schema {
	return &Schema{
		Tables: map[string]*TableDescriptor{},
	}
}

func (db *Database) AddBuiltinSchema() {
	// these never go in the on-disk __tables__ and __columns__ Bolt buckets
	// doing ids like this is kind of precarious...
	db.AddTable("__tables__", "name", []*ColumnDescriptor{
		{
			ID:   0,
			Name: "name",
			Type: TypeString,
		},
		{
			ID:   1,
			Name: "primary_key",
			Type: TypeString,
		},
	})
	db.AddTable("__columns__", "id", []*ColumnDescriptor{
		{
			ID:   2,
			Name: "id",
			Type: TypeString, // TODO: switch to int when they work
		},
		{
			ID:   3,
			Name: "name",
			Type: TypeString,
		},
		{
			ID:   4,
			Name: "table_name",
			Type: TypeString,
			ReferencesColumn: &ColumnReference{
				TableName: "__tables__",
			},
		},
		{
			ID:   5,
			Name: "type",
			Type: TypeString,
		},
		{
			ID:   6,
			Name: "references", // TODO: this is a keyword. rename to "references_table"
			Type: TypeString,
		},
	})
	db.AddTable("__record_listeners__", "id", []*ColumnDescriptor{
		{
			ID:   7,
			Name: "id",
			Type: TypeString,
		},
		{
			ID:   8,
			Name: "connection_id",
			Type: TypeString,
		},
		{
			ID:   9,
			Name: "channel_id",
			Type: TypeString,
		},
		{
			ID:   10,
			Name: "table_name",
			Type: TypeString,
			ReferencesColumn: &ColumnReference{
				TableName: "__tables__",
			},
		},
		{
			ID:   11,
			Name: "pk_value",
			Type: TypeString,
		},
		{
			ID:   12,
			Name: "query_path",
			Type: TypeString,
		},
	})
	db.Schema.NextColumnID = 13 // ugh magic numbers.
}

// TODO: __connections__, __channels__, __whole_table_listeners__, __filtered_table_listeners__
