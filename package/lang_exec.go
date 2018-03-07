package treesql

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/vilterp/treesql/package/lang"
)

type Txn struct {
	boltTxn *bolt.Tx
	db      *Database
}

func (s *Schema) toScope(txn *Txn) *lang.Scope {
	newScope := lang.NewScope(lang.BuiltinsScope)
	for _, table := range s.Tables {
		if table.IsBuiltin {
			continue
		}
		newScope.Add(table.Name, table.toVObject(txn))
	}
	return newScope
}

func (table *TableDescriptor) toVObject(txn *Txn) *lang.VObject {
	attrs := map[string]lang.Value{}

	for _, col := range table.Columns {
		if col.Name == table.PrimaryKey {
			iter, err := txn.getTableIterator(table, col.Name)
			if err != nil {
				panic(fmt.Sprintf("err getting table iterator: %v", err))
			}
			attrs[col.Name] = lang.NewVObject(map[string]lang.Value{
				"scan": lang.NewVIteratorRef(iter, lang.TInt),
				"get":  lang.NewVInt(2), // getter
			})
		}
	}

	return lang.NewVObject(attrs)
}

// TODO: maybe name BoltIterator
// once there are also virtual table iterators
type tableIterator struct {
	cursor        *bolt.Cursor
	table         *TableDescriptor
	seekedToFirst bool
}

var _ lang.Iterator = &tableIterator{}

func (ti *tableIterator) Next() (lang.Value, error) {
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
	// TODO: actually deserialize
	//record := ti.table.RecordFromBytes(value)
	return lang.NewVInt(len(value)), nil
}

func (ti *tableIterator) Close() error {
	// surprisingly, bolt.Cursor doesn't have a .Close()
	return nil
}

func (txn *Txn) getTableIterator(table *TableDescriptor, colName string) (*tableIterator, error) {
	colID, err := table.colIDForName(colName)
	if err != nil {
		return nil, err
	}
	tableBucket := txn.boltTxn.Bucket([]byte(table.Name))
	if tableBucket == nil {
		return nil, fmt.Errorf("bucket doesn't exist: %s", table.Name)
	}
	idxBucket := tableBucket.Bucket(encodeInteger(int32(colID)))
	if idxBucket == nil {
		return nil, fmt.Errorf("bucket doesn't exist: %s/%d", table.Name, colID)
	}

	cursor := idxBucket.Cursor()
	//cursor.
	return &tableIterator{
		table:  table,
		cursor: cursor,
	}, nil
}

// TODO: build an vIteratorRef with the right type
// may require using the Type type in the table descriptor
// which would really f*ck things up
