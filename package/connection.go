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
	ID              int
	Database        *Database
	NextStatementID int
	Messages        chan *ChannelMessage
	Context         context.Context
}

func (db *Database) NewConnection(conn *websocket.Conn) *Connection {
	ctx := context.WithValue(db.Ctx, clog.ConnIDKey, db.NextConnectionID)
	dbConn := &Connection{
		clientConn:      conn,
		ID:              db.NextConnectionID,
		Database:        db,
		NextStatementID: 0,
		Messages:        make(chan *ChannelMessage),
		Context:         ctx,
	}
	db.NextConnectionID++
	go dbConn.writeMessagesToSocket()
	return dbConn
}

func (conn *Connection) Ctx() context.Context {
	return conn.Context
}

func (conn *Connection) writeMessagesToSocket() {
	for msg := range conn.Messages {
		if err := conn.clientConn.WriteJSON(msg); err != nil {
			clog.Println(conn, "error writing to socket:", err)
		}
	}
}

func (conn *Connection) HandleStatements() {
	clog.Println(conn, "initiated from", conn.clientConn.RemoteAddr())
	for {
		_, message, readErr := conn.clientConn.ReadMessage()
		if readErr != nil {
			clog.Println(conn, "terminated:", readErr)
			return
		}
		stringMessage := string(message)
		channel := conn.NewChannel(stringMessage)

		if err := channel.HandleStatement(); err != nil {
			clog.Printf(channel, err.Error())
			channel.WriteErrorMessage(err)
		}
	}
}

func (channel *Channel) HandleStatement() error {
	// parse what was sent to us
	statement, err := Parse(channel.RawStatement)
	if err != nil {
		return &ParseError{error: err}
	}

	// validate statement
	queryErr := channel.Connection.Database.ValidateStatement(statement)
	if queryErr != nil {
		return &ValidationError{error: queryErr}
	}
	return channel.Connection.ExecuteStatement(statement, channel)
}

func (conn *Connection) ExecuteStatement(statement *Statement, channel *Channel) error {
	if statement.Select != nil {
		return conn.ExecuteTopLevelQuery(statement.Select, channel)
	}
	if statement.Insert != nil {
		return conn.ExecuteInsert(statement.Insert, channel)
	}
	if statement.CreateTable != nil {
		return conn.ExecuteCreateTable(statement.CreateTable, channel)
	}
	if statement.Update != nil {
		return conn.ExecuteUpdate(statement.Update, channel)
	}
	panic(fmt.Sprintf("unknown statement type %v", statement))
}
