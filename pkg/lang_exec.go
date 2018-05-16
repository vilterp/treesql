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

func (s *schema) toScope(txn *txn) (*lang.Scope, *lang.TypeScope) {
	// TODO: grab schema mutex here
	// also, only do this when the schema changes
	newScope := lang.BuiltinsScope.NewChildScope()
	newTypeScope := lang.BuiltinsTypeScope.NewChildScope()
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
					table.getPKType(),
					table.getPKType(),
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
