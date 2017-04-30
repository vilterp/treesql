package treesql

import (
	"fmt"

	sophia "github.com/pzhin/go-sophia"
)

func ExecuteQuery(conn *Connection, query *Select) {
	executeSelect(conn, query, nil)
	conn.ClientConn.Write([]byte("\n"))
}

type Scope struct {
	table    *Table
	document *sophia.Document
}

// the question: read everything into memory and serialize at the end,
// or just write everything to the socket as we go?

func executeSelect(conn *Connection, query *Select, scope *Scope) {
	resultWriter := conn.ClientConn
	// TODO: really have to learn how to use bufio...
	table := conn.Database.Dbs[query.Table]
	tableSchema := conn.Database.Schema.Tables[query.Table]
	doc := table.Document()
	cursor, _ := table.Cursor(doc)
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
		resultWriter.Write([]byte("{"))
		// get schema fields into a map (maybe it should be this in the schema? idk)
		columnsMap := map[string]*Column{}
		for _, column := range tableSchema.Columns {
			columnsMap[column.Name] = column
		}
		// extract fields
		for selectionIdx, selection := range query.Selections {
			resultWriter.Write([]byte(fmt.Sprintf("\"%s\":", selection.Name)))
			if selection.SubSelect != nil {
				executeSelect(conn, selection.SubSelect, &Scope{
					table:    tableSchema,
					document: nextDoc,
				})
			} else {
				columnSpec := columnsMap[selection.Name]
				switch columnSpec.Type {
				case TypeInt:
					val := nextDoc.GetInt(columnSpec.Name)
					resultWriter.Write([]byte(fmt.Sprintf("%d", val)))

				case TypeString:
					size := 0
					val := nextDoc.GetString(columnSpec.Name, &size)
					resultWriter.Write([]byte(fmt.Sprintf("\"%s\"", val)))
				}
			}
			if selectionIdx < len(query.Selections)-1 {
				resultWriter.Write([]byte(","))
			}
		}
		rowsRead++
		resultWriter.Write([]byte("}"))
	}
	resultWriter.Write([]byte("]"))
}
