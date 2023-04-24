package treesql

import (
	"context"
	"errors"

	"github.com/boltdb/bolt"
	"github.com/gorilla/websocket"
)

type Database struct {
	schema           *schema
	boltDB           *bolt.DB
	connections      map[connectionID]*connection
	nextConnectionID int

	ctx     context.Context
	metrics *metrics
}

func NewDatabase(dataFile string) (*Database, error) {
	boltDB, openErr := bolt.Open(dataFile, 0600, nil)
	if openErr != nil {
		return nil, openErr
	}

	ctx := context.Background()

	// TODO: load this from somewhere in data dir
	database := &Database{
		schema:           emptySchema(),
		boltDB:           boltDB,
		connections:      make(map[connectionID]*connection),
		nextConnectionID: 0,
		ctx:              ctx,
	}
	database.addBuiltinSchema()
	database.ensureBuiltinSchema()
	database.loadUserSchema()

	database.metrics = newMetrics(database)

	return database, nil
}

// addConnection connects a websocket to the database, s.t. the database
// will interact with the connection.
func (db *Database) addConnection(wsConn *websocket.Conn) {
	conn := newConnection(wsConn, db, db.nextConnectionID)
	db.nextConnectionID++
	db.connections[conn.id] = conn
	conn.handleStatements()
}

func (db *Database) removeConn(conn *connection) {
	delete(db.connections, conn.id)
	for _, table := range db.schema.tables {
		table.removeListenersForConn(conn.id)
	}
}

func (db *Database) Close() error {
	return db.boltDB.Close()
}

func (db *Database) validateStatement(statement *Statement) error {
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

func (db *Database) pushTableEvent(
	channel *channel, // originating channel
	tableName string,
	oldRecord *record,
	newRecord *record,
) {
	db.schema.tables[tableName].liveQueryInfo.TableEvents <- &tableEvent{
		TableName: tableName,
		OldRecord: oldRecord,
		NewRecord: newRecord,
		channel:   channel,
	}
}
