package treesql

import (
	"context"
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
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

// TODO: this is pretty ugly. Maybe it should be embedded in the value?
func (cm *ChannelMessage) GetCaller() lang.Caller {
	if cm.Message.InitialResultMessage != nil {
		return cm.Message.InitialResultMessage.Caller
	}
	if cm.Message.AckMessage != nil {
		return nil
	}
	if cm.Message.ErrorMessage != nil {
		return nil
	}
	panic(fmt.Sprintf("can't get caller for %+v", cm.Message))
}

func (cm *ChannelMessage) ToVal() lang.Value {
	return lang.NewVRecord(map[string]lang.Value{
		"StatementID": lang.NewVInt(cm.StatementID),
		"Message":     cm.Message.ToVal(),
	})
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

func (m *MessageToClientType) String() string {
	switch *m {
	case ErrorMessage:
		return "error"
	case AckMessage:
		return "ack"
	case InitialResultMessage:
		return "initial_result"
	case RecordUpdateMessage:
		return "record_update"
	case TableUpdateMessage:
		return "table_update"
	}
	panic(fmt.Errorf("unknown type %d", *m))
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

func (mtc *MessageToClient) ToVal() lang.Value {
	vals := map[string]lang.Value{}
	vals["type"] = lang.NewVString(mtc.Type.String())
	if mtc.ErrorMessage != nil {
		vals["error"] = lang.NewVString(*mtc.ErrorMessage)
	}
	if mtc.AckMessage != nil {
		vals["ack"] = lang.NewVString(*mtc.AckMessage)
	}
	if mtc.InitialResultMessage != nil {
		vals["initial_result"] = mtc.InitialResultMessage.ToVal()
	}
	if mtc.RecordUpdateMessage != nil {
		vals["record_update"] = mtc.RecordUpdateMessage.ToVal()
	}
	if mtc.TableUpdateMessage != nil {
		vals["table_update"] = mtc.TableUpdateMessage.ToVal()
	}
	return lang.NewVRecord(vals)
}

type InitialResult struct {
	Type   lang.Type
	Value  lang.Value
	Caller lang.Caller
}

func (ir *InitialResult) ToVal() lang.Value {
	return lang.NewVRecord(map[string]lang.Value{
		// TODO: send Type in a structured format, not pretty-prim
		"Type":  lang.NewVString(ir.Value.GetType().Format().String()),
		"Value": ir.Value,
	})
}

type TableUpdate struct {
	Selection lang.Value
	QueryPath FlattenedQueryPath
}

func (tu *TableUpdate) ToVal() lang.Value {
	panic("unimplemented")
}

type RecordUpdate struct {
	TableEvent *TableEvent
	QueryPath  FlattenedQueryPath
}

func (ru *RecordUpdate) ToVal() lang.Value {
	panic("unimplemented")
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
