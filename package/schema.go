package treesql

import sophia "github.com/pzhin/go-sophia"
import "bytes"
import "log"

type Schema struct {
	Tables map[string]*Table
}

type Table struct {
	Name       string
	Columns    []*Column
	PrimaryKey string
}

type Column struct {
	Name             string
	Type             ColumnType
	ReferencesColumn *ColumnReference
}

type ColumnReference struct {
	TableName string // we're gonna assume for now that you can only reference the primary key
}

type Record struct {
	Table  *Table
	Values []Value
}

type Value struct {
	// tagged union plz?
	Type      ColumnType
	StringVal string
	IntVal    int
}

// maybe I should use that iota weirdness
type ColumnType byte

const TypeString ColumnType = 0
const TypeInt ColumnType = 1

var ToSophiaType = map[ColumnType]sophia.FieldType{
	TypeString: sophia.FieldTypeString,
	TypeInt:    sophia.FieldTypeUInt32, // what about signed ints? lol
}

func (table *Table) ToSophiaSchema() *sophia.Schema {
	result := &sophia.Schema{}
	for _, column := range table.Columns {
		sophiaType := ToSophiaType[column.Type]
		if column.Name == table.PrimaryKey {
			result.AddKey(column.Name, sophiaType)
		} else {
			result.AddValue(column.Name, sophiaType)
		}
	}
	return result
}

func (table *Table) RecordFromBytes(raw []byte) *Record {
	record := &Record{
		Table:  table,
		Values: make([]Value, len(table.Columns)),
	}
	buffer := bytes.NewBuffer(raw)
	for valueIdx := 0; valueIdx < len(table.Columns); valueIdx++ {
		typeCode, _ := buffer.ReadByte()
		switch ColumnType(typeCode) {
		case TypeString:
			length, _ := readInteger(buffer)
			stringBytes := make([]byte, length)
			buffer.Read(stringBytes)
			record.Values[valueIdx] = Value{
				Type:      TypeString,
				StringVal: string(stringBytes),
			}
		case TypeInt:
			val, _ := readInteger(buffer)
			record.Values[valueIdx] = Value{
				Type:   TypeInt,
				IntVal: val,
			}
		}
	}
	return record
}

func (record *Record) GetField(name string) *Value {
	idx := -1
	for curIdx, column := range record.Table.Columns {
		if column.Name == name {
			idx = curIdx
			break
		}
	}
	if idx == -1 {
		log.Fatalln("field not found for table", record.Table.Name, ":", name)
	}
	return &record.Values[idx]
}

// seriosly, why am I writing this
func readInteger(buffer *bytes.Buffer) (int, error) {
	a, _ := buffer.ReadByte()
	b, _ := buffer.ReadByte()
	c, _ := buffer.ReadByte()
	d, _ := buffer.ReadByte()
	return ((int(a) << 24) | (int(b) << 16) | (int(c) << 8) | int(d)), nil
}

// func writeInteger(writer *bytes.Writer, int val) {

// }

func GetTestSchema() *Schema {
	tables := map[string]*Table{
		"__tables__": &Table{
			Name:       "__tables__",
			PrimaryKey: "name",
			Columns: []*Column{
				&Column{
					Name: "name",
					Type: TypeString,
				},
				&Column{
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
					Name: "name",
					Type: TypeString,
				},
				&Column{
					Name: "table_name",
					Type: TypeString,
					ReferencesColumn: &ColumnReference{
						TableName: "__tables__",
					},
				},
				&Column{
					Name: "references",
					Type: TypeString,
				},
			},
		},
		"blog_posts": &Table{
			Name:       "blog_posts",
			PrimaryKey: "id",
			Columns: []*Column{
				&Column{
					Name: "id",
					Type: TypeString,
				},
				&Column{
					Name: "author_id",
					Type: TypeString,
					ReferencesColumn: &ColumnReference{
						TableName: "users",
					},
				},
				&Column{
					Name: "title",
					Type: TypeString,
				},
				&Column{
					Name: "body",
					Type: TypeString,
				},
			},
		},
		"comments": &Table{
			Name:       "comments",
			PrimaryKey: "id",
			Columns: []*Column{
				&Column{
					Name: "id",
					Type: TypeString,
				},
				&Column{
					Name: "post_id",
					Type: TypeString,
					ReferencesColumn: &ColumnReference{
						TableName: "blog_posts",
					},
				},
				&Column{
					Name: "author_id",
					Type: TypeString,
					ReferencesColumn: &ColumnReference{
						TableName: "users",
					},
				},
				&Column{
					Name: "body",
					Type: TypeString,
				},
			},
		},
		"users": &Table{
			Name:       "users",
			PrimaryKey: "id",
			Columns: []*Column{
				&Column{
					Name: "id",
					Type: TypeString,
				},
				&Column{
					Name: "name",
					Type: TypeString,
				},
			},
		},
	}
	return &Schema{
		Tables: tables,
	}
}
