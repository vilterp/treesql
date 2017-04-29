package treesql

import (
	"encoding/json"
	"fmt"
	"net"
)

type Connection struct {
	ClientConn net.Conn
	ID         int
	Database   *Database
}

func ExecuteQuery(conn *Connection, query *Select) {
	resultWriter := conn.ClientConn
	// TODO: really have to learn how to use bufio...
	db, ok := conn.Database.Dbs[query.From.Table]
	schema, ok := conn.Database.Schema.Tables[query.From.Table]
	if !ok {
		errorMsg := fmt.Sprintf("nonexistent table: %s", query.From.Table)
		resultWriter.Write([]byte(errorMsg + "\n"))
		resultWriter.Write([]byte("done"))
		fmt.Println(errorMsg)
		return
	}
	doc := db.Document()
	cursor, _ := db.Cursor(doc)
	rowsRead := 0
	resultWriter.Write([]byte("["))
	for {
		nextDoc := cursor.Next()
		if nextDoc == nil {
			break
		}
		rowsRead++
		fmt.Println("next doc")
		if rowsRead > 0 {
			resultWriter.Write([]byte(","))
		}
		// extract fields
		output := map[string]interface{}{}
		for _, columnSpec := range schema.Columns {
			fmt.Println("extract ")
			switch columnSpec.Type {
			case TypeInt:
				output[columnSpec.Name] = nextDoc.GetInt(columnSpec.Name)

			case TypeString:
				size := 0
				output[columnSpec.Name] = nextDoc.GetString(columnSpec.Name, &size)
			}
		}
		inJSON, _ := json.Marshal(output)
		resultWriter.Write(inJSON)
	}
	resultWriter.Write([]byte("]\n"))
	fmt.Println("wrote response")
}
