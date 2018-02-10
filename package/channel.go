package treesql

import (
	"context"
	"fmt"

	"github.com/vilterp/treesql/package/log"
)

type Channel struct {
	Connection   *Connection
	RawStatement string
	StatementID  int

	Context context.Context
}

func (channel *Channel) Ctx() context.Context {
	return channel.Context
}

func (conn *Connection) NewChannel(rawStatement string) *Channel {
	stmtID := conn.NextStatementID
	conn.NextStatementID++
	ctx := context.WithValue(conn.Ctx(), log.StmtIDKey, stmtID)
	channel := &Channel{
		Connection:   conn,
		RawStatement: rawStatement,
		StatementID:  stmtID,
		Context:      ctx,
	}
	return channel
}

type ChannelMessage struct {
	StatementID int
	Message     *MessageToClient
}

// ugh. this sucks.
type MessageToClientType int

const (
	ErrorMessage MessageToClientType = iota
	AckMessage
	InitialResultMessage
	RecordUpdateMessage
	TableUpdateMessage
)

func (m *MessageToClientType) MarshalJSON() ([]byte, error) {
	switch *m {
	case ErrorMessage:
		return []byte("\"error\""), nil
	case AckMessage:
		return []byte("\"ack\""), nil
	case InitialResultMessage:
		return []byte("\"initial_result\""), nil
	case RecordUpdateMessage:
		return []byte("\"record_update\""), nil
	case TableUpdateMessage:
		return []byte("\"table_update\""), nil
	}
	return nil, fmt.Errorf("unknown error type %d", *m)
}

func (m *MessageToClientType) UnmarshalText(text []byte) error {
	textStr := string(text)
	switch textStr {
	case "error":
		*m = ErrorMessage
	case "ack":
		*m = AckMessage
	case "initial_result":
		*m = InitialResultMessage
	case "record_update":
		*m = RecordUpdateMessage
	case "table_update":
		*m = TableUpdateMessage
	}
	return nil
}

type MessageToClient struct {
	Type         MessageToClientType `json:"type"`
	ErrorMessage *string             `json:"error,omitempty"`
	AckMessage   *string             `json:"ack,omitempty"`
	// data
	InitialResultMessage *InitialResult `json:"initial_result,omitempty"`
	RecordUpdateMessage  *RecordUpdate  `json:"record_update,omitempty"`
	TableUpdateMessage   *TableUpdate   `json:"table_update,omitempty"`
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

func (channel *Channel) WriteErrorMessage(err error) {
	errStr := err.Error()
	channel.writeMessage(&MessageToClient{
		Type:         ErrorMessage,
		ErrorMessage: &errStr,
	})
}

func (channel *Channel) WriteAckMessage(message string) {
	channel.writeMessage(&MessageToClient{
		Type:       AckMessage,
		AckMessage: &message,
	})
}

func (channel *Channel) WriteInitialResult(result *InitialResult) {
	channel.writeMessage(&MessageToClient{
		Type:                 InitialResultMessage,
		InitialResultMessage: result,
	})
}

func (channel *Channel) WriteTableUpdate(update *TableUpdate) {
	channel.writeMessage(&MessageToClient{
		Type:               TableUpdateMessage,
		TableUpdateMessage: update,
	})
}

func (channel *Channel) WriteRecordUpdate(update *TableEvent, queryPath *QueryPath) {
	channel.writeMessage(&MessageToClient{
		Type: RecordUpdateMessage,
		RecordUpdateMessage: &RecordUpdate{
			QueryPath:  queryPath,
			TableEvent: update,
		},
	})
}

func (channel *Channel) writeMessage(message *MessageToClient) {
	// TODO: why send this to a channel? why not just write it here?
	channel.Connection.Messages <- &ChannelMessage{
		StatementID: channel.StatementID,
		Message:     message,
	}
}
