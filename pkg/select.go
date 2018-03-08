package treesql

import (
	"context"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	clog "github.com/vilterp/treesql/pkg/log"
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

// TODO: maybe these should be on Channel, not Connection
func (conn *Connection) ExecuteTopLevelQuery(query *Select, channel *Channel) error {
	result, _, selectErr := conn.executeQuery(query, channel)
	if selectErr != nil {
		return errors.Wrap(selectErr, "query error")
	}
	channel.WriteInitialResult(&InitialResult{
		Data:   result,
		Schema: schemaOfQuery(query),
	})
	return nil
}

func (conn *Connection) ExecuteQueryForTableListener(query *Select, statementID int, channel *Channel) (SelectResult, error) {
	result, _, selectErr := conn.executeQuery(query, channel)
	//clog.Println(
	//	channel, "executed table listener query for statement", statementID, "in", duration,
	//)
	return result, selectErr
}

func schemaOfQuery(query *Select) map[string]interface{} {
	result := map[string]interface{}{}
	result["table"] = query.Table
	selectionSchemas := map[string]interface{}{}
	for _, selection := range query.Selections {
		if selection.SubSelect != nil {
			selectionSchemas[selection.Name] = schemaOfQuery(selection.SubSelect)
		}
	}
	result["selections"] = selectionSchemas
	return result
}

// can be from a live query or a top-level query
func (conn *Connection) executeQuery(
	query *Select,
	channel *Channel,
) (SelectResult, *time.Duration, error) {
	startTime := time.Now()
	tx, _ := conn.Database.BoltDB.Begin(false)
	ctx := context.WithValue(conn.Context, clog.ChannelIDKey, channel.ID)

	execution := &SelectExecution{
		ID:          ChannelID(channel.ID),
		Channel:     channel,
		Query:       query,
		Transaction: tx,
		Context:     ctx,
	}

	result, selectErr := execution.executeSelect(query, nil)
	if selectErr != nil {
		return nil, nil, selectErr
	}
	commitErr := tx.Rollback()
	if commitErr != nil {
		return nil, nil, commitErr
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	conn.Database.Metrics.selectLatency.Observe(float64(duration))
	//clog.Println(execution, "executed select in:", duration, "live:", query.Live)
	// TODO: structured logging XD

	return result, &duration, nil
}

// maybe this should be called transaction? idk
type SelectExecution struct {
	ID          ChannelID
	Channel     *Channel
	Query       *Select
	Transaction *bolt.Tx
	Context     context.Context
}

func (ex *SelectExecution) Ctx() context.Context {
	return ex.Context
}

type Scope struct {
	table         *TableDescriptor
	document      *Record
	pathSoFar     *QueryPath
	selectionName string
}

type FilterCondition struct {
	InnerColumnName string
	OuterColumnName string
}

// TODO: wrap & annotate with one/many
type SelectResult []map[string]interface{}

func (ex *SelectExecution) executeSelect(query *Select, scope *Scope) (SelectResult, error) {
	database := ex.Channel.Connection.Database
	table := database.Schema.Tables[query.Table]
	// if we're an inner loop, figure out a condition for our loop
	var filterCondition *FilterCondition
	if scope != nil {
		filterCondition = getFilterCondition(query, table, scope)
	}
	if ex.Query.Live {
		// add table subscription
		innerTable := database.Schema.Tables[query.Table]
		channel := database.Schema.Tables[innerTable.Name].LiveQueryInfo.TableSubscriptionEvents
		var colNameForSub *string
		var valueForSub *Value
		if filterCondition != nil {
			colNameForSub = &filterCondition.InnerColumnName
			valueForSub = scope.document.GetField(filterCondition.OuterColumnName)
		}
		if query.Where != nil {
			// TODO: unify these conditions and support ANDs in filtered table listeners
			// so don't need to worry about this
			if colNameForSub != nil {
				clog.Println(ex, "warn:", "overriding filter cond with where cond for subscription")
			}
			colNameForSub = &query.Where.ColumnName
			valueForSub = &Value{
				Type:      TypeString,
				StringVal: query.Where.Value,
			}
		}
		var queryPath *QueryPath
		if scope != nil {
			queryPath = scope.pathSoFar
		}
		channel <- &TableSubscriptionEvent{
			ColumnName:     colNameForSub,
			Value:          valueForSub,
			SubQuery:       query,
			QueryExecution: ex,
			QueryPath:      queryPath,
		}
	}
	if query.Where != nil {
		if query.Where.ColumnName == table.PrimaryKey {
			return ex.lookupRecord(query, query.Where.Value, scope, table)
		} else {
			return ex.scanTable(query, filterCondition, scope, table)
		}
	}
	if filterCondition != nil {
		if filterCondition.InnerColumnName == table.PrimaryKey {
			pkVal := scope.document.GetField(filterCondition.OuterColumnName).StringVal
			return ex.lookupRecord(query, pkVal, scope, table)
		} else {
			return ex.scanTable(query, filterCondition, scope, table)
		}
	}

	return ex.scanTable(query, filterCondition, scope, table)
}

func (ex *SelectExecution) lookupRecord(
	query *Select,
	pk string,
	scope *Scope,
	table *TableDescriptor,
) (SelectResult, error) {
	start := time.Now()

	// TODO: DRY
	// get schema fields into a map (maybe it should be this in the schema? idk)
	columnsMap := map[string]*ColumnDescriptor{}
	for _, column := range table.Columns {
		columnsMap[column.Name] = column
	}

	iterator, _ := ex.getTableIterator(table.Name)
	record, err := iterator.Get(pk)
	if err != nil {
		return nil, err
	}

	// This query is in the result set; subscribe to it.
	if ex.Query.Live {
		ex.subscribeToRecord(scope, record, table)
	}

	// Extract needed columns.
	recordResults, subSelectErr := getRecordResults(query, scope, table, record, ex, columnsMap)
	if subSelectErr != nil {
		return nil, subSelectErr
	}

	// Record duration.
	end := time.Now()
	duration := end.Sub(start)
	ex.Channel.Connection.Database.Metrics.lookupLatency.Observe(float64(duration.Nanoseconds()))

	return []map[string]interface{}{
		recordResults,
	}, nil
}

func (ex *SelectExecution) scanTable(
	query *Select,
	filterCondition *FilterCondition,
	scope *Scope,
	table *TableDescriptor,
) (SelectResult, error) {
	start := time.Now()
	result := make([]map[string]interface{}, 0)

	// get schema fields into a map (maybe it should be this in the schema? idk)
	columnsMap := map[string]*ColumnDescriptor{}
	for _, column := range table.Columns {
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
			return nil, fmt.Errorf("one row requested, but found > 1")
		}
		// this record is in the result set... let's subscribe to it
		if ex.Query.Live {
			ex.subscribeToRecord(scope, record, table)
		}
		// get all fields for selection
		recordResults, subSelectErr := getRecordResults(query, scope, table, record, ex, columnsMap)
		if subSelectErr != nil {
			return nil, subSelectErr
		}
		rowsRead++
		result = append(result, recordResults)
	}
	iterator.Close()
	if query.One && rowsRead == 0 {
		return nil, errors.New("error: requested one row, but none found")
		// TODO: this could be in the middle of a result set, lol
	}
	// Record duration.
	end := time.Now()
	duration := end.Sub(start)
	ex.Channel.Connection.Database.Metrics.scanLatency.Observe(float64(duration.Nanoseconds()))
	return result, nil
}

func getRecordResults(
	query *Select,
	scope *Scope,
	tableSchema *TableDescriptor,
	record *Record,
	ex *SelectExecution,
	columnsMap map[string]*ColumnDescriptor,
) (map[string]interface{}, error) {

	recordResults := map[string]interface{}{}
	// extract & write fields
	for _, selection := range query.Selections {
		if selection.SubSelect != nil {
			// execute subquery
			var queryPathSoFar *QueryPath
			if scope != nil {
				queryPathSoFar = scope.pathSoFar
			}
			// TODO: refactor: we've already made this in `executeSelect` above
			// maybe fold scope chain & query path together for fewer parameters
			queryPathWithPkVal := &QueryPath{
				ID:              &record.GetField(tableSchema.PrimaryKey).StringVal,
				PreviousSegment: queryPathSoFar,
			}
			queryPathWithSelection := &QueryPath{
				Selection:       &selection.Name,
				PreviousSegment: queryPathWithPkVal,
			}
			// execute subquery
			nextScope := &Scope{
				table:         tableSchema,
				document:      record,
				selectionName: selection.Name,
				pathSoFar:     queryPathWithSelection,
			}
			subselectResult, subselectErr := ex.executeSelect(selection.SubSelect, nextScope)
			if subselectErr != nil {
				return nil, subselectErr
			}
			recordResults[selection.Name] = subselectResult
		} else {
			// save field value
			//columnSpec := columnsMap[selection.Name]
			//switch columnSpec.Type {
			//case TypeInt:
			//	val := record.GetField(columnSpec.Name).StringVal
			//	recordResults[columnSpec.Name] = val
			//
			//case TypeString:
			//	val := record.GetField(columnSpec.Name).StringVal
			//	recordResults[columnSpec.Name] = val
			//}
		}
	}
	return recordResults, nil
}

func recordMatchesFilter(condition *FilterCondition, innerRec *Record, outerRec *Record) bool {
	innerField := innerRec.GetField(condition.InnerColumnName)
	outerField := outerRec.GetField(condition.OuterColumnName)
	return innerField.StringVal == outerField.StringVal // TODO: more than strings someday
}

func getFilterCondition(query *Select, tableSchema *TableDescriptor, scope *Scope) *FilterCondition {
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

func (ex *SelectExecution) subscribeToRecord(scope *Scope, record *Record, table *TableDescriptor) {
	var previousQueryPath *QueryPath
	if scope != nil {
		previousQueryPath = scope.pathSoFar
	}
	queryPathWithPkVal := &QueryPath{
		ID:              &record.GetField(table.PrimaryKey).StringVal,
		PreviousSegment: previousQueryPath,
	}
	tableEventsChannel := table.LiveQueryInfo.RecordSubscriptionEvents
	tableEventsChannel <- &RecordSubscriptionEvent{
		Value:          record.GetField(table.PrimaryKey),
		QueryExecution: ex,
		QueryPath:      queryPathWithPkVal,
	}
}
