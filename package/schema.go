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
	Name             string
	Type             ColumnType
	ReferencesColumn *ColumnReference
}

type ColumnReference struct {
	TableName string // we're gonna assume for now that you can only reference the primary key
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
