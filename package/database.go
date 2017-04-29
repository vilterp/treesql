package treesql

import (
	sophia "github.com/pzhin/go-sophia"
)

type Database struct {
	Schema *Schema
	Dbs    map[string]*sophia.Database
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

	return database, nil
}
