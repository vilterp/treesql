package treesql

import (
	"bufio"
	"fmt"
	"log"
	"strconv"

	"time"

	"github.com/boltdb/bolt"
)

func (conn *Connection) ExecuteQuery(query *Select) {
	// TODO: put all these reads in a transaction
	startTime := time.Now()
	tx, _ := conn.Database.BoltDB.Begin(false)
	writer := bufio.NewWriter(conn.ClientConn)
	execution := &QueryExecution{
		Connection:   conn,
		Query:        query,
		Transaction:  tx,
		ResultWriter: writer,
	}
	executeSelect(execution, query, nil)
	commitErr := tx.Rollback()
	fmt.Printf("read commit err:", commitErr)
	writer.WriteString("\n")
	writer.Flush()
	endTime := time.Now()
	log.Println("connection", conn.ID, "serviced query in", endTime.Sub(startTime))
}

// maybe this should be called transaction? idk
type QueryExecution struct {
	Connection   *Connection
	Query        *Select
	Transaction  *bolt.Tx
	ResultWriter *bufio.Writer
	Logger       *log.Logger
}

type Scope struct {
	table    *Table
	document *Record
}

// the question: read everything into memory and serialize at the end,
// or just write everything to the socket as we go?

type FilterCondition struct {
	InnerColumnName string
	OuterColumnName string
}

func executeSelect(ex *QueryExecution, query *Select, scope *Scope) {
	resultWriter := ex.ResultWriter
	tableSchema := ex.Connection.Database.Schema.Tables[query.Table]
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
			// e.g. one comment { blog_post: one blog_posts }
			// => inner: id, outer: post_id
			for _, columnSpec := range scope.table.Columns {
				if columnSpec.ReferencesColumn != nil &&
					columnSpec.ReferencesColumn.TableName == tableSchema.Name {
					filterCondition = &FilterCondition{
						InnerColumnName: tableSchema.PrimaryKey,
						OuterColumnName: columnSpec.Name,
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
	iterator, _ := ex.getTableIterator(query.Table)
	rowsRead := 0
	if query.Many {
		ex.ResultWriter.WriteString("[")
	}
	for {
		// get next doc
		nextDoc := iterator.Next()
		if nextDoc == nil {
			break
		}
		// decide if we want to write it
		if filterCondition != nil {
			if !recordMatchesFilter(filterCondition, nextDoc, scope.document) {
				continue
			}
		}
		if query.Where != nil {
			// again ignoring int vals for now...
			if nextDoc.GetField(query.Where.ColumnName).StringVal != query.Where.Value {
				continue
			}
		}
		if rowsRead == 1 && query.One {
			break
		}
		// start writing it
		if rowsRead > 0 {
			ex.ResultWriter.WriteString(",")
		}
		ex.ResultWriter.WriteString("{")
		// extract & write fields
		for selectionIdx, selection := range query.Selections {
			resultWriter.WriteString(fmt.Sprintf("\"%s\":", selection.Name))
			if selection.SubSelect != nil {
				// execute subquery
				executeSelect(ex, selection.SubSelect, &Scope{
					table:    tableSchema,
					document: nextDoc,
				})
			} else {
				// write field value out to socket
				columnSpec := columnsMap[selection.Name]
				switch columnSpec.Type {
				case TypeInt:
					val := nextDoc.GetField(columnSpec.Name).StringVal
					resultWriter.WriteString(fmt.Sprintf("%d", val))

				case TypeString:
					val := nextDoc.GetField(columnSpec.Name).StringVal
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

func recordMatchesFilter(condition *FilterCondition, innerRec *Record, outerRec *Record) bool {
	innerField := innerRec.GetField(condition.InnerColumnName)
	outerField := outerRec.GetField(condition.OuterColumnName)
	return *innerField == *outerField
}
