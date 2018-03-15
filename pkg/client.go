package treesql

// maybe this should be in a different package idk
// this should pretty much be the same API as TreeSQLClient.js

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

type Client struct {
	webSocketConn    *websocket.Conn
	URL              string
	nextStatementID  int
	statementsToSend chan *statementRequest
	incomingMessages chan *BasicChannelMessage
	channels         map[int]*ClientChannel
	ServerClosed     chan bool
}

type statementRequest struct {
	Statement  string
	ResultChan chan *ClientChannel
}

func NewClient(url string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	clientConn := &Client{
		nextStatementID:  0,
		webSocketConn:    conn,
		URL:              url,
		statementsToSend: make(chan *statementRequest),
		incomingMessages: make(chan *BasicChannelMessage),
		channels:         map[int]*ClientChannel{},
		ServerClosed:     make(chan bool),
	}
	go clientConn.handleStatements()
	go clientConn.handleIncoming()
	return clientConn, nil
}

func (conn *Client) Close() error {
	return conn.webSocketConn.Close()
	// idk if it should also do something to the channels
}

func (conn *Client) handleStatements() {
	for {
		select {
		case request := <-conn.statementsToSend:
			channel := &ClientChannel{
				Conn:        conn,
				StatementID: conn.nextStatementID,
				Statement:   request.Statement,
				Updates:     make(chan *BasicMessageToClient),
			}
			conn.nextStatementID++
			conn.channels[channel.StatementID] = channel
			request.ResultChan <- channel
			conn.webSocketConn.WriteMessage(websocket.TextMessage, []byte(request.Statement))

		case incomingMsg := <-conn.incomingMessages:
			if incomingMsg == nil {
				for _, channel := range conn.channels {
					close(channel.Updates)
				}
				return
			}
			channel := conn.channels[incomingMsg.StatementID]
			channel.Updates <- incomingMsg.Message
		}
	}
}

// TODO: actually parse Values
type BasicChannelMessage struct {
	StatementID int
	Message     *BasicMessageToClient
}

type BasicMessageToClient struct {
	Type         MessageToClientType `json:"type"`
	ErrorMessage *string             `json:"error,omitempty"`
	AckMessage   *string             `json:"ack,omitempty"`
	// data
	InitialResultMessage *basicInitialResult `json:"initial_result,omitempty"`
}

type basicInitialResult struct {
	Type  string
	Value interface{}
}

func (conn *Client) handleIncoming() {
	defer conn.webSocketConn.Close()
	for {
		parsedMessage := &BasicChannelMessage{}
		err := conn.webSocketConn.ReadJSON(&parsedMessage)

		if err != nil {
			log.Println("error in handleIncoming:", err)
			close(conn.incomingMessages)
			conn.ServerClosed <- true
			return
			// uh... should probably recover gracefully from this, but
			// idk how to return an error from a goroutine. how would its
			// supervisor (???) handle it? I want erlang lol
		}
		conn.incomingMessages <- parsedMessage
	}
}

type ClientChannel struct {
	Conn        *Client
	StatementID int
	Statement   string
	Updates     chan *BasicMessageToClient
}

func (conn *Client) RunStatement(statement string) *ClientChannel {
	resultChan := make(chan *ClientChannel)
	conn.statementsToSend <- &statementRequest{
		ResultChan: resultChan,
		Statement:  statement,
	}
	return <-resultChan
}

func (conn *Client) LiveQuery(query string) (*basicInitialResult, *ClientChannel, error) {
	channel := conn.RunStatement(query)
	update := <-channel.Updates
	if update.ErrorMessage != nil {
		return nil, nil, errors.New(*update.ErrorMessage)
	}
	if update.InitialResultMessage != nil {
		return update.InitialResultMessage, channel, nil
	}
	return nil, nil, errors.New("query result neither error nor initial result")
}

func (conn *Client) Query(query string) (*basicInitialResult, error) {
	resultChan := conn.RunStatement(query)
	update := <-resultChan.Updates
	if update.ErrorMessage != nil {
		return nil, errors.New(*update.ErrorMessage)
	}
	if update.InitialResultMessage != nil {
		return update.InitialResultMessage, nil
	}
	return nil, errors.New("query result neither error nor initial result")
}

func (conn *Client) Exec(statement string) (string, error) {
	resultChan := conn.RunStatement(statement)
	update := <-resultChan.Updates
	if update.ErrorMessage != nil {
		return "", errors.New(*update.ErrorMessage)
	} else if update.AckMessage != nil {
		return *update.AckMessage, nil
	}
	return "", errors.New("exec result neither error nor ack")
}
