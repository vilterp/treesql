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

	for idx := range table.columns {
		// By declaring `col` inside of the loop, we can use it in closures.
		col := table.columns[idx]
		if col.name == table.primaryKey {
			// Construct VIndex to return.
			attrs[col.name] = lang.NewVIndex(
				table.getPKType(),
				table.getType(),
				func() (lang.Iterator, error) {
					return txn.getIndexIterator(table, col)
				},
				func(key lang.Value) (lang.Value, error) {
					return txn.getValue(table, col, key)
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
				func() (lang.Iterator, error) {
					panic("TODO: implement scan on non-unique indices")
				},
				func(key lang.Value) (lang.Value, error) {
					return txn.getSubIndex(table, col, key)
				},
			)
		}
	}

	return lang.NewVRecord(attrs)
}

// TODO: maybe name BoltIterator
// once there are also virtual table iterators
type indexIterator struct {
	cursor        *bolt.Cursor
	seekedToFirst bool
}

var _ lang.Iterator = &indexIterator{}

func (ti *indexIterator) Next(_ lang.Caller) (lang.Value, error) {
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

func (ti *indexIterator) Close() error {
	// surprisingly, bolt.Cursor doesn't have a .Close()
	return nil
}

func (txn *txn) getIndexIterator(
	table *tableDescriptor, col *columnDescriptor,
) (*indexIterator, error) {
	idxBucket, err := txn.getIndexBucket(table, col)
	if err != nil {
		return nil, err
	}

	cursor := idxBucket.Cursor()
	return &indexIterator{
		cursor: cursor,
	}, nil
}

func (txn *txn) getValue(
	table *tableDescriptor, col *columnDescriptor, key lang.Value,
) (lang.Value, error) {
	idxBucket, err := txn.getIndexBucket(table, col)
	if err != nil {
		return nil, err
	}

	res := idxBucket.Get(lang.MustEncode(key))
	return lang.Decode(res)
}

func (txn *txn) getSubIndex(
	table *tableDescriptor, col *columnDescriptor, key lang.Value,
) (*lang.VIndex, error) {
	idxBucket, err := txn.getIndexBucket(table, col)
	if err != nil {
		return nil, err
	}

	return lang.NewVIndex(
		col.typ,
		col.typ,
		func() (lang.Iterator, error) {
			fmt.Println("gave out cursor for", table.name, col.name)
			cursor := idxBucket.Cursor()
			return &indexIterator{
				cursor: cursor,
			}, nil
		},
		func(key lang.Value) (lang.Value, error) {
			panic("TODO: implement get on sub index")
		},
	), nil
}

func (txn *txn) getIndexBucket(
	table *tableDescriptor, col *columnDescriptor,
) (*bolt.Bucket, error) {
	colID, err := table.colIDForName(col.name)

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

	return idxBucket, nil
}
