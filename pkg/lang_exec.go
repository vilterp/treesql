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
		"addFilteredTableListener": &lang.VBuiltin{
			Name:    "addFilteredTableListener",
			RetType: lang.TInt,
			Params: []lang.Param{
				{
					Name: "table",
					Typ:  lang.TString,
				},
				{
					Name: "column",
					Typ:  lang.TString,
				},
				{
					Name: "value",
					Typ:  lang.NewTVar("V"),
				},
			},
			Impl: func(interp lang.Caller, args []lang.Value) (lang.Value, error) {
				fmt.Println("addFilteredTableListener", args)
				return lang.NewVInt(42), nil
			},
		},
		"addWholeTableListener": &lang.VBuiltin{
			Name: "addWholeTableListener",
			Params: []lang.Param{
				{
					Name: "table",
					Typ:  lang.TString,
				},
				// TODO: expr
			},
			RetType: lang.TInt,
			Impl: func(interp lang.Caller, args []lang.Value) (lang.Value, error) {
				fmt.Println("addWholeTableListener", args)
				return lang.NewVInt(42), nil
			},
		},
		"addRecordListener": &lang.VBuiltin{
			Name: "addRecordListener",
			Params: []lang.Param{
				{
					Name: "table",
					Typ:  lang.TString,
				},
				{
					Name: "pk",
					Typ:  lang.NewTVar("V"),
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
		tables[table.name] = table.toVRecord(txn)
	}
	tablesRec := lang.NewVRecord(tables)
	newScope.Add("tables", tablesRec)
	newTypeScope.Add("tables", tablesRec.GetType())

	return newScope, newTypeScope
}

func (table *tableDescriptor) toVRecord(txn *txn) *lang.VRecord {
	attrs := map[string]lang.Value{}

	for _, col := range table.columns {
		if col.name == table.primaryKey {
			// Construct VIndex to return.
			attrs[col.name] = lang.NewVIndex(
				table.getType(),
				col.name,
				func(colName string) (lang.Iterator, error) {
					return txn.getTableIterator(table, colName)
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
