package treesql

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/vilterp/treesql/pkg/lang"
)

type txn struct {
	boltTxn *bolt.Tx
	db      *Database
}

var builtinsScope *lang.Scope
var builtinsTypeScope *lang.TypeScope

func init() {
	builtinsScope = lang.BuiltinsScope.NewChildScope()

	// TODO: how are these gonna get access to the DB to add listeners?
	builtinsScope.AddMap(map[string]lang.Value{
		// TODO: move this somewhere where it has access to the database...
		"get": &lang.VBuiltin{
			Name:    "get",
			RetType: lang.NewTVar("V"),
			Params: []lang.Param{
				{
					Name: "index",
					Typ:  lang.NewTIndex(lang.NewTVar("K"), lang.NewTVar("V")),
				},
				{
					Name: "value",
					Typ:  lang.NewTVar("K"),
				},
			},
			Impl: func(interp lang.Caller, args []lang.Value) (lang.Value, error) {
				return lang.NewVInt(42), nil
			},
		},
		"addInsertListener": &lang.VBuiltin{
			Name:    "addInsertListener",
			RetType: lang.TInt,
			Params: []lang.Param{
				{
					Name: "index",
					Typ:  lang.NewTIndex(lang.NewTVar("K"), lang.NewTVar("V")),
				},
				{
					Name: "selection",
					Typ: lang.NewTFunction(
						[]lang.Param{
							{
								Name: "row",
								// TODO: not sure this will always be the same as the index type
								Typ: lang.NewTVar("V"),
							},
						},
						lang.NewTVar("S"),
					),
				},
			},
			Impl: func(interp lang.Caller, args []lang.Value) (lang.Value, error) {
				fmt.Println("addFilteredTableListener", args)
				return lang.NewVInt(42), nil
			},
		},
		"addUpdateListener": &lang.VBuiltin{
			Name: "addUpdateListener",
			Params: []lang.Param{
				{
					Name: "index",
					Typ:  lang.NewTIndex(lang.NewTVar("K"), lang.NewTVar("V")),
				},
				{
					Name: "pk",
					Typ:  lang.NewTVar("K"),
				},
				{
					Name: "selection",
					Typ: lang.NewTFunction(
						[]lang.Param{
							{
								Name: "row",
								// TODO: not sure this will always be the same as the index type
								Typ: lang.NewTVar("V"),
							},
						},
						lang.NewTVar("S"),
					),
				},
			},
			RetType: lang.TInt,
			Impl: func(interp lang.Caller, args []lang.Value) (lang.Value, error) {
				return lang.NewVInt(42), nil
			},
		},
	})

	builtinsTypeScope = builtinsScope.ToTypeScope()
}

func (s *schema) toScope(txn *txn) (*lang.Scope, *lang.TypeScope) {
	// TODO: grab schema mutex here
	// also, only do this when the schema changes
	newScope := builtinsScope.NewChildScope()
	newTypeScope := builtinsTypeScope.NewChildScope()
	tables := map[string]lang.Value{}
	for _, table := range s.tables {
		if table.isBuiltin {
			continue
		}
		tables[table.name] = table.toRecordOfIndices(txn)
	}
	tablesRec := lang.NewVRecord(tables)
	newScope.Add("tables", tablesRec)
	newTypeScope.Add("tables", tablesRec.GetType())

	return newScope, newTypeScope
}

func (table *tableDescriptor) toRecordOfIndices(txn *txn) *lang.VRecord {
	attrs := map[string]lang.Value{}

	for _, col := range table.columns {
		if col.name == table.primaryKey {
			// Construct VIndex to return.
			attrs[col.name] = lang.NewVIndex(
				table.getPKType(),
				table.getType(),
				col.name,
				func(colName string) (lang.Iterator, error) {
					return txn.getTableIterator(table, colName)
				},
			)
		} else if col.referencesColumn != nil {
			// e.g. comments.blog_post_id: Index<BlogPostID, Index<CommentID, CommentID>>
			// TODO: just map to unit I guess
			attrs[col.name] = lang.NewVIndex(
				col.typ,
				lang.NewTIndex(
					col.typ,
					lang.NewTIndex(
						table.getPKType(),
						table.getPKType(),
					),
				),
				col.name,
				func(colName string) (lang.Iterator, error) {
					return nil, nil
				},
			)
		}
	}

	return lang.NewVRecord(attrs)
}

// TODO: maybe name BoltIterator
// once there are also virtual table iterators
type tableIterator struct {
	cursor        *bolt.Cursor
	table         *tableDescriptor
	seekedToFirst bool
}

var _ lang.Iterator = &tableIterator{}

func (ti *tableIterator) Next(_ lang.Caller) (lang.Value, error) {
	var key []byte
	var value []byte
	if !ti.seekedToFirst {
		key, value = ti.cursor.First()
		ti.seekedToFirst = true
	} else {
		key, value = ti.cursor.Next()
	}
	if key == nil {
		return nil, lang.EndOfIteration
	}
	record, err := lang.Decode(value)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (ti *tableIterator) Close() error {
	// surprisingly, bolt.Cursor doesn't have a .Close()
	return nil
}

func (txn *txn) getTableIterator(table *tableDescriptor, colName string) (*tableIterator, error) {
	colID, err := table.colIDForName(colName)

	if err != nil {
		return nil, err
	}
	tableBucket := txn.boltTxn.Bucket([]byte(table.name))
	if tableBucket == nil {
		return nil, fmt.Errorf("bucket doesn't exist: %s", table.name)
	}
	idxBucket := tableBucket.Bucket(lang.EncodeInteger(int32(colID)))
	if idxBucket == nil {
		return nil, fmt.Errorf("bucket doesn't exist: %s/%d", table.name, colID)
	}

	cursor := idxBucket.Cursor()
	//cursor.
	return &tableIterator{
		table:  table,
		cursor: cursor,
	}, nil
}
