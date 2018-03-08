package treesql

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	clog "github.com/vilterp/treesql/pkg/log"
)

func (db *Database) validateUpdate(update *Update) error {
	table, ok := db.Schema.Tables[update.Table]
	// table exists
	if !ok {
		return &NoSuchTable{
			TableName: update.Table,
		}
	}
	// table isn't a builtin
	if update.Table == "__tables__" || update.Table == "__columns__" {
		return &BuiltinWriteAttempt{
			TableName: update.Table,
		}
	}
	// column to update exists
	updateColExists := false
	for _, column := range table.Columns {
		if column.Name == update.ColumnName {
			updateColExists = true
		}
	}
	if !updateColExists {
		return &NoSuchColumn{
			TableName:  update.Table,
			ColumnName: update.ColumnName,
		}
	}
	// column in where clause exists
	whereColExists := false
	for _, column := range table.Columns {
		if column.Name == update.WhereColumnName {
			whereColExists = true
		}
	}
	if !whereColExists {
		return &NoSuchColumn{
			TableName:  update.Table,
			ColumnName: update.ColumnName,
		}
	}
	return nil
}

func (conn *Connection) ExecuteUpdate(update *Update, channel *Channel) error {
	startTime := time.Now()

	// Write to table.
	table := conn.Database.Schema.Tables[update.Table]
	rowsUpdated := 0
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(update.Table))
		bucket.ForEach(func(key []byte, value []byte) error {
			record := table.RecordFromBytes(value)
			if record.GetField(update.WhereColumnName).StringVal == update.EqualsValue {
				// Clone and update old record.
				clonedOldRecord := record.Clone()
				record.SetString(update.ColumnName, update.Value)
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
				conn.Database.PushTableEvent(channel, update.Table, clonedOldRecord, clonedNewRecord)
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
	channel.WriteAckMessage(fmt.Sprintf("UPDATE %d", rowsUpdated))

	// Record latency.
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	conn.Database.Metrics.updateLatency.Observe(float64(duration.Nanoseconds()))
	clog.Println(channel, "handled update in", duration)
	return nil
}
