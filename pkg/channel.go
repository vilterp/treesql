package treesql

import (
	"context"
	"fmt"

	clog "github.com/vilterp/treesql/pkg/log"
)

type ChannelID int

type Channel struct {
	Connection   *Connection
	RawStatement string
	ID           int // unique with containing connection

	Context context.Context
}

func (channel *Channel) Ctx() context.Context {
	return channel.Context
}

func NewChannel(rawStatement string, ID int, conn *Connection) *Channel {
	ctx := context.WithValue(conn.Ctx(), clog.ChannelIDKey, ID)
	channel := &Channel{
		Connection:   conn,
		RawStatement: rawStatement,
		ID:           ID,
		Context:      ctx,
	}
	return channel
}

func (channel *Channel) HandleStatement() {
	err, done := channel.validateAndRun()
	if err != nil {
		clog.Printf(channel, err.Error())
		channel.WriteErrorMessage(err)
	}
	// Remove this channel if we're done.
	if done {
		channel.Connection.removeChannel(channel)
	}
}

// validateAndRun returns an error if there was one, and a boolean
// representing whether this statement is done (i.e. whether a live query
// is still running on this channel)
func (channel *Channel) validateAndRun() (error, bool) {
	// Parse what was sent to us.
	statement, err := Parse(channel.RawStatement)
	if err != nil {
		return &ParseError{error: err}, true
	}

	// Validate statement.
	queryErr := channel.Connection.Database.ValidateStatement(statement)
	if queryErr != nil {
		return &ValidationError{error: queryErr}, true
	}
	return channel.run(statement)
}

// run runs the statement, returning and error if there was one
// and a boolean indicating whether the statement is "done"
// (only false if this is a live query)
func (channel *Channel) run(statement *Statement) (error, bool) {
	conn := channel.Connection
	// TODO: maybe move all these methods onto Channel?
	if statement.Select != nil {
		return conn.ExecuteTopLevelQuery(statement.Select, channel), !statement.Select.Live
	}
	if statement.Insert != nil {
		return conn.ExecuteInsert(statement.Insert, channel), true
	}
	if statement.CreateTable != nil {
		return conn.ExecuteCreateTable(statement.CreateTable, channel), true
	}
	if statement.Update != nil {
		return conn.ExecuteUpdate(statement.Update, channel), true
	}
	panic(fmt.Sprintf("unknown statement type %v", statement))
}

type ChannelMessage struct {
	// TODO: change this to ChannelID, as well as usages in JS
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
	Selection SelectResult
	QueryPath FlattenedQueryPath
}

type RecordUpdate struct {
	TableEvent *TableEvent
	QueryPath  FlattenedQueryPath
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
			QueryPath:  queryPath.Flatten(),
			TableEvent: update,
		},
	})
}

func (channel *Channel) writeMessage(message *MessageToClient) {
	channel.Connection.Messages <- &ChannelMessage{
		StatementID: channel.ID,
		Message:     message,
	}
}
