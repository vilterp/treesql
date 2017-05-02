package treesql

import (
	"bufio"
	"fmt"
	"strconv"

	sophia "github.com/pzhin/go-sophia"
)

func ExecuteQuery(conn *Connection, query *Select) {
	// TODO: put all these reads in a transaction
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

type FilterCondition struct {
	InnerColumnName string
	OuterColumnName string
}

func executeSelect(conn *Connection, resultWriter *bufio.Writer, query *Select, scope *Scope) {
	tableSchema := conn.Database.Schema.Tables[query.Table]
	// if we're an inner loop, figure out a condition for our loop
	var filterCondition *FilterCondition
	if scope != nil {
		if query.Many {
			// find reference from inner table to outer table
			// TODO: this is the kind of thing that should be done in a query planner,
			// not in every nested loop
			for _, columnSpec := range tableSchema.Columns {
				if columnSpec.ReferencesColumn != nil &&
					columnSpec.ReferencesColumn.TableName == scope.table.Name {
					filterCondition = &FilterCondition{
						InnerColumnName: columnSpec.Name,
						OuterColumnName: scope.table.PrimaryKey,
					}
				}
			}
		} else {
			// find reference from outer table to inner table
			for _, columnSpec := range scope.table.Columns {
				if columnSpec.ReferencesColumn != nil &&
					columnSpec.ReferencesColumn.TableName == tableSchema.Name {
					filterCondition = &FilterCondition{
						InnerColumnName: columnSpec.Name,
						OuterColumnName: tableSchema.PrimaryKey,
					}
				}
			}
		}
	}
	// get schema fields into a map (maybe it should be this in the schema? idk)
	columnsMap := map[string]*Column{}
	for _, column := range tableSchema.Columns {
		columnsMap[column.Name] = column
	}
	// start iterating
	iterator, _ := conn.Database.getTableIterator(query.Table)
	rowsRead := 0
	if query.Many {
		resultWriter.WriteString("[")
	}
	for {
		// get next doc
		nextDoc := iterator.Next()
		if nextDoc == nil {
			break
		}
		// decide if we want to write it
		if filterCondition != nil {
			if !docMatchesFilter(filterCondition, nextDoc, scope.document) {
				continue
			}
		}
		if query.Where != nil {
			whereSize := 0
			if nextDoc.GetString(query.Where.ColumnName, &whereSize) != query.Where.Value {
				continue
			}
		}
		// start writing it
		if rowsRead > 0 {
			resultWriter.WriteString(",")
		}
		resultWriter.WriteString("{")
		// extract & write fields
		for selectionIdx, selection := range query.Selections {
			resultWriter.WriteString(fmt.Sprintf("\"%s\":", selection.Name))
			if selection.SubSelect != nil {
				// execute subquery
				executeSelect(conn, resultWriter, selection.SubSelect, &Scope{
					table:    tableSchema,
					document: nextDoc,
				})
			} else {
				// write field
				columnSpec := columnsMap[selection.Name]
				switch columnSpec.Type {
				case TypeInt:
					val := nextDoc.GetInt(columnSpec.Name)
					resultWriter.WriteString(fmt.Sprintf("%d", val))

				case TypeString:
					size := 0
					val := nextDoc.GetString(columnSpec.Name, &size)
					resultWriter.WriteString(strconv.Quote(val))
				}
			}
			if selectionIdx < len(query.Selections)-1 {
				resultWriter.WriteString(",")
			}
		}
		rowsRead++
		resultWriter.WriteString("}")
	}
	iterator.Close()
	if query.Many {
		resultWriter.WriteString("]")
	}
}

func docMatchesFilter(condition *FilterCondition, innerDoc *sophia.Document, outerDoc *sophia.Document) bool {
	// again, ignoring non-string types for now...
	innerSize := 0
	innerField := innerDoc.GetString(condition.InnerColumnName, &innerSize)
	outerSize := 0
	outerField := outerDoc.GetString(condition.OuterColumnName, &outerSize)
	return innerField == outerField
}
