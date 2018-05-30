package treesql

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/vilterp/treesql/pkg/lang"
)

type txn struct {
	boltTxn *bolt.Tx
	channel *channel
}

func newTxn(boltTx *bolt.Tx, channel *channel) *txn {
	return &txn{
		channel: channel,
		boltTxn: boltTx,
	}
}

func (s *schema) toSchemaIndexMap(txn *txn) SchemaIndexMap {
	// TODO: grab schema mutex here
	// also, only do this when the schema changes
	tables := map[string]map[string]*lang.VIndex{}
	for _, table := range s.tables {
		if table.isBuiltin {
			continue
		}
		tables[table.name] = table.toIndexMap(txn)
	}

	return tables
}

func (table *tableDescriptor) toIndexMap(txn *txn) map[string]*lang.VIndex {
	indices := map[string]*lang.VIndex{}

	for idx := range table.columns {
		// By declaring `col` inside of the loop, we can use it in closures.
		col := table.columns[idx]
		if col.name == table.primaryKey {
			// Construct unique primary index to return.
			indices[col.name] = lang.NewVIndex(
				table.getPKType(),
				table.getType(),
				func() (lang.Iterator, error) {
					return txn.getIndexIterator(table, col)
				},
				func(key lang.Value) (lang.Value, error) {
					return txn.getValue(table, col, key)
				},
				func(lambda lang.VFunction) {
					lqi := table.liveQueryInfo
					lqi.mu.Lock()
					defer lqi.mu.Unlock()

					lqi.tableSubscriptionEvents <- &tableSubscriptionEvent{
						channel:  txn.channel,
						SubQuery: &Select{}, // TODO: put query in here
					}

					fmt.Println("hello from addInsertListener on primary index")
				},
			)
		} else if col.referencesColumn != nil {
			// Construct non-unique index to return.
			// e.g. comments.blog_post_id: Index<BlogPostID, Index<CommentID, CommentID>>
			// TODO: just map to unit I guess
			indices[col.name] = lang.NewVIndex(
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
				func(lambda lang.VFunction) {
					fmt.Println("hello from addInsertListener on secondary index")
				},
			)
		}
	}

	return indices
}

// TODO: maybe name BoltIterator
// once there are also virtual table iterators
type indexIterator struct {
	cursor        *bolt.Cursor
	typ           lang.DecodableType
	seekedToFirst bool
}

var _ lang.Iterator = &indexIterator{}

func newIndexIterator(cursor *bolt.Cursor, typ lang.DecodableType) *indexIterator {
	return &indexIterator{
		cursor: cursor,
		typ:    typ,
	}
}

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

	return lang.Decode(ti.typ, value)
}

func (ti *indexIterator) Close() error {
	// surprisingly, bolt.Cursor doesn't have a .Close()
	return nil
}

func (txn *txn) getIndexIterator(
	table *tableDescriptor, col *columnDescriptor,
) (*indexIterator, error) {
	idxBucket, err := getIndexBucket(txn.boltTxn, table, col)
	if err != nil {
		return nil, err
	}

	// Choose type based on whether this is the primary index or not.
	var typ lang.DecodableType
	if col.name == table.primaryKey {
		typ = table.getType()
	} else {
		typ = col.typ
	}

	decodableType, ok := typ.(lang.DecodableType)
	if !ok {
		panic(fmt.Sprintf("not a decodable type: %s", col.typ.Format()))
	}

	cursor := idxBucket.Cursor()
	return newIndexIterator(cursor, decodableType), nil
}

func (txn *txn) getValue(
	table *tableDescriptor, col *columnDescriptor, key lang.Value,
) (lang.Value, error) {
	idxBucket, err := getIndexBucket(txn.boltTxn, table, col)
	if err != nil {
		return nil, err
	}

	res := idxBucket.Get(lang.MustEncode(key))

	var typ lang.DecodableType
	if col.name == table.primaryKey {
		typ = table.getType()
	} else {
		typ = col.typ
	}

	return lang.Decode(typ, res)
}

func (txn *txn) getSubIndex(
	table *tableDescriptor, col *columnDescriptor, key lang.Value,
) (*lang.VIndex, error) {
	// Get index.
	idxBucket, err := getIndexBucket(txn.boltTxn, table, col)
	if err != nil {
		return nil, err
	}

	// Get sub-index.
	subIdxBucket := idxBucket.Bucket(lang.MustEncode(key))
	if subIdxBucket == nil {
		// TODO: probably not an error for this not to exist.
		return nil, fmt.Errorf("sub-index `%s/%s/%s` doesn't exist", table.name, col.name, key.Format())
	}

	return lang.NewVIndex(
		col.typ,
		col.typ,
		func() (lang.Iterator, error) {
			cursor := subIdxBucket.Cursor()
			return newIndexIterator(cursor, col.typ), nil
		},
		func(key lang.Value) (lang.Value, error) {
			panic("TODO: implement get on sub index")
		},
		func(lambda lang.VFunction) {
			fmt.Println("hello from addInsertistener on sub index")
			lqi := table.liveQueryInfo
			lqi.mu.Lock()
			defer lqi.mu.Unlock()

			lqi.tableSubscriptionEvents <- &tableSubscriptionEvent{
				ColumnName: &col.name,
				Value:      key,
			}
		},
	), nil
}

// TODO: put this back on txn...
func getIndexBucket(
	txn *bolt.Tx, table *tableDescriptor, col *columnDescriptor,
) (*bolt.Bucket, error) {
	colID, err := table.colIDForName(col.name)

	if err != nil {
		return nil, err
	}
	tableBucket := txn.Bucket([]byte(table.name))
	if tableBucket == nil {
		return nil, fmt.Errorf("bucket doesn't exist: %s", table.name)
	}
	idxBucket := tableBucket.Bucket(lang.EncodeInteger(int32(colID)))
	if idxBucket == nil {
		return nil, fmt.Errorf("bucket doesn't exist: %s/%d", table.name, colID)
	}

	return idxBucket, nil
}
