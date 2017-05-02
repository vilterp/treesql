package treesql

import (
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

type Database struct {
	Schema                  *Schema
	boltDB                  *bolt.DB
	queryValidationRequests chan *QueryValidationRequest
}

func Open(dataFile string) (*Database, error) {
	boltDB, openErr := bolt.Open(dataFile, 0600, nil)
	if openErr != nil {
		return nil, openErr
	}

	// TODO: load this from somewhere in data dir
	testSchema := GetTestSchema()
	database := &Database{
		Schema:                  GetTestSchema(),
		boltDB:                  boltDB,
		queryValidationRequests: make(chan *QueryValidationRequest),
	}

	// open tables
	boltDB.Update(func(tx *bolt.Tx) error {
		for tableName, _ := range testSchema.Tables {
			_, bucketErr := tx.CreateBucketIfNotExists([]byte(tableName))
			if bucketErr != nil {
				return bucketErr
			}
		}
		return nil
	})

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
	err := db.boltDB.Close()
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
	_, ok := db.Schema.Tables[query.Table]
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
