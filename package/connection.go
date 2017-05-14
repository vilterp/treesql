package treesql

import (
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

type Connection struct {
	clientConn      *websocket.Conn
	Messages        chan *ChannelMessage
	ID              int
	Database        *Database
	NextStatementID int
}

func (db *Database) NewConnection(conn *websocket.Conn) *Connection {
	dbConn := &Connection{
		clientConn:      conn,
		Messages:        make(chan *ChannelMessage),
		ID:              db.NextConnectionID,
		Database:        db,
		NextStatementID: 0,
	}
	db.NextConnectionID++
	return dbConn
}

type Channel struct {
	Connection   *Connection
	RawStatement string
	StatementID  int
}

type ChannelMessage struct {
	StatementID int
	Message     interface{}
}

func (conn *Connection) NewChannel(rawStatement string) *Channel {
	channel := &Channel{
		Connection:   conn,
		RawStatement: rawStatement,
		StatementID:  conn.NextStatementID,
	}
	conn.NextStatementID++
	return channel
}

func (channel *Channel) WriteMessage(message interface{}) {
	channel.Connection.Messages <- &ChannelMessage{
		StatementID: channel.StatementID,
		Message:     message,
	}
}

func (conn *Connection) Run() {
	log.Println("connection id", conn.ID, " from", conn.clientConn.RemoteAddr())
	go conn.writeMessagesToSocket()
	for {
		_, message, readErr := conn.clientConn.ReadMessage()
		if readErr != nil {
			log.Println("connection", conn.ID, "terminated:", readErr)
			return
		}
		stringMessage := string(message)
		channel := conn.NewChannel(stringMessage)

		// parse what was sent to us
		statement, err := Parse(stringMessage)
		if err != nil {
			log.Println("connection", conn.ID, "parse error:", err)
			channel.WriteMessage(fmt.Sprintf("parse error: %s", err))
			continue
		}

		// output message received
		// fmt.Print("SQL statement received:", spew.Sdump(statement))

		// validate statement
		queryErr := conn.Database.ValidateStatement(statement)
		if queryErr != nil {
			channel.WriteMessage(fmt.Sprintf("validation error: %s", queryErr))
			log.Println("connection", conn.ID, "statement validation error:", queryErr)
			continue
		}
		conn.ExecuteStatement(statement, channel)
	}
}

func (conn *Connection) ExecuteStatement(statement *Statement, channel *Channel) {
	if statement.Select != nil {
		conn.ExecuteQuery(statement.Select, conn.NextStatementID, channel)
	} else if statement.Insert != nil {
		conn.ExecuteInsert(statement.Insert, channel)
	} else if statement.CreateTable != nil {
		conn.ExecuteCreateTable(statement.CreateTable, channel)
	} else if statement.Update != nil {
		conn.ExecuteUpdate(statement.Update, channel)
	} else {
		panic(fmt.Sprintf("unknown statement type %v", statement))
	}
}

func (conn *Connection) writeMessagesToSocket() {
	for {
		message := <-conn.Messages
		writeErr := conn.clientConn.WriteJSON(message)
		if writeErr != nil {
			panic(writeErr)
		}
	}
}

func (db *Database) PushTableEvent(tableName string, oldRecord *Record, newRecord *Record) {
	db.TableListeners[tableName].TableEvents <- &TableEvent{
		TableName: tableName,
		OldRecord: oldRecord,
		NewRecord: newRecord,
	}
}
