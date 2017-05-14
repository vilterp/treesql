package treesql

import (
	"encoding/json"
	"fmt"
)

type Channel struct {
	Connection   *Connection
	RawStatement string
	StatementID  int
}

type ChannelMessage struct {
	StatementID int
	Message     interface{}
}

type MessageToClient struct {
	ErrorMessage  error
	AckMessage    *string
	UpdateMessage interface{}
}

func (message *MessageToClient) MarshalJSON() ([]byte, error) {
	if message.ErrorMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":  "error",
			"error": message.ErrorMessage.Error(),
		})
	} else if message.AckMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type": "ack",
			"ack":  *message.AckMessage,
		})
	} else if message.UpdateMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":   "update",
			"update": message.UpdateMessage,
		})
	} else {
		panic(fmt.Sprintf("unknown message type: %v", message))
	}
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

func (channel *Channel) writeMessage(message *MessageToClient) {
	channel.Connection.Messages <- &ChannelMessage{
		StatementID: channel.StatementID,
		Message:     message,
	}
}

func (channel *Channel) WriteErrorMessage(err error) {
	channel.writeMessage(&MessageToClient{
		ErrorMessage: err,
	})
}

func (channel *Channel) WriteUpdateMessage(update interface{}) {
	channel.writeMessage(&MessageToClient{
		UpdateMessage: update,
	})
}

func (channel *Channel) WriteAckMessage(message string) {
	channel.writeMessage(&MessageToClient{
		AckMessage: &message,
	})
}
