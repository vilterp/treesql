package treesql

import (
	"context"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/vilterp/treesql/pkg/lang"
	clog "github.com/vilterp/treesql/pkg/log"
)

// TODO: maybe these should be on Channel, not Connection
func (conn *Connection) ExecuteTopLevelQuery(query *Select, channel *Channel) error {
	result, caller, _, selectErr := conn.executeQuery(query, channel)
	if selectErr != nil {
		return errors.Wrap(selectErr, "query error")
	}
	channel.WriteInitialResult(&InitialResult{
		Value:  result,
		Caller: caller,
		Type:   result.GetType(),
	})
	return nil
}

func (conn *Connection) ExecuteQueryForTableListener(
	query *Select, statementID int, channel *Channel,
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
func (conn *Connection) executeQuery(
	query *Select,
	channel *Channel,
) (lang.Value, lang.Caller, *time.Duration, error) {
	startTime := time.Now()
	tx, _ := conn.Database.BoltDB.Begin(false)
	// ctx := context.WithValue(conn.Context, clog.ChannelIDKey, channel.ID)

	// Plan the query.
	expr, err := conn.Database.Schema.planSelect(query)
	if err != nil {
		return nil, nil, nil, err
	}

	clog.Println(conn, "QUERY PLAN:", expr.Format())

	// Make transaction and scope.
	txn := &Txn{
		db:      conn.Database,
		boltTxn: tx,
	}
	scope, _ := conn.Database.Schema.toScope(txn)

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
