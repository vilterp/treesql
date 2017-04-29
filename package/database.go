package treesql

import (
	sophia "github.com/pzhin/go-sophia"
)

type Database struct {
	Schema                  *Schema
	Dbs                     map[string]*sophia.Database
	queryValidationRequests chan *QueryValidationRequest
}

func Open(dataDir string) (*Database, error) {
	env, _ := sophia.NewEnvironment()
	env.Set("sophia.path", dataDir)

	// TODO: load this from somewhere in data dir
	testSchema := GetTestSchema()
	database := &Database{
		Schema: GetTestSchema(),
		Dbs:    map[string]*sophia.Database{},
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
		database.Dbs[tableName] = newDb
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

// query validation

func (db *Database) handleValidationRequest(request *QueryValidationRequest) {
	request.responseChan <- nil
}

type QueryValidationRequest struct {
	query        *Select
	responseChan chan error
}

func (db *Database) ValidateQuery(query *Select) error {
	responseChan := make(chan error)
	db.queryValidationRequests <- &QueryValidationRequest{
		query:        query,
		responseChan: responseChan,
	}
	return <-responseChan
}
