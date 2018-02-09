package treesql

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	clog "github.com/vilterp/treesql/package/log"
)

type ConnectionID int
type StatementID int

type Connection struct {
	clientConn      *websocket.Conn
	Messages        chan *ChannelMessage
	ID              int
	Database        *Database
	NextStatementID int
	Context         context.Context
}

func (db *Database) NewConnection(conn *websocket.Conn) *Connection {
	ctx := context.WithValue(db.Ctx, clog.ConnIDKey, db.NextConnectionID)
	dbConn := &Connection{
		clientConn:      conn,
		Messages:        make(chan *ChannelMessage),
		ID:              db.NextConnectionID,
		Database:        db,
		NextStatementID: 0,
		Context:         ctx,
	}
	db.NextConnectionID++
	return dbConn
}

func (conn *Connection) Ctx() context.Context {
	return conn.Context
}

func (conn *Connection) HandleStatements() {
	clog.Println(conn, "initiated from", conn.clientConn.RemoteAddr())
	go conn.writeMessagesToSocket()
	for {
		_, message, readErr := conn.clientConn.ReadMessage()
		if readErr != nil {
			clog.Println(conn, "terminated:", readErr)
			return
		}
		stringMessage := string(message)
		channel := conn.NewChannel(stringMessage)

		// parse what was sent to us
		statement, err := Parse(stringMessage)
		if err != nil {
			clog.Println(channel, "parse error:", err)
			channel.WriteErrorMessage(fmt.Errorf("parse error: %s", err))
			continue
		}

		// validate statement
		queryErr := conn.Database.ValidateStatement(statement)
		if queryErr != nil {
			clog.Println(channel, "statement validation error:", queryErr)
			channel.WriteErrorMessage(fmt.Errorf("validation error: %s", queryErr))
			continue
		}
		conn.ExecuteStatement(statement, channel)
	}
}

func (conn *Connection) ExecuteStatement(statement *Statement, channel *Channel) {
	if statement.Select != nil {
		conn.ExecuteTopLevelQuery(statement.Select, channel)
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
			clog.Println(conn, "error couldn't write to socket", writeErr)
			// TODO: when a connection closes, clear out listeners!
		}
	}
}
