package treesql

import (
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
)

type Schema struct {
	Tables       map[string]*Table
	NextColumnId int
}

type Table struct {
	Name       string
	Columns    []*Column
	PrimaryKey string
}

type Column struct {
	Id               int
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

func (column *Column) ToRecord(tableName string, db *Database) *Record {
	columnsTable := db.Schema.Tables["__columns__"]
	record := columnsTable.NewRecord()
	record.SetString("id", fmt.Sprintf("%d", column.Id))
	record.SetString("name", column.Name)
	record.SetString("table_name", tableName)
	record.SetString("type", TypeToName[column.Type])
	if column.ReferencesColumn != nil {
		record.SetString("references", column.ReferencesColumn.TableName)
	}
	return record
}

func ColumnFromRecord(record *Record) *Column {
	idInt, _ := strconv.Atoi(record.GetField("id").StringVal)
	return &Column{
		Id:   idInt,
		Name: record.GetField("name").StringVal,
		Type: NameToType[record.GetField("type").StringVal],
	}
}

func (table *Table) ToRecord(db *Database) *Record {
	record := db.Schema.Tables["__tables__"].NewRecord()
	record.SetString("name", table.Name)
	record.SetString("primary_key", table.PrimaryKey)
	return record
}

func TableFromRecord(record *Record) *Table {
	return &Table{
		Columns:    make([]*Column, 0),
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
		nextColumnIdBytes := sequencesBucket.Get([]byte("__next_column_id__"))
		if nextColumnIdBytes == nil {
			// write it
			nextColumnIdBytes = make([]byte, 4)
			binary.BigEndian.PutUint32(nextColumnIdBytes, uint32(db.Schema.NextColumnId))
			sequencesBucket.Put([]byte("__next_column_id__"), nextColumnIdBytes)
		} else {
			// read it
			nextColumnId := binary.BigEndian.Uint32(nextColumnIdBytes)
			db.Schema.NextColumnId = int(nextColumnId)
		}
		return nil
	})
}

func (db *Database) LoadUserSchema() {
	tablesTable := db.Schema.Tables["__tables__"]
	columnsTable := db.Schema.Tables["__columns__"]
	db.BoltDB.View(func(tx *bolt.Tx) error {
		tx.Bucket([]byte("__tables__")).ForEach(func(_ []byte, tableBytes []byte) error {
			tableRecord := tablesTable.RecordFromBytes(tableBytes)
			tableSpec := TableFromRecord(tableRecord)
			db.Schema.Tables[tableSpec.Name] = tableSpec
			return nil
		})
		tx.Bucket([]byte("__columns__")).ForEach(func(key []byte, columnBytes []byte) error {
			columnRecord := columnsTable.RecordFromBytes(columnBytes)
			columnSpec := ColumnFromRecord(columnRecord)
			tableSpec := db.Schema.Tables[columnRecord.GetField("table_name").StringVal]
			tableSpec.Columns = append(tableSpec.Columns, columnSpec)
			return nil
		})
		return nil
	})
}

func GetBuiltinSchema() *Schema {
	// these never go in the on-disk __tables__ and __columns__ Bolt buckets
	// doing ids like this is kind of precarious...
	tables := map[string]*Table{
		"__tables__": &Table{
			Name:       "__tables__",
			PrimaryKey: "name",
			Columns: []*Column{
				&Column{
					Id:   0,
					Name: "name",
					Type: TypeString,
				},
				&Column{
					Id:   1,
					Name: "primary_key",
					Type: TypeString,
				},
			},
		},
		"__columns__": &Table{
			Name:       "__columns__",
			PrimaryKey: "name",
			Columns: []*Column{
				&Column{
					Id:   2,
					Name: "id",
					Type: TypeString, // TODO: switch to int when they work
				},
				&Column{
					Id:   3,
					Name: "name",
					Type: TypeString,
				},
				&Column{
					Id:   4,
					Name: "table_name",
					Type: TypeString,
					ReferencesColumn: &ColumnReference{
						TableName: "__tables__",
					},
				},
				&Column{
					Id:   5,
					Name: "type",
					Type: TypeString,
				},
				&Column{
					Id:   6,
					Name: "references", // TODO: this is a keyword. rename to "references_table"
					Type: TypeString,
				},
			},
		},
	}
	return &Schema{
		Tables:       tables,
		NextColumnId: 7,
	}
}
