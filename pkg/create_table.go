package treesql

import (
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/vilterp/treesql/package/lang"
	clog "github.com/vilterp/treesql/pkg/log"
)

func (db *Database) validateCreateTable(create *CreateTable) error {
	// does table already exist?
	_, ok := db.Schema.Tables[create.Name]
	if ok {
		return &TableAlreadyExists{TableName: create.Name}
	}
	// types are real
	for _, column := range create.Columns {
		_, err := lang.ParseType(column.TypeName)
		if err != nil {
			return &NonexistentType{TypeName: column.TypeName}
		}
	}
	// only one primary key
	primaryKeyCount := 0
	for _, column := range create.Columns {
		if column.PrimaryKey {
			primaryKeyCount++
		}
	}
	if primaryKeyCount != 1 {
		return &WrongNoPrimaryKey{Count: primaryKeyCount}
	}
	// referenced table exists
	// TODO: column same type as primary key
	for _, column := range create.Columns {
		if column.References != nil {
			_, tableExists := db.Schema.Tables[*column.References]
			if !tableExists {
				return &NoSuchTable{TableName: *column.References}
			}
		}
	}
	// TODO: dedup column names
	return nil
}

func (conn *Connection) ExecuteCreateTable(create *CreateTable, channel *Channel) error {
	tableDesc, err := conn.Database.buildTableDescriptor(create)
	if err != nil {
		return err
	}
	tableRecord := tableDesc.ToRecord(conn.Database)
	columnRecords := make([]*Record, len(create.Columns))
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		// TODO: give ids to tables; create bucket from that
		// create bucket for new table
		tableBucket, err := tx.CreateBucket([]byte(create.Name))
		if err != nil {
			return err
		}
		// create a bucket for each index
		// primary key, and each column that references another table
		for _, col := range tableDesc.Columns {
			if col.ReferencesColumn != nil || tableDesc.PrimaryKey == col.Name {
				// TODO: factor this out to an encoding file
				colIDBytes := make([]byte, 4)
				binary.BigEndian.PutUint32(colIDBytes, uint32(col.ID))
				_, err := tableBucket.CreateBucket(colIDBytes)
				if err != nil {
					return err
				}
			}
		}
		// write record to __tables__
		tablesBucket := tx.Bucket([]byte("__tables__"))
		tableBytes, err := tableRecord.ToBytes()
		if err != nil {
			return err
		}
		if err := tablesBucket.Put([]byte(create.Name), tableBytes); err != nil {
			return err
		}
		// write column descriptors to __columns__
		for idx, columnDesc := range tableDesc.Columns {
			// serialize descriptor
			columnRecord := columnDesc.ToRecord(create.Name, conn.Database)
			value, err := columnRecord.ToBytes()
			if err != nil {
				return err
			}
			// write to bucket
			columnsBucket := tx.Bucket([]byte("__columns__"))
			key := []byte(fmt.Sprintf("%d", columnDesc.ID))
			if err := columnsBucket.Put(key, value); err != nil {
				return err
			}
			columnRecords[idx] = columnRecord
		}
		// write next column id sequence
		nextColumnIDBytes := encodeInteger(int32(conn.Database.Schema.NextColumnID))
		tx.Bucket([]byte("__sequences__")).Put([]byte("__next_column_id__"), nextColumnIDBytes)
		return nil
	})
	if updateErr != nil {
		return errors.Wrap(updateErr, "creating table")
	}
	// add to in-memory schema
	conn.Database.addTableDescriptor(tableDesc)
	// push live query messages
	conn.Database.PushTableEvent(channel, "__tables__", nil, tableRecord)
	for _, columnRecord := range columnRecords {
		conn.Database.PushTableEvent(channel, "__columns__", nil, columnRecord)
	}
	clog.Println(channel, "created table", create.Name)
	channel.WriteAckMessage("CREATE TABLE")
	return nil
}
