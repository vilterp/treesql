package treesql

import (
	"log"

	"fmt"

	"github.com/vilterp/treesql/pkg/lang"
)

type listenerList struct {
	Table        *tableDescriptor
	Listeners    map[connectionID]map[channelID][]*Listener
	numListeners int
}

type Listener struct {
	QueryExecution *selectExecution
	// vv nil for record listeners
	Query     *Select
	QueryPath *queryPath
}

func (table *tableDescriptor) newListenerList() *listenerList {
	return &listenerList{
		Table:     table,
		Listeners: map[connectionID]map[channelID][]*Listener{},
	}
}

func (list *listenerList) addListener(listener *Listener) {
	stmtID := listener.QueryExecution.ID
	connID := connectionID(listener.QueryExecution.Channel.connection.id)
	listenersForConn := list.Listeners[connID]
	if listenersForConn == nil {
		listenersForConn = map[channelID][]*Listener{}
		list.Listeners[connID] = listenersForConn
	}
	listenersForStatement := listenersForConn[stmtID]
	if listenersForStatement == nil {
		listenersForStatement = make([]*Listener, 0)
	}
	listenersForStatement = append(listenersForStatement, listener)
	listenersForConn[stmtID] = listenersForStatement
	list.numListeners++
}

func (list *listenerList) removeListenersForConn(id connectionID) {
	count := 0
	for _, listenersForConn := range list.Listeners {
		for _, listenersForChan := range listenersForConn {
			count += len(listenersForChan)
		}
	}
	delete(list.Listeners, id)
	list.numListeners -= count
}

func (list *listenerList) getNumListeners() int {
	return list.numListeners
}

func (list *listenerList) addQueryListener(ex *selectExecution, query *Select, queryPath *queryPath) {
	list.addListener(&Listener{
		QueryExecution: ex,
		Query:          query,
		QueryPath:      queryPath,
	})
}

func (list *listenerList) addRecordListener(ex *selectExecution, queryPath *queryPath) {
	list.addListener(&Listener{
		QueryExecution: ex,
		QueryPath:      queryPath,
	})
}

func (list *listenerList) sendEvent(event *tableEvent) {
	for _, listenersForConn := range list.Listeners {
		for _, listenersForChannel := range listenersForConn {
			for _, listener := range listenersForChannel {
				if listener.Query != nil {
					// whole table or filtered table update
					conn := listener.QueryExecution.Channel.connection
					// want to just be like "clone this, with this different..."
					// like object spread operator in JS (also Elixir, Elm)
					// TODO: change this to be an FP expression

					whereVal := event.NewRecord.GetValue(list.Table.primaryKey)
					whereValString, err := valueMustBeString(whereVal)
					if err != nil {
						log.Println("can't do live query on non-string value:", whereVal.Format())
					}

					newQuery := &Select{
						Live:       true,
						Many:       listener.Query.Many,
						One:        listener.Query.One, // ugh
						Selections: listener.Query.Selections,
						Table:      listener.Query.Table,
						Where: &Where{
							ColumnName: list.Table.primaryKey,
							Value:      whereValString,
						}, // TODO: doesn't work if there was already a query... need AND support
					}
					go func() {
						result, selectErr := conn.executeQueryForTableListener(
							newQuery, int(listener.QueryExecution.ID), listener.QueryExecution.Channel,
						)
						if selectErr != nil {
							log.Println("failed to execute query for table listener statement id", listener.QueryExecution.ID)
						}
						listener.QueryExecution.Channel.writeTableUpdate(&TableUpdate{
							QueryPath: listener.QueryPath.flatten(),
							Selection: result,
						})
					}()
				} else {
					// record update
					listener.QueryExecution.Channel.writeRecordUpdate(event, listener.QueryPath)
				}
			}
		}
	}
}

func valueMustBeString(val lang.Value) (string, error) {
	vString, ok := val.(*lang.VString)
	if !ok {
		return "", fmt.Errorf("value in listener where clause must be string")
	}
	return string(*vString), nil
}
