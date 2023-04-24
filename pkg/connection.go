package treesql

import (
	"bufio"
	"context"

	"github.com/gorilla/websocket"
	clog "github.com/vilterp/treesql/pkg/log"
)

type connectionID int

type connection struct {
	clientConn    *websocket.Conn
	id            connectionID
	database      *Database
	channels      map[int]*channel // keyed by statement id (aka channel id)
	nextChannelID int
	messages      chan *ChannelMessage
	context       context.Context
}

func newConnection(wsConn *websocket.Conn, db *Database, ID int) *connection {
	ctx := context.WithValue(db.ctx, clog.ConnIDKey, ID)
	conn := &connection{
		clientConn:    wsConn,
		id:            connectionID(ID),
		database:      db,
		channels:      make(map[int]*channel),
		nextChannelID: 0,
		messages:      make(chan *ChannelMessage),
		context:       ctx,
	}
	go conn.writeMessagesToSocket()
	return conn
}

func (conn *connection) Ctx() context.Context {
	return conn.context
}

func (conn *connection) writeMessagesToSocket() {
	for msg := range conn.messages {
		writer, err := conn.clientConn.NextWriter(websocket.TextMessage)
		if err != nil {
			clog.Println(conn, "error writing to socket:", err)
			break
		}

		bufWriter := bufio.NewWriter(writer)

		if err := msg.toVal().WriteAsJSON(bufWriter, msg.getCaller()); err != nil {
			clog.Println(conn, "error writing msg to conn: writing value: ", err)
		}
		if err := bufWriter.Flush(); err != nil {
			clog.Println(conn, "error writing msg to conn: flusing buffer: ", err)
		}
		if err := writer.Close(); err != nil {
			clog.Println(conn, "error writing msg to conn: closing writer: ", err)
		}
	}
}

func (conn *connection) handleStatements() {
	clog.Println(conn, "initiated from", conn.clientConn.RemoteAddr())
	for {
		_, message, readErr := conn.clientConn.ReadMessage()
		if readErr != nil {
			clog.Println(conn, "terminated:", readErr)
			conn.database.removeConn(conn)
			return
		}
		stringMessage := string(message)
		conn.addChannel(stringMessage)
	}
}

func (conn *connection) addChannel(statement string) {
	channel := newChannel(statement, conn.nextChannelID, conn)
	conn.nextChannelID++
	conn.channels[channel.id] = channel

	channel.handleStatement()
}

func (conn *connection) removeChannel(channel *channel) {
	delete(conn.channels, channel.id)
}
