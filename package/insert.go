package treesql

import (
	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	clog "github.com/vilterp/treesql/package/log"
)

func (db *Database) validateInsert(insert *Insert) error {
	// does table exist
	tableSpec, ok := db.Schema.Tables[insert.Table]
	if !ok {
		return &NoSuchTable{TableName: insert.Table}
	}
	// can't insert into builtins
	if insert.Table == "__tables__" || insert.Table == "__columns__" {
		return &BuiltinWriteAttempt{TableName: insert.Table}
	}
	// right # fields (TODO: validate types)
	wanted := len(tableSpec.Columns)
	got := len(insert.Values)
	if wanted != got {
		return &InsertWrongNumFields{TableName: insert.Table, Wanted: wanted, Got: got}
	}
	return nil
}

func (conn *Connection) ExecuteInsert(insert *Insert, channel *Channel) error {
	table := conn.Database.Schema.Tables[insert.Table]
	record := table.NewRecord()
	for idx, value := range insert.Values {
		record.SetString(table.Columns[idx].Name, value)
	}
	key := record.GetField(table.PrimaryKey).StringVal
	// write to table
	err := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(insert.Table))
		if current := bucket.Get([]byte(key)); current != nil {
			return &RecordAlreadyExists{ColName: table.PrimaryKey, Val: key}
		}
		return bucket.Put([]byte(key), record.ToBytes())
	})
	if err != nil {
		return errors.Wrap(err, "executing insert")
	}
	// push to live query listeners
	conn.Database.PushTableEvent(channel, insert.Table, nil, record)
	clog.Println(channel, "handled insert")
	channel.WriteAckMessage("INSERT 1")
	return nil
}
