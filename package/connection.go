package treesql

import (
	"bufio"
	"fmt"
	"log"
	"net"

	"github.com/boltdb/bolt"
	"github.com/davecgh/go-spew/spew"
)

type Connection struct {
	ClientConn net.Conn
	ID         int
	Database   *Database
}

func (conn *Connection) Run() {
	log.Printf("connection id %d from %s\n", conn.ID, conn.ClientConn.RemoteAddr())
	for {
		// will listen for message to process ending in newline (\n)
		message, err := bufio.NewReader(conn.ClientConn).ReadString('\n')

		if err != nil {
			log.Printf("conn id %d terminated: %v\n", conn.ID, err)
			return
		}

		// parse what was sent to us
		statement, err := Parse(message)
		if err != nil {
			log.Println("connection", conn.ID, "parse error:", err)
			conn.ClientConn.Write([]byte(fmt.Sprintf("parse error: %s\n", err)))
			continue
		}

		// output message received
		// fmt.Print("SQL statement received:", spew.Sdump(statement))

		// validate statement
		queryErr := conn.Database.ValidateStatement(statement)
		if queryErr != nil {
			conn.ClientConn.Write([]byte(fmt.Sprintf("statement error: %s\n", queryErr)))
			log.Println("connection", conn.ID, "statement validation error", queryErr)
			continue
		}
		if statement.Select != nil {
			// execute query
			conn.ExecuteQuery(statement.Select)
		} else if statement.Insert != nil {
			conn.ExecuteInsert(statement.Insert)
		} else if statement.CreateTable != nil {
			conn.ExecuteCreateTable(statement.CreateTable)
		}
	}
}

// TODO: some other file, alongside executor.go? idk
func (conn *Connection) ExecuteInsert(insert *Insert) {
	// TODO: handle any errors
	conn.Database.BoltDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(insert.Table))
		table := conn.Database.Schema.Tables[insert.Table]
		record := table.NewRecord()
		for idx, value := range insert.Values {
			record.SetString(table.Columns[idx].Name, value)
		}
		key := record.GetField(table.PrimaryKey).StringVal
		bucket.Put([]byte(key), record.ToBytes())
		return nil
	})
	log.Println("connection", conn.ID, "handled insert")
	conn.ClientConn.Write([]byte("INSERT 1\n")) // heh
}

func (conn *Connection) ExecuteCreateTable(create *CreateTable) {
	fmt.Println("create table whooo", spew.Sdump(create))
}
