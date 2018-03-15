package treesql

import (
	"context"
	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
	clog "github.com/vilterp/treesql/pkg/log"
)

type channelID int

type channel struct {
	connection   *connection
	rawStatement string
	id           int // unique with containing connection

	context context.Context
}

func (channel *channel) Ctx() context.Context {
	return channel.context
}

func newChannel(rawStatement string, ID int, conn *connection) *channel {
	ctx := context.WithValue(conn.Ctx(), clog.ChannelIDKey, ID)
	channel := &channel{
		connection:   conn,
		rawStatement: rawStatement,
		id:           ID,
		context:      ctx,
	}
	return channel
}

func (channel *channel) handleStatement() {
	done, err := channel.validateAndRun()
	if err != nil {
		clog.Printf(channel, err.Error())
		channel.writeErrorMessage(err)
	}
	// Remove this channel if we're done.
	if done {
		channel.connection.removeChannel(channel)
	}
}

// validateAndRun returns an error if there was one, and a boolean
// representing whether this statement is done (i.e. whether a live query
// is still running on this channel)
func (channel *channel) validateAndRun() (bool, error) {
	// Parse what was sent to us.
	statement, err := Parse(channel.rawStatement)
	if err != nil {
		return true, &parseError{error: err}
	}

	// Validate statement.
	queryErr := channel.connection.database.validateStatement(statement)
	if queryErr != nil {
		return true, &validationError{error: queryErr}
	}
	return channel.run(statement)
}

// run runs the statement, returning and error if there was one
// and a boolean indicating whether the statement is "done"
// (only false if this is a live query)
func (channel *channel) run(statement *Statement) (bool, error) {
	conn := channel.connection
	// TODO: maybe move all these methods onto channel?
	if statement.Select != nil {
		return !statement.Select.Live, conn.executeTopLevelQuery(statement.Select, channel)
	}
	if statement.Insert != nil {
		return true, conn.executeInsert(statement.Insert, channel)
	}
	if statement.CreateTable != nil {
		return true, conn.executeCreateTable(statement.CreateTable, channel)
	}
	if statement.Update != nil {
		return true, conn.executeUpdate(statement.Update, channel)
	}
	panic(fmt.Sprintf("unknown statement type %v", statement))
}

type ChannelMessage struct {
	// TODO: change this to channelID, as well as usages in JS
	StatementID int
	Message     *MessageToClient
}

// TODO: this is pretty ugly. Maybe it should be embedded in the value?
func (cm *ChannelMessage) getCaller() lang.Caller {
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

func (cm *ChannelMessage) toVal() lang.Value {
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
		vals["record_update"] = mtc.RecordUpdateMessage.toVal()
	}
	if mtc.TableUpdateMessage != nil {
		vals["table_update"] = mtc.TableUpdateMessage.toVal()
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
		// TODO: send typ in a structured format, not pretty-prim
		"typ":   lang.NewVString(ir.Value.GetType().Format().String()),
		"Value": ir.Value,
	})
}

type TableUpdate struct {
	Selection lang.Value
	QueryPath flattenedQueryPath
}

func (tu *TableUpdate) toVal() lang.Value {
	panic("unimplemented")
}

type RecordUpdate struct {
	TableEvent *tableEvent
	QueryPath  flattenedQueryPath
}

func (ru *RecordUpdate) toVal() lang.Value {
	panic("unimplemented")
}

func (channel *channel) writeErrorMessage(err error) {
	errStr := err.Error()
	channel.writeMessage(&MessageToClient{
		Type:         ErrorMessage,
		ErrorMessage: &errStr,
	})
}

func (channel *channel) writeAckMessage(message string) {
	channel.writeMessage(&MessageToClient{
		Type:       AckMessage,
		AckMessage: &message,
	})
}

func (channel *channel) writeInitialResult(result *InitialResult) {
	channel.writeMessage(&MessageToClient{
		Type:                 InitialResultMessage,
		InitialResultMessage: result,
	})
}

func (channel *channel) writeTableUpdate(update *TableUpdate) {
	channel.writeMessage(&MessageToClient{
		Type:               TableUpdateMessage,
		TableUpdateMessage: update,
	})
}

func (channel *channel) writeRecordUpdate(update *tableEvent, queryPath *queryPath) {
	channel.writeMessage(&MessageToClient{
		Type: RecordUpdateMessage,
		RecordUpdateMessage: &RecordUpdate{
			QueryPath:  queryPath.flatten(),
			TableEvent: update,
		},
	})
}

func (channel *channel) writeMessage(message *MessageToClient) {
	channel.connection.messages <- &ChannelMessage{
		StatementID: channel.id,
		Message:     message,
	}
}
