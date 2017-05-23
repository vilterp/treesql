package treesql

// maybe this should be in a different package idk
// this should pretty much be the same API as TreeSQLClient.js

import (
	"fmt"

	"github.com/gorilla/websocket"
)

type ClientConn struct {
	WebSocketConn    *websocket.Conn
	NextStatementID  int
	StatementsToSend chan *StatementRequest
	IncomingMessages chan *ChannelMessage
	Channels         map[int]*ClientChannel
}

type StatementRequest struct {
	Statement  string
	ResultChan chan *ClientChannel
}

func NewClientConn(url string) (*ClientConn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	clientConn := &ClientConn{
		NextStatementID:  0,
		WebSocketConn:    conn,
		StatementsToSend: make(chan *StatementRequest),
		IncomingMessages: make(chan *ChannelMessage),
		Channels:         map[int]*ClientChannel{},
	}
	go clientConn.handleStatements()
	go clientConn.handleIncoming()
	return clientConn, nil
}

func (conn *ClientConn) handleStatements() {
	for {
		select {
		case request := <-conn.StatementsToSend:
			channel := &ClientChannel{
				Conn:        conn,
				StatementID: conn.NextStatementID,
				Statement:   request.Statement,
				Updates:     make(chan *MessageToClient),
			}
			conn.NextStatementID++
			conn.Channels[channel.StatementID] = channel
			request.ResultChan <- channel
			conn.WebSocketConn.WriteMessage(websocket.TextMessage, []byte(request.Statement))
			fmt.Println("wrote message")

		case incomingMsg := <-conn.IncomingMessages:
			channel := conn.Channels[incomingMsg.StatementID]
			channel.Updates <- incomingMsg.Message
			fmt.Println("incoming message", incomingMsg.Message)
		}
	}
}

func (conn *ClientConn) handleIncoming() {
	defer conn.WebSocketConn.Close()
	for {
		parsedMessage := &ChannelMessage{}
		fmt.Println("about to read json")
		err := conn.WebSocketConn.ReadJSON(&parsedMessage)
		fmt.Println("read json")
		if err != nil {
			panic(err)
			// uh... should probably recover gracefully from this, but
			// idk how to return an error from a goroutine. how would its
			// supervisor (???) handle it? I want erlang lol
		}
		conn.IncomingMessages <- parsedMessage
	}
}

type ClientChannel struct {
	Conn        *ClientConn
	StatementID int
	Statement   string
	Updates     chan *MessageToClient
}

func (conn *ClientConn) SendStatement(statement string) *ClientChannel {
	resultChan := make(chan *ClientChannel)
	conn.StatementsToSend <- &StatementRequest{
		ResultChan: resultChan,
		Statement:  statement,
	}
	return <-resultChan
}
