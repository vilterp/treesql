package treesql

import (
	"fmt"
	"log"

	sophia "github.com/pzhin/go-sophia"
)

type Database struct {
	Schema                  *Schema
	Env                     *sophia.Environment
	Tables                  map[string]*sophia.Database
	queryValidationRequests chan *QueryValidationRequest
}

func Open(dataDir string) (*Database, error) {
	env, _ := sophia.NewEnvironment()
	env.Set("sophia.path", dataDir)

	// TODO: load this from somewhere in data dir
	testSchema := GetTestSchema()
	database := &Database{
		Schema: GetTestSchema(),
		Tables: map[string]*sophia.Database{},
		Env:    env,
	}

	// open databases
	for tableName, table := range testSchema.Tables {
		newDb, err := env.NewDatabase(&sophia.DatabaseConfig{
			Name:   tableName,
			Schema: table.ToSophiaSchema(),
		})
		if err != nil {
			return database, err
		}
		database.Tables[tableName] = newDb
	}
	env.Open()

	// serve query validation requests
	// TODO: a `select` here for schema changes
	// serializing access to the schema
	go func() {
		for {
			query := <-database.queryValidationRequests
			database.handleValidationRequest(query)
		}
	}()

	return database, nil
}

func (db *Database) Close() {
	log.Println("Closing storage layer...")
	err := db.Env.Close()
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

func (db *Database) ValidateQuery(query *Select) error {
	responseChan := make(chan error)
	fmt.Println("about to send request")
	db.queryValidationRequests <- &QueryValidationRequest{
		query:        query,
		responseChan: responseChan,
	}
	fmt.Println("sent request")
	return <-responseChan
}

func (db *Database) handleValidationRequest(request *QueryValidationRequest) {
	fmt.Printf("hello from handleValidationRequest")
	request.responseChan <- db.ValidateSelect(request.query)
}

// want to not export this and do it via the server, but...
func (db *Database) ValidateSelect(query *Select) error {
	// does table exist?
	_, ok := db.Tables[query.Table]
	if !ok && query.Table != "__tables__" && query.Table != "__columns__" {
		return &NoSuchTable{TableName: query.Table}
	}
	// do columns exist / are subqueries valid?
	// TODO: dedup
	for _, selection := range query.Selections {
		if selection.SubSelect != nil {
			err := db.ValidateSelect(selection.SubSelect)
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
