package treesql

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/vilterp/treesql/pkg/lang"
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

	// Create VRecord
	// TODO: move this to planner
	values := make(map[string]lang.Value)
	for idx, value := range insert.Values {
		col := table.columns[idx]
		values[col.name] = lang.NewVString(value)
	}
	record := lang.NewVRecord(values)
	key := values[table.primaryKey]

	// Write to table.
	err := conn.database.boltDB.Update(func(tx *bolt.Tx) error {
		tableBucket := tx.Bucket([]byte(insert.Table))
		primaryIndexBucket := tableBucket.Bucket(table.pkBucketKey())

		keyBytes, err := lang.Encode(key)
		if err != nil {
			return err
		}

		if current := primaryIndexBucket.Get(keyBytes); current != nil {
			return &recordAlreadyExists{ColName: table.primaryKey, Val: key}
		}
		recordBytes, err := lang.Encode(record)
		if err != nil {
			return err
		}
		return primaryIndexBucket.Put(keyBytes, recordBytes)
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
