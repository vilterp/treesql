package treesql

import (
	"github.com/boltdb/bolt"
	"github.com/vilterp/treesql/package/lang"
)

type Txn struct {
	boltTxn bolt.Tx
	db      *Database
}

func (s *Schema) toScope(txn *Txn) *lang.Scope {
	newScope := lang.NewScope(lang.BuiltinsScope)
	for _, table := range s.Tables {
		newScope.Add(table.Name, table.toVObject(txn))
	}
	return newScope
}

func (table *TableDescriptor) toVObject(txn *Txn) *lang.VObject {
	attrs := map[string]lang.Value{}

	for _, col := range table.Columns {
		if col.Name == table.PrimaryKey {
			attrs[col.Name] = lang.NewVObject(map[string]lang.Value{
				"scan": lang.NewVInt(2), // iterator
				"get":  lang.NewVInt(2), // getter
			})
		}
	}

	return lang.NewVObject(attrs)
}

// TODO: maybe name BoltIterator
// once there are also virtual table iterators
type tableIterator struct {
	cursor *bolt.Cursor
	table  *TableDescriptor
}

var _ lang.Iterator = &tableIterator{}

func (txn *Txn) getTableIterator(table *TableDescriptor, colName string) (*tableIterator, error) {
	colID, err := table.colIDForName(colName)
	if err != nil {
		return nil, err
	}
	tableBucket := txn.boltTxn.Bucket([]byte(table.Name))
	idxBucket := tableBucket.Bucket(encodeInteger(int32(colID)))

	return &tableIterator{
		table:  table,
		cursor: idxBucket.Cursor(),
	}, nil
}

// TODO: build an vIteratorRef with the right type
// may require using the Type type in the table descriptor
// which would really f*ck things up
