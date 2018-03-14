package treesql

import (
	"encoding/binary"
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
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
		knownType := column.TypeName == "string" || column.TypeName == "int"
		if !knownType {
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
	// find primary key
	var primaryKey string
	for _, column := range create.Columns {
		if column.PrimaryKey {
			primaryKey = column.Name
			break
		}
	}
	columnRecords := make([]*Record, len(create.Columns))
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		tableSpec := conn.Database.AddTable(create.Name, primaryKey, make([]*ColumnDescriptor, len(create.Columns)))
		// create bucket for new table
		tx.CreateBucket([]byte(create.Name))
		// add to in-memory schema
		// write record to __tables__
		tablesBucket := tx.Bucket([]byte("__tables__"))
		tableRecord := tableSpec.ToRecord(conn.Database)
		tablePutErr := tablesBucket.Put([]byte(create.Name), tableRecord.ToBytes())
		if tablePutErr != nil {
			return tablePutErr
		}
		// write to __columns__
		for idx, parsedColumn := range create.Columns {
			// extract reference
			var reference *ColumnReference
			if parsedColumn.References != nil {
				reference = &ColumnReference{
					TableName: *parsedColumn.References,
				}
			}
			// build column spec
			columnSpec := &ColumnDescriptor{
				ID:               conn.Database.Schema.NextColumnID,
				Name:             parsedColumn.Name,
				ReferencesColumn: reference,
				Type:             NameToType[parsedColumn.TypeName],
			}
			conn.Database.Schema.NextColumnID++
			// put column spec in in-memory schema copy
			// TODO: synchronize access to this mutable shared data structure!!
			tableSpec.Columns[idx] = columnSpec
			// write record to __columns__
			columnRecord := columnSpec.ToRecord(create.Name, conn.Database)
			columnsBucket := tx.Bucket([]byte("__columns__"))
			key := []byte(fmt.Sprintf("%d", columnSpec.ID))
			value := columnRecord.ToBytes()
			columnPutErr := columnsBucket.Put(key, value)
			if columnPutErr != nil {
				return columnPutErr
			}
			columnRecords[idx] = columnRecord
		}
		// push live query messages
		conn.Database.PushTableEvent(channel, "__tables__", nil, tableRecord)
		for _, columnRecord := range columnRecords {
			conn.Database.PushTableEvent(channel, "__columns__", nil, columnRecord)
		}
		// write next column id sequence
		nextColumnIDBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(nextColumnIDBytes, uint32(conn.Database.Schema.NextColumnID))
		tx.Bucket([]byte("__sequences__")).Put([]byte("__next_column_id__"), nextColumnIDBytes)
		return nil
	})
	if updateErr != nil {
		return errors.Wrap(updateErr, "creating table")
	}
	clog.Println(channel, "created table", create.Name)
	channel.WriteAckMessage("CREATE TABLE")
	return nil
}
