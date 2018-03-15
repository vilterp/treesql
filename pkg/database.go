package treesql

import (
	"context"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
)

type Database struct {
	Schema           *Schema
	BoltDB           *bolt.DB
	Connections      map[ConnectionID]*Connection
	NextConnectionID int

	Ctx     context.Context
	Metrics *Metrics
}

func NewDatabase(dataFile string) (*Database, error) {
	boltDB, openErr := bolt.Open(dataFile, 0600, nil)
	if openErr != nil {
		return nil, openErr
	}

	ctx := context.Background()

	// TODO: load this from somewhere in data dir
	database := &Database{
		Schema:           EmptySchema(),
		BoltDB:           boltDB,
		Connections:      make(map[ConnectionID]*Connection),
		NextConnectionID: 0,
		Ctx:              ctx,
	}
	database.AddBuiltinSchema()
	database.EnsureBuiltinSchema()
	database.LoadUserSchema()

	database.Metrics = NewMetrics(database)

	return database, nil
}

// AddConnection connects a websocket to the database, s.t. the database
// will interact with the connection.
func (db *Database) AddConnection(wsConn *websocket.Conn) {
	conn := NewConnection(wsConn, db, db.NextConnectionID)
	db.NextConnectionID++
	db.Connections[conn.ID] = conn
	conn.HandleStatements()
}

func (db *Database) removeConn(conn *Connection) {
	delete(db.Connections, conn.ID)
	for _, table := range db.Schema.Tables {
		table.removeListenersForConn(conn.ID)
	}
}

func (db *Database) Close() error {
	return db.BoltDB.Close()
}

// query validation
// this is more rigamarole than it would be in Erlang

type QueryValidationRequest struct {
	query        *Select
	responseChan chan error
}

func (db *Database) ValidateStatement(statement *Statement) error {
	if statement.Select != nil {
		// Validates during the planning phase
		// TODO: replace entire `ValidateStatement` with planning
		return nil
	}
	if statement.Insert != nil {
		return db.validateInsert(statement.Insert)
	}
	if statement.CreateTable != nil {
		return db.validateCreateTable(statement.CreateTable)
	}
	if statement.Update != nil {
		return db.validateUpdate(statement.Update)
	}
	return errors.New("unknown statement type")
}

func (db *Database) PushTableEvent(
	channel *Channel, // originating channel
	tableName string,
	oldRecord *Record,
	newRecord *Record,
) {
	db.Schema.Tables[tableName].LiveQueryInfo.TableEvents <- &TableEvent{
		TableName: tableName,
		OldRecord: oldRecord,
		NewRecord: newRecord,
		channel:   channel,
	}
}
