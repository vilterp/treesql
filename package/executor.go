package treesql

import (
	"bufio"
	"fmt"

	sophia "github.com/pzhin/go-sophia"
)

func ExecuteQuery(conn *Connection, query *Select) {
	writer := bufio.NewWriter(conn.ClientConn)
	executeSelect(conn, writer, query, nil)
	writer.WriteString("\n")
	writer.Flush()
}

type Scope struct {
	table    *Table
	document *sophia.Document
}

// the question: read everything into memory and serialize at the end,
// or just write everything to the socket as we go?

func executeSelect(conn *Connection, resultWriter *bufio.Writer, query *Select, scope *Scope) {
	// TODO: really have to learn how to use resultWriter...
	table := conn.Database.Dbs[query.Table]
	tableSchema := conn.Database.Schema.Tables[query.Table]
	doc := table.Document()
	cursor, _ := table.Cursor(doc)
	rowsRead := 0
	resultWriter.WriteString("[")
	for {
		nextDoc := cursor.Next()
		if nextDoc == nil {
			break
		}
		if rowsRead > 0 {
			resultWriter.WriteString(",")
		}
		resultWriter.WriteString("{")
		// get schema fields into a map (maybe it should be this in the schema? idk)
		columnsMap := map[string]*Column{}
		for _, column := range tableSchema.Columns {
			columnsMap[column.Name] = column
		}
		// extract fields
		for selectionIdx, selection := range query.Selections {
			resultWriter.WriteString(fmt.Sprintf("\"%s\":", selection.Name))
			if selection.SubSelect != nil {
				executeSelect(conn, resultWriter, selection.SubSelect, &Scope{
					table:    tableSchema,
					document: nextDoc,
				})
			} else {
				columnSpec := columnsMap[selection.Name]
				switch columnSpec.Type {
				case TypeInt:
					val := nextDoc.GetInt(columnSpec.Name)
					resultWriter.WriteString(fmt.Sprintf("%d", val))

				case TypeString:
					size := 0
					val := nextDoc.GetString(columnSpec.Name, &size)
					resultWriter.WriteString(fmt.Sprintf("\"%s\"", val))
				}
			}
			if selectionIdx < len(query.Selections)-1 {
				resultWriter.WriteString(",")
			}
		}
		rowsRead++
		resultWriter.WriteString("}")
	}
	resultWriter.WriteString("]")
}
