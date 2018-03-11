package treesql

import "github.com/vilterp/treesql/package/lang"

func (db *Database) AddBuiltinSchema() {
	// these never go in the on-disk __tables__ and __columns__ Bolt buckets
	// doing ids like this is kind of precarious...

	// TODO: could use string CREATE TABLEs here
	// parse 'em; call buildTableDescriptor to assign ids
	db.addTableDescriptor(&TableDescriptor{
		Name:       "__tables__",
		PrimaryKey: "name",
		Columns: []*ColumnDescriptor{
			{
				ID:   0,
				Name: "name",
				Type: lang.TString,
			},
			{
				ID:   1,
				Name: "primary_key",
				Type: lang.TString,
			},
		},
		IsBuiltin: true,
	})
	db.addTableDescriptor(&TableDescriptor{
		Name:       "__columns__",
		PrimaryKey: "id",
		Columns: []*ColumnDescriptor{
			{
				ID:   2,
				Name: "id",
				Type: lang.TString, // TODO: switch to int when they work
			},
			{
				ID:   3,
				Name: "name",
				Type: lang.TString,
			},
			{
				ID:   4,
				Name: "table_name",
				Type: lang.TString,
				ReferencesColumn: &ColumnReference{
					TableName: "__tables__",
				},
			},
			{
				ID:   5,
				Name: "type",
				Type: lang.TString,
			},
			{
				ID:   6,
				Name: "references", // TODO: this is a keyword. rename to "references_table"
				Type: lang.TString,
			},
		},
		IsBuiltin: true,
	})
	db.addTableDescriptor(&TableDescriptor{
		Name:       "__record_listeners__",
		PrimaryKey: "id",
		Columns: []*ColumnDescriptor{
			{
				ID:   7,
				Name: "id",
				Type: lang.TString,
			},
			{
				ID:   8,
				Name: "connection_id",
				Type: lang.TString,
			},
			{
				ID:   9,
				Name: "channel_id",
				Type: lang.TString,
			},
			{
				ID:   10,
				Name: "table_name",
				Type: lang.TString,
				ReferencesColumn: &ColumnReference{
					TableName: "__tables__",
				},
			},
			{
				ID:   11,
				Name: "pk_value",
				Type: lang.TString,
			},
			{
				ID:   12,
				Name: "query_path",
				Type: lang.TString,
			},
		},
		IsBuiltin: true,
	})
	db.Schema.NextColumnID = 13 // ugh magic numbers.
}

// TODO: __connections__, __channels__, __whole_table_listeners__, __filtered_table_listeners__
