package treesql

import "github.com/vilterp/treesql/pkg/lang"

func (db *Database) addBuiltinSchema() {
	// these never go in the on-disk __tables__ and __columns__ Bolt buckets
	// doing ids like this is kind of precarious...

	// TODO: could use string CREATE TABLEs here
	// parse 'em; call buildTableDescriptor to assign ids
	db.addTableDescriptor(&tableDescriptor{
		name:       "__tables__",
		primaryKey: "name",
		columns: []*columnDescriptor{
			{
				id:   0,
				name: "name",
				typ:  lang.TString,
			},
			{
				id:   1,
				name: "primary_key",
				typ:  lang.TString,
			},
		},
		isBuiltin: true,
	})
	db.addTableDescriptor(&tableDescriptor{
		name:       "__columns__",
		primaryKey: "id",
		columns: []*columnDescriptor{
			{
				id:   2,
				name: "id",
				typ:  lang.TString, // TODO: switch to int when they work
			},
			{
				id:   3,
				name: "name",
				typ:  lang.TString,
			},
			{
				id:   4,
				name: "table_name",
				typ:  lang.TString,
				referencesColumn: &columnReference{
					tableName: "__tables__",
				},
			},
			{
				id:   5,
				name: "type",
				typ:  lang.TString,
			},
			{
				id:   6,
				name: "references", // TODO: this is a keyword. rename to "references_table"
				typ:  lang.TString,
			},
		},
		isBuiltin: true,
	})
	db.addTableDescriptor(&tableDescriptor{
		name:       "__record_listeners__",
		primaryKey: "id",
		columns: []*columnDescriptor{
			{
				id:   7,
				name: "id",
				typ:  lang.TString,
			},
			{
				id:   8,
				name: "connection_id",
				typ:  lang.TString,
			},
			{
				id:   9,
				name: "channel_id",
				typ:  lang.TString,
			},
			{
				id:   10,
				name: "table_name",
				typ:  lang.TString,
				referencesColumn: &columnReference{
					tableName: "__tables__",
				},
			},
			{
				id:   11,
				name: "pk_value",
				typ:  lang.TString,
			},
			{
				id:   12,
				name: "query_path",
				typ:  lang.TString,
			},
		},
		isBuiltin: true,
	})
	db.schema.nextColumnID = 13 // ugh magic numbers.
}

// TODO: __connections__, __channels__, __whole_table_listeners__, __filtered_table_listeners__
