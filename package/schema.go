package treesql

import sophia "github.com/pzhin/go-sophia"

type Schema struct {
	Tables map[string]*Table
}

type Table struct {
	Name       string
	Columns    []*Column
	PrimaryKey string
}

type Column struct {
	Name string
	Type ColumnType
}

type ColumnType byte

// maybe I should use that iota weirdness
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

func GetTestSchema() *Schema {
	tables := map[string]*Table{
		"blog_posts": &Table{
			Name:       "blog_posts",
			PrimaryKey: "id",
			Columns: []*Column{
				&Column{
					Name: "id",
					Type: TypeString,
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
				},
				&Column{
					Name: "body",
					Type: TypeString,
				},
			},
		},
	}
	return &Schema{
		Tables: tables,
	}
}
