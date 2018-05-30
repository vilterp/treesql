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
	// TODO: move this to planner -- generate & run FP instead of doing it here.
	values := make(map[string]lang.Value)
	for idx, value := range insert.Values {
		col := table.columns[idx]
		values[col.name] = lang.NewVString(value)
	}
	record := lang.NewVRecord(values)
	key := values[table.primaryKey]

	keyBytes, err := lang.Encode(key)
	if err != nil {
		return err
	}

	recordBytes, err := lang.Encode(record)
	if err != nil {
		return err
	}

	// Write to table.
	if err := conn.database.boltDB.Update(func(tx *bolt.Tx) error {
		tableBucket := tx.Bucket([]byte(insert.Table))

		// Write to primary index.
		primaryIndexBucket := tableBucket.Bucket(table.pkBucketKey())

		if current := primaryIndexBucket.Get(keyBytes); current != nil {
			return &recordAlreadyExists{ColName: table.primaryKey, Val: key}
		}
		if err := primaryIndexBucket.Put(keyBytes, recordBytes); err != nil {
			return err
		}

		// Write to secondary indices.
		for _, col := range table.columns {
			if col.referencesColumn != nil {
				indexBucket, err := getIndexBucket(tx, table, col)
				if err != nil {
					return err
				}

				valueForColumn := record.GetValue(col.name)
				encodedValueForColumn, err := lang.Encode(valueForColumn)
				if err != nil {
					return err
				}

				subIndexBucket, err := indexBucket.CreateBucketIfNotExists(encodedValueForColumn)
				if err != nil {
					return err
				}

				// TODO: don't put the key twice. Will require sorting some things out on the
				// query codegen side.
				if err := subIndexBucket.Put(keyBytes, keyBytes); err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		return errors.Wrap(err, "executing insert")
	}

	// Push to live query listeners.
	conn.database.pushTableEvent(channel, insert.Table, nil, recordBytes)
	// Return ack.
	channel.writeAckMessage("INSERT 1")

	// record latency.
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	conn.database.metrics.insertLatency.Observe(float64(duration.Nanoseconds()))
	// clog.Println(channel, "handled insert in", duration)
	return nil
}
