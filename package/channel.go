package treesql

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

// type MessageToClientType = int
// const MessageToClientType (
// 	ErrorMessage := iota,
// 	AckMessage,
// 	InitialResult,
// 	RecordUpdate,
// 	TableUpdate
// )

type MessageToClientType int

const (
	ErrorMessage MessageToClientType = iota
	AckMessage
	InitialResultMessage
	RecordUpdateMessage
	TableUpdateMessage
)

type MessageToClient struct {
	Type         MessageToClientType
	ErrorMessage *string
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
	channel.Connection.Messages <- &ChannelMessage{
		StatementID: channel.StatementID,
		Message:     message,
	}
}
