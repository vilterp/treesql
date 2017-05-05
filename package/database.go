package treesql

import (
	"errors"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

type Database struct {
	Schema                  *Schema
	BoltDB                  *bolt.DB
	QueryValidationRequests chan *QueryValidationRequest
	TableListeners          map[string]*TableListener
}

func Open(dataFile string) (*Database, error) {
	boltDB, openErr := bolt.Open(dataFile, 0600, nil)
	if openErr != nil {
		return nil, openErr
	}

	// TODO: load this from somewhere in data dir
	database := &Database{
		Schema:                  GetBuiltinSchema(),
		BoltDB:                  boltDB,
		QueryValidationRequests: make(chan *QueryValidationRequest),
		TableListeners:          map[string]*TableListener{},
	}
	database.EnsureBuiltinSchema()
	database.LoadUserSchema()
	database.MakeTableListeners()

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

func readSchema(db *bolt.DB) *Schema {
	fmt.Println("TODO: read schema from disk")
	return nil
}

// query validation
// this is more rigamarole than it would be in Erlang

type QueryValidationRequest struct {
	query        *Select
	responseChan chan error
}

func (db *Database) ValidateStatement(statement *Statement) error {
	if statement.Select != nil {
		return db.validateSelect(statement.Select)
	} else if statement.Insert != nil {
		return db.validateInsert(statement.Insert)
	} else if statement.CreateTable != nil {
		return db.validateCreateTable(statement.CreateTable)
	} else {
		return errors.New("unknown statement type")
	}
}

// func (db *Database) validateQuery(query *Select) error {
// 	responseChan := make(chan error)
// 	fmt.Println("about to send request")
// 	db.QueryValidationRequests <- &QueryValidationRequest{
// 		query:        query,
// 		responseChan: responseChan,
// 	}
// 	fmt.Println("sent request")
// 	return <-responseChan
// }

// func (db *Database) handleValidationRequest(request *QueryValidationRequest) {
// 	fmt.Printf("hello from handleValidationRequest")
// 	request.responseChan <- db.ValidateSelect(request.query)
// }

func (db *Database) validateInsert(insert *Insert) error {
	// does table exist
	tableSpec, ok := db.Schema.Tables[insert.Table]
	if !ok {
		return &NoSuchTable{TableName: insert.Table}
	}
	// can't insert into builtins
	if insert.Table == "__tables__" || insert.Table == "__columns__" {
		return &BuiltinWriteAttempt{TableName: insert.Table}
	}
	// right # fields (TODO: validate types)
	wanted := len(tableSpec.Columns)
	got := len(insert.Values)
	if wanted != got {
		return &InsertWrongNumFields{TableName: insert.Table, Wanted: wanted, Got: got}
	}
	return nil
}

// want to not export this and do it via the server, but...
func (db *Database) validateSelect(query *Select) error {
	// does table exist?
	_, ok := db.Schema.Tables[query.Table]
	if !ok && query.Table != "__tables__" && query.Table != "__columns__" {
		return &NoSuchTable{TableName: query.Table}
	}
	// do columns exist / are subqueries valid?
	// TODO: dedup
	for _, selection := range query.Selections {
		if selection.SubSelect != nil {
			err := db.validateSelect(selection.SubSelect)
			if err != nil {
				return err
			}
		} else {
			// hoo, I miss filter
			hasColumn := false
			for _, column := range db.Schema.Tables[query.Table].Columns {
				if column.Name == selection.Name {
					hasColumn = true
				}
			}
			if !hasColumn {
				return &NoSuchColumn{TableName: query.Table, ColumnName: selection.Name}
			}
		}
	}
	return nil
}

func (db *Database) validateCreateTable(create *CreateTable) error {
	// does table already exist?
	_, ok := db.Schema.Tables[create.Name]
	if ok {
		return &TableAlreadyExists{TableName: create.Name}
	}
	// types are real
	for _, column := range create.Columns {
		knownType := column.TypeName == "string" || column.TypeName == "int"
		if !knownType {
			return &NonexistentType{TypeName: column.TypeName}
		}
	}
	// only one primary key
	primaryKeyCount := 0
	for _, column := range create.Columns {
		if column.PrimaryKey {
			primaryKeyCount++
		}
	}
	if primaryKeyCount != 1 {
		return &WrongNoPrimaryKey{Count: primaryKeyCount}
	}
	// referenced table exists
	// TODO: column same type as primary key
	for _, column := range create.Columns {
		if column.References != nil {
			_, tableExists := db.Schema.Tables[*column.References]
			if !tableExists {
				return &NoSuchTable{TableName: *column.References}
			}
		}
	}
	// TODO: dedup column names
	return nil
}
