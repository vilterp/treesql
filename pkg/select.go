package treesql

import (
	"context"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/vilterp/treesql/pkg/lang"
	clog "github.com/vilterp/treesql/pkg/log"
)

// TODO: maybe these should be on channel, not connection
func (conn *connection) executeTopLevelQuery(query *Select, channel *channel) error {
	result, caller, _, selectErr := conn.executeQuery(query, channel)
	if selectErr != nil {
		return errors.Wrap(selectErr, "query error")
	}
	channel.writeInitialResult(&InitialResult{
		Value:  result,
		Caller: caller,
		Type:   result.GetType(),
	})
	return nil
}

func (conn *connection) executeQueryForTableListener(
	query *Select, statementID int, channel *channel,
) (lang.Value, error) {
	result, _, _, selectErr := conn.executeQuery(query, channel)
	//clog.Println(
	//	channel, "executed table listener query for statement", statementID, "in", duration,
	//)
	return result, selectErr
}

// can be from a live query or a top-level query
// TODO: add live query stuff back in
// TODO: add timing back somewhere else
func (conn *connection) executeQuery(
	query *Select,
	channel *channel,
) (lang.Value, lang.Caller, *time.Duration, error) {
	startTime := time.Now()
	tx, _ := conn.database.boltDB.Begin(false)
	// ctx := context.WithValue(conn.context, clog.ChannelIDKey, channel.id)

	// Make transaction and scope.
	txn := &txn{
		db:      conn.database,
		boltTxn: tx,
	}
	scope, typeScope := conn.database.schema.toScope(txn)

	// Plan the query.
	expr, err := conn.database.schema.planSelect(query, typeScope)
	if err != nil {
		return nil, nil, nil, err
	}

	clog.Println(conn, "QUERY PLAN:", expr.Format())

	// Interpret the expr.
	interp := lang.NewInterpreter(scope, expr)
	val, err := interp.Interpret()
	if err != nil {
		return nil, nil, nil, err
	}

	// Measure execution time.
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	return val, interp, &duration, nil
}

// maybe this should be called transaction? idk
type selectExecution struct {
	ID          channelID
	Channel     *channel
	Query       *Select
	Transaction *bolt.Tx
	Context     context.Context
}

func (ex *selectExecution) Ctx() context.Context {
	return ex.Context
}

type scope struct {
	table         *tableDescriptor
	document      *record
	pathSoFar     *queryPath
	selectionName string
}

type filterCondition struct {
	InnerColumnName string
	OuterColumnName string
}

func (ex *selectExecution) subscribeToRecord(scope *scope, record *record, table *tableDescriptor) {
	var previousQueryPath *queryPath
	if scope != nil {
		previousQueryPath = scope.pathSoFar
	}
	queryPathWithPkVal := &queryPath{
		ID:              &record.GetField(table.primaryKey).stringVal,
		PreviousSegment: previousQueryPath,
	}
	tableEventsChannel := table.liveQueryInfo.RecordSubscriptionEvents
	tableEventsChannel <- &recordSubscriptionEvent{
		Value:          record.GetField(table.primaryKey),
		QueryExecution: ex,
		QueryPath:      queryPathWithPkVal,
	}
}
