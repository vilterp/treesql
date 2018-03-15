package treesql

import (
	"bufio"
	"context"

	"github.com/gorilla/websocket"
	clog "github.com/vilterp/treesql/pkg/log"
)

type ConnectionID int

type Connection struct {
	clientConn    *websocket.Conn
	ID            ConnectionID
	Database      *Database
	Channels      map[int]*Channel // keyed by statement ID (aka channel id)
	NextChannelID int
	Messages      chan *ChannelMessage
	Context       context.Context
}

func NewConnection(wsConn *websocket.Conn, db *Database, ID int) *Connection {
	ctx := context.WithValue(db.Ctx, clog.ConnIDKey, ID)
	conn := &Connection{
		clientConn:    wsConn,
		ID:            ConnectionID(ID),
		Database:      db,
		Channels:      make(map[int]*Channel),
		NextChannelID: 0,
		Messages:      make(chan *ChannelMessage),
		Context:       ctx,
	}
	go conn.writeMessagesToSocket()
	return conn
}

func (conn *Connection) Ctx() context.Context {
	return conn.Context
}

func (conn *Connection) writeMessagesToSocket() {
	for msg := range conn.Messages {
		writer, err := conn.clientConn.NextWriter(websocket.TextMessage)
		if err != nil {
			clog.Println(conn, "error writing to socket:", err)
			break
		}

		bufWriter := bufio.NewWriter(writer)

		if err := msg.ToVal().WriteAsJSON(bufWriter, msg.GetCaller()); err != nil {
			clog.Println(conn, "error writing msg to conn:", err)
		}
		if err := bufWriter.Flush(); err != nil {
			clog.Println(conn, "error writing msg to conn:", err)
		}
		if err := writer.Close(); err != nil {
			clog.Println(conn, "error writing msg to conn:", err)
		}
	}
}

func (conn *Connection) HandleStatements() {
	clog.Println(conn, "initiated from", conn.clientConn.RemoteAddr())
	for {
		_, message, readErr := conn.clientConn.ReadMessage()
		if readErr != nil {
			clog.Println(conn, "terminated:", readErr)
			conn.Database.removeConn(conn)
			return
		}
		stringMessage := string(message)
		conn.addChannel(stringMessage)
	}
}

func (conn *Connection) addChannel(statement string) {
	channel := NewChannel(statement, conn.NextChannelID, conn)
	conn.NextChannelID++
	conn.Channels[channel.ID] = channel

	channel.HandleStatement()
}

func (conn *Connection) removeChannel(channel *Channel) {
	delete(conn.Channels, channel.ID)
}
