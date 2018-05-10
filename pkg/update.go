package treesql

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/vilterp/treesql/pkg/lang"
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

	table := conn.database.schema.tables[update.Table]

	// Write to table.
	rowsUpdated := 0
	updateErr := conn.database.boltDB.Update(func(tx *bolt.Tx) error {
		// TODO: plan and execute this in FP...
		bucket := tx.Bucket([]byte(update.Table)).Bucket(table.pkBucketKey())
		err := bucket.ForEach(func(keyBytes []byte, valBytes []byte) error {
			// Parse record
			value, err := lang.Decode(valBytes)
			if err != nil {
				return fmt.Errorf(`couldn't decode "%s": %v`, string(valBytes), err)
			}
			record, ok := value.(*lang.VRecord)
			if !ok {
				return fmt.Errorf("decoded value not record: %v", value.Format())
			}
			// Check that it's in our update set
			colVal := record.GetValue(update.WhereColumnName)
			colValStr, ok := colVal.(*lang.VString)
			if !ok {
				return fmt.Errorf("can't do comparison on non-string value: %v", colVal)
			}

			if string(*colValStr) == update.EqualsValue {
				updated := record.Update(update.ColumnName, lang.NewVString(update.Value))
				updatedBytes, err := lang.Encode(updated)
				if err != nil {
					return err
				}
				if err := bucket.Put(keyBytes, updatedBytes); err != nil {
					return err
				}
				// Send live query updates.
				conn.database.pushTableEvent(channel, update.Table, record, updated)
				rowsUpdated++
			}
			return nil
		})
		if err != nil {
			return err
		}

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
