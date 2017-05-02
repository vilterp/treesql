package treesql

import (
	"fmt"

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

var TypeNames = map[ColumnType]string{
	TypeString: "string",
	TypeInt:    "int",
}

func (column *Column) ToRecord(tableName string, db *Database) *Record {
	columnsTable := db.Schema.Tables["__columns__"]
	record := columnsTable.NewRecord()
	record.SetString("id", fmt.Sprintf("%d", column.Id))
	record.SetString("name", column.Name)
	record.SetString("table_name", tableName)
	record.SetString("type", TypeNames[column.Type])
	if column.ReferencesColumn != nil {
		record.SetString("references", column.ReferencesColumn.TableName)
	}
	return record
}

func ColumnFromRecord(record *Record) *Column {
	return nil
}

func (table *Table) ToRecord(db *Database) *Record {
	record := db.Schema.Tables["__tables__"].NewRecord()
	record.SetString("name", table.Name)
	record.SetString("primary_key", table.PrimaryKey)
	return record
}

func TableFromRecord(record *Record) *Table {
	return nil
}

func (db *Database) EnsureBuiltinSchema() {
	db.BoltDB.Update(func(tx *bolt.Tx) error {
		fmt.Println("TODO: create and populate __tables__ and __columns__ buckets")
		fmt.Println("TODO: create __sequences__ bucket")
		tx.CreateBucketIfNotExists([]byte("__sequences__")) // TODO: if it didn't exist, write to it
		return nil
	})
}

func GetBuiltinSchema() *Schema {
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
