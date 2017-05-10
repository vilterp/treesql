package treesql

import (
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
)

func (db *Database) NewConnection(conn *websocket.Conn) *Connection {
	dbConn := &Connection{
		ClientConn:  conn,
		ID:          db.NextConnectionId,
		Database:    db,
		NextQueryId: 0,
	}
	dbConn.NextQueryId++
	return dbConn
}

type Connection struct {
	ClientConn  *websocket.Conn
	ID          int
	Database    *Database
	NextQueryId int
}

func (conn *Connection) Run() {
	log.Println("connection id", conn.ID, " from", conn.ClientConn.RemoteAddr())
	for {
		_, message, readErr := conn.ClientConn.ReadMessage()
		if readErr != nil {
			log.Println("connection", conn.ID, "terminated:", readErr)
			return
		}

		// parse what was sent to us
		statement, err := Parse(string(message))
		if err != nil {
			log.Println("connection", conn.ID, "parse error:", err)
			conn.ClientConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("parse error: %s", err)))
			continue
		}

		// output message received
		// fmt.Print("SQL statement received:", spew.Sdump(statement))

		// validate statement
		queryErr := conn.Database.ValidateStatement(statement)
		if queryErr != nil {
			conn.ClientConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("parse error: %s", err)))
			log.Println("connection", conn.ID, "statement validation error:", queryErr)
			continue
		}
		if statement.Select != nil {
			// execute query
			conn.ExecuteQuery(statement.Select, conn.NextQueryId, conn.ClientConn)
			conn.NextQueryId++
		} else if statement.Insert != nil {
			conn.ExecuteInsert(statement.Insert, conn.ClientConn)
		} else if statement.CreateTable != nil {
			conn.ExecuteCreateTable(statement.CreateTable, conn.ClientConn)
		} else if statement.Update != nil {
			conn.ExecuteUpdate(statement.Update, conn.ClientConn)
		} else {
			panic(fmt.Sprintf("unknown statement %v", statement))
		}
	}
}

// TODO: some other file, alongside executor.go? idk
func (conn *Connection) ExecuteInsert(insert *Insert, channel *websocket.Conn) {
	table := conn.Database.Schema.Tables[insert.Table]
	record := table.NewRecord()
	for idx, value := range insert.Values {
		record.SetString(table.Columns[idx].Name, value)
	}
	key := record.GetField(table.PrimaryKey).StringVal
	// write to table
	// TODO: handle any errors
	conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(insert.Table))
		bucket.Put([]byte(key), record.ToBytes())
		return nil
	})
	// push to live query listeners
	conn.Database.TableListeners[insert.Table].TableEvents <- &TableEvent{
		NewRecord: record,
		OldRecord: nil,
	}
	log.Println("connection", conn.ID, "handled insert")
	channel.WriteMessage(websocket.TextMessage, []byte("INSERT 1\n")) // heh
}

func (conn *Connection) ExecuteCreateTable(create *CreateTable, channel *websocket.Conn) {
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
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		// create bucket for new table
		tx.CreateBucket([]byte(create.Name))
		// add to in-memory schema
		// TODO: synchronize access to this shared mutable data structure!
		conn.Database.Schema.Tables[tableSpec.Name] = tableSpec
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
	conn.Database.AddTableListener(tableSpec)
	if updateErr != nil {
		// TODO: structured errors on the wire...
		channel.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("error creating table: %s", updateErr)))
		log.Println("connection", conn.ID, "error creating table:", updateErr)
	} else {
		log.Println("connection", conn.ID, "created table", create.Name)
		channel.WriteMessage(websocket.TextMessage, []byte("CREATE TABLE"))
	}
}

func (conn *Connection) ExecuteUpdate(update *Update, channel *websocket.Conn) {
	startTime := time.Now()
	table := conn.Database.Schema.Tables[update.Table]
	rowsUpdated := 0
	updateErr := conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(update.Table))
		bucket.ForEach(func(key []byte, value []byte) error {
			record := table.RecordFromBytes(value)
			if record.GetField(update.WhereColumnName).StringVal == update.EqualsValue {
				record.SetString(update.ColumnName, update.Value)
				rowUpdateErr := bucket.Put(key, record.ToBytes())
				if rowUpdateErr != nil {
					return rowUpdateErr
				}
				rowsUpdated++
			}
			return nil
		})
		return nil
	})
	if updateErr != nil {
		channel.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("error executing update: %s", updateErr)))
	} else {
		channel.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("UPDATE %d", rowsUpdated)))
		endTime := time.Now()
		log.Println("connection", conn.ID, "handled update in", endTime.Sub(startTime))
	}
}
