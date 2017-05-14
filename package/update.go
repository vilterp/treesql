package treesql

import (
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
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

func (conn *Connection) ExecuteUpdate(update *Update, channel *Channel) {
	startTime := time.Now()
	table := conn.Database.Schema.Tables[update.Table]
	rowsUpdated := 0
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(update.Table))
		bucket.ForEach(func(key []byte, value []byte) error {
			record := table.RecordFromBytes(value)
			if record.GetField(update.WhereColumnName).StringVal == update.EqualsValue {
				clonedOldRecord := record.Clone()
				record.SetString(update.ColumnName, update.Value)
				clonedNewRecord := record.Clone()
				rowUpdateErr := bucket.Put(key, record.ToBytes())
				if rowUpdateErr != nil {
					return rowUpdateErr
				}
				// send live query updates
				conn.Database.PushTableEvent(update.Table, clonedOldRecord, clonedNewRecord)
				rowsUpdated++
			}
			return nil
		})
		return nil
	})
	if updateErr != nil {
		channel.WriteErrorMessage(fmt.Errorf("error executing update: %s", updateErr))
	} else {
		channel.WriteAckMessage(fmt.Sprintf("UPDATE %d", rowsUpdated))
		endTime := time.Now()
		log.Println("connection", conn.ID, "handled update in", endTime.Sub(startTime))
	}
}
