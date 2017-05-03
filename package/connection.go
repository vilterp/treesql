package treesql

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"net"

	"github.com/boltdb/bolt"
)

type Connection struct {
	ClientConn net.Conn
	ID         int
	Database   *Database
}

func (conn *Connection) Run() {
	log.Printf("connection id %d from %s\n", conn.ID, conn.ClientConn.RemoteAddr())
	for {
		// will listen for message to process ending in newline (\n)
		message, err := bufio.NewReader(conn.ClientConn).ReadString('\n')

		if err != nil {
			log.Printf("connection %d terminated: %v\n", conn.ID, err)
			return
		}

		// parse what was sent to us
		statement, err := Parse(message)
		if err != nil {
			log.Println("connection", conn.ID, "parse error:", err)
			conn.ClientConn.Write([]byte(fmt.Sprintf("parse error: %s\n", err)))
			continue
		}

		// output message received
		// fmt.Print("SQL statement received:", spew.Sdump(statement))

		// validate statement
		queryErr := conn.Database.ValidateStatement(statement)
		if queryErr != nil {
			conn.ClientConn.Write([]byte(fmt.Sprintf("statement error: %s\n", queryErr)))
			log.Println("connection", conn.ID, "statement validation error", queryErr)
			continue
		}
		if statement.Select != nil {
			// execute query
			conn.ExecuteQuery(statement.Select)
		} else if statement.Insert != nil {
			conn.ExecuteInsert(statement.Insert)
		} else if statement.CreateTable != nil {
			conn.ExecuteCreateTable(statement.CreateTable)
		}
	}
}

// TODO: some other file, alongside executor.go? idk
func (conn *Connection) ExecuteInsert(insert *Insert) {
	// TODO: handle any errors
	conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(insert.Table))
		table := conn.Database.Schema.Tables[insert.Table]
		record := table.NewRecord()
		for idx, value := range insert.Values {
			record.SetString(table.Columns[idx].Name, value)
		}
		key := record.GetField(table.PrimaryKey).StringVal
		bucket.Put([]byte(key), record.ToBytes())
		return nil
	})
	log.Println("connection", conn.ID, "handled insert")
	conn.ClientConn.Write([]byte("INSERT 1\n")) // heh
}

func (conn *Connection) ExecuteCreateTable(create *CreateTable) {
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		// create bucket for new table
		tx.CreateBucket([]byte(create.Name))
		// write to __tables__
		var primaryKey string
		for _, column := range create.Columns {
			if column.PrimaryKey {
				primaryKey = column.Name
				break
			}
		}
		tableSpec := &Table{
			Name:       create.Name,
			Columns:    make([]*Column, len(create.Columns)),
			PrimaryKey: primaryKey,
		}
		// add to in-memory schema
		// TODO: synchronize access to this shared mutable data structure!
		conn.Database.Schema.Tables[tableSpec.Name] = tableSpec
		// write record
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
			columnSpec := &Column{
				Id:               conn.Database.Schema.NextColumnId,
				Name:             parsedColumn.Name,
				ReferencesColumn: reference,
				Type:             NameToType[parsedColumn.TypeName],
			}
			conn.Database.Schema.NextColumnId++
			// put column spec in in-memory schema copy
			// TODO: synchronize access to this mutable shared data structure!!
			tableSpec.Columns[idx] = columnSpec
			// write record
			columnRecord := columnSpec.ToRecord(create.Name, conn.Database)
			columnsBucket := tx.Bucket([]byte("__columns__"))
			key := []byte(fmt.Sprintf("%d", columnSpec.Id))
			value := columnRecord.ToBytes()
			columnPutErr := columnsBucket.Put(key, value)
			if columnPutErr != nil {
				return columnPutErr
			}
		}
		// write next column id sequence
		nextColumnIdBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(nextColumnIdBytes, uint32(conn.Database.Schema.NextColumnId))
		tx.Bucket([]byte("__sequences__")).Put([]byte("__next_column_id__"), nextColumnIdBytes)
		return nil
	})
	if updateErr != nil {
		// TODO: structured errors on the wire...
		conn.ClientConn.Write([]byte(fmt.Sprintf("error creating table: %s\n", updateErr)))
		log.Println("connection", conn.ID, "error creating table:", updateErr)
	} else {
		log.Println("connection", conn.ID, "created table", create.Name)
		conn.ClientConn.Write([]byte("CREATE TABLE\n"))
	}
}
