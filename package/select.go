package treesql

import (
	"errors"
	"log"
	"time"

	"github.com/boltdb/bolt"
)

// want to not export this and do it via the server, but...
func (db *Database) validateSelect(query *Select, tableAbove *string) error {
	// does table exist?
	_, ok := db.Schema.Tables[query.Table]
	if !ok && query.Table != "__tables__" && query.Table != "__columns__" {
		return &NoSuchTable{TableName: query.Table}
	}
	// is there a reference from this table to table above or vice versa?
	if tableAbove != nil {
		var fromTable string
		var toTable string
		// ugh I want f*cking checked switch statements
		if query.Many {
			// reference from inner to outer
			fromTable = query.Table
			toTable = *tableAbove
		} else if query.One {
			// reference from outer to inner
			fromTable = *tableAbove
			toTable = query.Table
		}
		referenceFound := false
		for _, column := range db.Schema.Tables[fromTable].Columns {
			if column.ReferencesColumn != nil {
				if column.ReferencesColumn.TableName == toTable {
					referenceFound = true
				}
			}
		}
		if !referenceFound {
			return &NoReferenceForJoin{
				FromTable: fromTable,
				ToTable:   toTable,
			}
		}
	}
	// do columns exist / are subqueries valid?
	// TODO: dedup
	for _, selection := range query.Selections {
		if selection.SubSelect != nil {
			err := db.validateSelect(selection.SubSelect, &query.Table)
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

func (conn *Connection) ExecuteQuery(query *Select, queryID int, channel *Channel) {
	// TODO: put all these reads in a transaction
	startTime := time.Now()
	tx, _ := conn.Database.BoltDB.Begin(false)
	execution := &QueryExecution{
		Channel:     channel,
		Query:       query,
		Transaction: tx,
	}
	result, selectErr := executeSelect(execution, query, nil)
	if selectErr != nil {
		channel.WriteMessage(selectErr.Error())
		log.Println("connection", conn.ID, "query error:", selectErr.Error())
	} else {
		channel.WriteMessage(result)
	}
	commitErr := tx.Rollback()
	if commitErr != nil {
		log.Println("read commit err:", commitErr)
	}
	endTime := time.Now()

	log.Println(
		"connection", conn.ID, "serviced query", queryID, "in", endTime.Sub(startTime),
		"live:", query.Live,
	) // TODO: structured logging XD
}

// maybe this should be called transaction? idk
type QueryExecution struct {
	Channel     *Channel
	Query       *Select
	Transaction *bolt.Tx
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

// responsibility of serializer to write result[0] for ONE queries
type SelectResult [](map[string]interface{})

func executeSelect(ex *QueryExecution, query *Select, scope *Scope) (SelectResult, error) {
	result := make([](map[string]interface{}), 0)
	database := ex.Channel.Connection.Database
	tableSchema := database.Schema.Tables[query.Table]
	// if we're an inner loop, figure out a condition for our loop
	var filterCondition *FilterCondition
	if scope != nil {
		filterCondition = getFilterCondition(query, tableSchema, scope)

		if ex.Query.Live {
			innerTable := database.Schema.Tables[query.Table]
			database.TableListeners[innerTable.Name].SubscriberEvents <- &SubscriberEvent{
				ColumnName:     filterCondition.InnerColumnName,
				QueryExecution: ex,
				Value:          scope.document.GetField(filterCondition.OuterColumnName),
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
	for {
		// get next doc
		record := iterator.Next()
		if record == nil {
			break
		}
		// decide if we want to write it
		if filterCondition != nil {
			if !recordMatchesFilter(filterCondition, record, scope.document) {
				continue
			}
		}
		if query.Where != nil {
			// again ignoring int vals for now...
			if record.GetField(query.Where.ColumnName).StringVal != query.Where.Value {
				continue
			}
		}
		if rowsRead == 1 && query.One {
			break // TODO: actually error if > 1
		}
		// we are interested in this record... let's subscribe to it
		if ex.Query.Live {
			database.TableListeners[tableSchema.Name].SubscriberEvents <- &SubscriberEvent{
				ColumnName:     tableSchema.PrimaryKey,
				QueryExecution: ex,
				Value:          record.GetField(tableSchema.PrimaryKey),
			}
		}
		// start writing it
		recordResults := map[string]interface{}{}
		// extract & write fields
		for _, selection := range query.Selections {
			if selection.SubSelect != nil {
				// execute subquery
				nextScope := &Scope{
					table:    tableSchema,
					document: record,
				}
				subselectResult, subselectErr := executeSelect(ex, selection.SubSelect, nextScope)
				if subselectErr != nil {
					return nil, subselectErr
				}
				recordResults[selection.Name] = subselectResult
			} else {
				// write field value out to socket
				columnSpec := columnsMap[selection.Name]
				switch columnSpec.Type {
				case TypeInt:
					val := record.GetField(columnSpec.Name).StringVal
					recordResults[columnSpec.Name] = val

				case TypeString:
					val := record.GetField(columnSpec.Name).StringVal
					recordResults[columnSpec.Name] = val
				}
			}
		}
		rowsRead++
		result = append(result, recordResults)
	}
	iterator.Close()
	if query.One && rowsRead == 0 {
		return nil, errors.New("error: requested one row, but none found")
		// TODO: this could be in the middle of a result set, lol
	}
	return result, nil
}

func recordMatchesFilter(condition *FilterCondition, innerRec *Record, outerRec *Record) bool {
	innerField := innerRec.GetField(condition.InnerColumnName)
	outerField := outerRec.GetField(condition.OuterColumnName)
	return *innerField == *outerField
}

func getFilterCondition(query *Select, tableSchema *Table, scope *Scope) *FilterCondition {
	var filterCondition *FilterCondition
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
	return filterCondition
}
