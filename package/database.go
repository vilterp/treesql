package treesql

import (
	"errors"
	"log"

	"github.com/boltdb/bolt"
)

type Database struct {
	Schema                  *Schema
	BoltDB                  *bolt.DB
	QueryValidationRequests chan *QueryValidationRequest
	NextConnectionID        int
}

func Open(dataFile string) (*Database, error) {
	boltDB, openErr := bolt.Open(dataFile, 0600, nil)
	if openErr != nil {
		return nil, openErr
	}

	// TODO: load this from somewhere in data dir
	database := &Database{
		Schema:                  EmptySchema(),
		BoltDB:                  boltDB,
		QueryValidationRequests: make(chan *QueryValidationRequest),
		NextConnectionID:        0,
	}
	database.AddBuiltinSchema()
	database.EnsureBuiltinSchema()
	database.LoadUserSchema()

	// serve query validation requests
	// TODO: a `select` here for schema changes
	// serializing access to the schema
	// go func() {
	// 	for {
	// 		query := <-database.QueryValidationRequests
	// 		database.handleValidationRequest(query)
	// 	}
	// }()

	return database, nil
}

func (db *Database) Close() {
	log.Println("Closing storage layer...")
	err := db.BoltDB.Close()
	if err != nil {
		log.Printf("error closing storage layer:", err)
	}
}

// query validation
// this is more rigamarole than it would be in Erlang

type QueryValidationRequest struct {
	query        *Select
	responseChan chan error
}

func (db *Database) ValidateStatement(statement *Statement) error {
	if statement.Select != nil {
		return db.validateSelect(statement.Select, nil)
	} else if statement.Insert != nil {
		return db.validateInsert(statement.Insert)
	} else if statement.CreateTable != nil {
		return db.validateCreateTable(statement.CreateTable)
	} else if statement.Update != nil {
		return db.validateUpdate(statement.Update)
	} else {
		return errors.New("unknown statement type")
	}
}

func (db *Database) PushTableEvent(tableName string, oldRecord *Record, newRecord *Record) {
	db.Schema.Tables[tableName].LiveQueryInfo.TableEvents <- &TableEvent{
		TableName: tableName,
		OldRecord: oldRecord,
		NewRecord: newRecord,
	}
}
