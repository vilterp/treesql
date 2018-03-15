package treesql

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	clog "github.com/vilterp/treesql/pkg/log"
)

func (db *Database) validateUpdate(update *Update) error {
	table, ok := db.schema.tables[update.Table]
	// table exists
	if !ok {
		return &noSuchTable{
			TableName: update.Table,
		}
	}
	// table isn't a builtin
	if update.Table == "__tables__" || update.Table == "__columns__" {
		return &builtinWriteAttempt{
			TableName: update.Table,
		}
	}
	// column to update exists
	updateColExists := false
	for _, column := range table.columns {
		if column.name == update.ColumnName {
			updateColExists = true
		}
	}
	if !updateColExists {
		return &noSuchColumn{
			TableName:  update.Table,
			ColumnName: update.ColumnName,
		}
	}
	// column in where clause exists
	whereColExists := false
	for _, column := range table.columns {
		if column.name == update.WhereColumnName {
			whereColExists = true
		}
	}
	if !whereColExists {
		return &noSuchColumn{
			TableName:  update.Table,
			ColumnName: update.ColumnName,
		}
	}
	return nil
}

func (conn *connection) executeUpdate(update *Update, channel *channel) error {
	startTime := time.Now()

	// Write to table.
	table := conn.database.schema.tables[update.Table]
	rowsUpdated := 0
	updateErr := conn.database.boltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(update.Table))
		bucket.ForEach(func(key []byte, value []byte) error {
			record := table.RecordFromBytes(value)
			if record.GetField(update.WhereColumnName).stringVal == update.EqualsValue {
				// Clone and update old record.
				clonedOldRecord := record.Clone()
				record.setString(update.ColumnName, update.Value)
				// Clone new record
				recordBytes, err := record.ToBytes()
				if err != nil {
					return err
				}
				if err := bucket.Put(key, recordBytes); err != nil {
					return err
				}
				// Send live query updates.
				clonedNewRecord := record.Clone()
				conn.database.pushTableEvent(channel, update.Table, clonedOldRecord, clonedNewRecord)
				rowsUpdated++
			}
			return nil
		})
		return nil
	})
	if updateErr != nil {
		return errors.Wrap(updateErr, "executing update")
	}

	// Return ack message.
	channel.writeAckMessage(fmt.Sprintf("UPDATE %d", rowsUpdated))

	// record latency.
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	conn.database.metrics.updateLatency.Observe(float64(duration.Nanoseconds()))
	clog.Println(channel, "handled update in", duration)
	return nil
}
