package treesql

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Channel struct {
	Connection   *Connection
	RawStatement string
	StatementID  int
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

type ChannelMessage struct {
	StatementID int
	Message     *MessageToClient
}

type MessageToClient struct {
	ErrorMessage error
	AckMessage   *string
	// data
	InitialResultMessage *InitialResult
	RecordUpdateMessage  *RecordUpdate
	TableUpdateMessage   *TableUpdate
}

type InitialResult struct {
	Schema map[string]interface{}
	Data   SelectResult
}

type TableUpdate struct {
	QueryPath *QueryPath
	Selection SelectResult
}

type RecordUpdate struct {
	TableEvent *TableEvent
	QueryPath  *QueryPath
}

// TODO: factor this out into an interface or something
func (message *MessageToClient) MarshalJSON() ([]byte, error) {
	if message.ErrorMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":    "error",
			"payload": message.ErrorMessage.Error(),
		})
	} else if message.AckMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":    "ack",
			"payload": *message.AckMessage,
		})
	} else if message.InitialResultMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":    "initial_result",
			"payload": message.InitialResultMessage,
		})
	} else if message.RecordUpdateMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":    "record_update",
			"payload": message.RecordUpdateMessage,
		})
	} else if message.TableUpdateMessage != nil {
		return json.Marshal(map[string]interface{}{
			"type":    "table_update",
			"payload": message.TableUpdateMessage,
		})
	} else {
		panic(fmt.Sprintf("unknown message type: %v", message))
	}
}

func (message *MessageToClient) UnmarshalJSON(data []byte) error {
	theMap := map[string]interface{}{}
	mapErr := json.Unmarshal(data, &theMap)
	if mapErr != nil {
		return mapErr
	}
	switch theMap["type"] {
	case "error":
		message.ErrorMessage = errors.New(theMap["payload"].(string))
		return nil

	case "ack":
		str := theMap["payload"].(string)
		message.AckMessage = &str
		return nil

	case "initial_result":
		message.InitialResultMessage = &InitialResult{}
		return json.Unmarshal(data, message.InitialResultMessage)

	case "record_update":
		message.RecordUpdateMessage = &RecordUpdate{}
		return json.Unmarshal(data, message.RecordUpdateMessage)

	case "table_update":
		message.TableUpdateMessage = &TableUpdate{}
		return json.Unmarshal(data, message.TableUpdateMessage)

	default:
		return fmt.Errorf("unrecognized message type: %v", theMap["type"])
	}
}

func (channel *Channel) WriteErrorMessage(err error) {
	channel.writeMessage(&MessageToClient{
		ErrorMessage: err,
	})
}

func (channel *Channel) WriteAckMessage(message string) {
	channel.writeMessage(&MessageToClient{
		AckMessage: &message,
	})
}

func (channel *Channel) WriteInitialResult(result *InitialResult) {
	channel.writeMessage(&MessageToClient{
		InitialResultMessage: result,
	})
}

func (channel *Channel) WriteTableUpdate(update *TableUpdate) {
	channel.writeMessage(&MessageToClient{
		TableUpdateMessage: update,
	})
}

func (channel *Channel) WriteRecordUpdate(update *TableEvent, queryPath *QueryPath) {
	channel.writeMessage(&MessageToClient{
		RecordUpdateMessage: &RecordUpdate{
			QueryPath:  queryPath,
			TableEvent: update,
		},
	})
}

func (channel *Channel) writeMessage(message *MessageToClient) {
	channel.Connection.Messages <- &ChannelMessage{
		StatementID: channel.StatementID,
		Message:     message,
	}
}
