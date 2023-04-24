package treesql

import (
	"time"

	"encoding/binary"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

func (db *Database) validateInsert(insert *Insert) error {
	// does table exist
	tableSpec, ok := db.schema.tables[insert.Table]
	if !ok {
		return &noSuchTable{TableName: insert.Table}
	}
	// can't insert into builtins
	if insert.Table == "__tables__" || insert.Table == "__columns__" {
		return &builtinWriteAttempt{TableName: insert.Table}
	}
	// right # fields (TODO: validate types)
	wanted := len(tableSpec.columns)
	got := len(insert.Values)
	if wanted != got {
		return &insertWrongNumFields{TableName: insert.Table, Wanted: wanted, Got: got}
	}
	return nil
}

func (conn *connection) executeInsert(insert *Insert, channel *channel) error {
	startTime := time.Now()
	table := conn.database.schema.tables[insert.Table]

	// Create record.
	record := table.NewRecord()
	for idx, value := range insert.Values {
		record.setString(table.columns[idx].name, value)
	}
	key := record.GetField(table.primaryKey).stringVal

	// Find id of PK column.
	var pkID int
	for _, col := range table.columns {
		if col.name == table.primaryKey {
			pkID = col.id
			break
		}
	}

	// Write to table.
	err := conn.database.boltDB.Update(func(tx *bolt.Tx) error {
		tableBucket := tx.Bucket([]byte(insert.Table))

		// TODO: factor this out to an encoding file
		pkIDBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(pkIDBytes, uint32(pkID))

		primaryIndexBucket := tableBucket.Bucket(pkIDBytes)
		if current := primaryIndexBucket.Get([]byte(key)); current != nil {
			return &recordAlreadyExists{ColName: table.primaryKey, Val: key}
		}
		recordBytes, err := record.ToBytes()
		if err != nil {
			return err
		}
		return primaryIndexBucket.Put([]byte(key), recordBytes)
	})
	if err != nil {
		return errors.Wrap(err, "executing insert")
	}

	// Push to live query listeners.
	conn.database.pushTableEvent(channel, insert.Table, nil, record)
	// Return ack.
	channel.writeAckMessage("INSERT 1")

	// record latency.
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	conn.database.metrics.insertLatency.Observe(float64(duration.Nanoseconds()))
	// clog.Println(channel, "handled insert in", duration)
	return nil
}
