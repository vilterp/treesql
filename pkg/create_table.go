package treesql

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/vilterp/treesql/pkg/lang"
	clog "github.com/vilterp/treesql/pkg/log"
)

func (db *Database) validateCreateTable(create *CreateTable) error {
	// does table already exist?
	_, ok := db.schema.tables[create.Name]
	if ok {
		return &tableAlreadyExists{TableName: create.Name}
	}
	// types are real
	for _, column := range create.Columns {
		_, err := lang.ParseDecodableType(column.TypeName)
		if err != nil {
			return err
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
		return &wrongNoPrimaryKey{Count: primaryKeyCount}
	}
	// referenced table exists
	// TODO: column same type as primary key
	for _, column := range create.Columns {
		if column.References != nil {
			_, tableExists := db.schema.tables[*column.References]
			if !tableExists {
				return &noSuchTable{TableName: *column.References}
			}
		}
	}
	// TODO: dedup column names
	return nil
}

func (conn *connection) executeCreateTable(create *CreateTable, channel *channel) error {
	// Build and serialize table desc.
	tableDesc, err := conn.database.buildTableDescriptor(create)
	if err != nil {
		return err
	}
	tableRecord := tableDesc.toRecord(conn.database)
	tableBytes, err := lang.Encode(tableRecord)
	if err != nil {
		return err
	}

	columnRecords := make([]*lang.VRecord, len(create.Columns))
	encodedColumnRecords := make([][]byte, len(create.Columns))

	updateErr := conn.database.boltDB.Update(func(tx *bolt.Tx) error {
		// TODO: give ids to tables; create bucket from that
		// create bucket for new table
		tableBucket, err := tx.CreateBucket([]byte(create.Name))
		if err != nil {
			return err
		}
		// create a bucket for each index
		// primary key, and each column that references another table
		for _, col := range tableDesc.columns {
			if col.referencesColumn != nil || tableDesc.primaryKey == col.name {
				// TODO: factor this out to an encoding file
				// TODO: non-unique indexes for foreign key columns
				_, err := tableBucket.CreateBucket(lang.EncodeInteger(int32(col.id)))
				if err != nil {
					return err
				}
			}
		}
		// write record to __tables__
		tablesBucket := tx.Bucket([]byte("__tables__"))
		if err := tablesBucket.Put([]byte(create.Name), tableBytes); err != nil {
			return err
		}
		// write column descriptors to __columns__
		for idx, columnDesc := range tableDesc.columns {
			// serialize descriptor
			columnRecord := columnDesc.toRecord(create.Name, conn.database)
			columnRecordBytes, err := lang.Encode(columnRecord)
			if err != nil {
				return err
			}
			// write to bucket
			columnsBucket := tx.Bucket([]byte("__columns__"))
			key := []byte(fmt.Sprintf("%d", columnDesc.id))
			if err := columnsBucket.Put(key, columnRecordBytes); err != nil {
				return err
			}
			columnRecords[idx] = columnRecord
			encodedColumnRecords[idx] = columnRecordBytes
		}
		// write next column id sequence
		nextColumnIDBytes := lang.EncodeInteger(int32(conn.database.schema.nextColumnID))
		tx.Bucket([]byte("__sequences__")).Put([]byte("__next_column_id__"), nextColumnIDBytes)
		return nil
	})
	if updateErr != nil {
		return errors.Wrap(updateErr, "creating table")
	}
	// add to in-memory schema
	conn.database.addTableDescriptor(tableDesc)
	// push live query messages
	conn.database.pushTableEvent(channel, "__tables__", nil, tableBytes)
	for _, encodedColumnRecord := range encodedColumnRecords {
		conn.database.pushTableEvent(channel, "__columns__", nil, encodedColumnRecord)
	}
	clog.Println(channel, "created table", create.Name)
	channel.writeAckMessage("CREATE TABLE")
	return nil
}
