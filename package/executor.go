package treesql

import (
	"encoding/json"
	"fmt"
)

func ExecuteQuery(conn *Connection, query *Select) {
	resultWriter := conn.ClientConn
	// TODO: really have to learn how to use bufio...
	db := conn.Database.Dbs[query.Table]
	schema := conn.Database.Schema.Tables[query.Table]
	doc := db.Document()
	cursor, _ := db.Cursor(doc)
	rowsRead := 0
	resultWriter.Write([]byte("["))
	for {
		nextDoc := cursor.Next()
		if nextDoc == nil {
			break
		}
		if rowsRead > 0 {
			resultWriter.Write([]byte(","))
		}
		// extract fields
		output := map[string]interface{}{}
		for _, columnSpec := range schema.Columns {
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
		rowsRead++
	}
	resultWriter.Write([]byte("]\n"))
	fmt.Println("wrote", rowsRead, "rows")
}
