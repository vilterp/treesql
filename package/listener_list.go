package treesql

import (
	"log"
)

// type ListenerList map[ConnectionID]([]*QueryExecution)
type ListenerList struct {
	Table     *Table
	Listeners map[ConnectionID](map[StatementID]([]*Listener))
}

type Listener struct {
	QueryExecution *QueryExecution
	// vv nil for record listeners
	Query     *Select
	QueryPath *QueryPath
}

func (table *Table) NewListenerList() *ListenerList {
	return &ListenerList{
		Table:     table,
		Listeners: map[ConnectionID](map[StatementID]([]*Listener)){},
	}
}

func (list *ListenerList) addListener(listener *Listener) {
	stmtID := listener.QueryExecution.ID
	connID := ConnectionID(listener.QueryExecution.Channel.Connection.ID)
	listenersForConn := list.Listeners[connID]
	if listenersForConn == nil {
		listenersForConn = map[StatementID]([]*Listener){}
		list.Listeners[connID] = listenersForConn
	}
	listenersForStatement := listenersForConn[stmtID]
	if listenersForStatement == nil {
		listenersForStatement = make([]*Listener, 0)
	}
	listenersForStatement = append(listenersForStatement, listener)
	listenersForConn[stmtID] = listenersForStatement
}

func (list *ListenerList) AddQueryListener(ex *QueryExecution, query *Select, queryPath *QueryPath) {
	list.addListener(&Listener{
		QueryExecution: ex,
		Query:          query,
		QueryPath:      queryPath,
	})
}

func (list *ListenerList) AddRecordListener(ex *QueryExecution, queryPath *QueryPath) {
	list.addListener(&Listener{
		QueryExecution: ex,
		QueryPath:      queryPath,
	})
}

func (list *ListenerList) SendEvent(event *TableEvent) {
	for _, listenersForConn := range list.Listeners {
		for _, listenersForChannel := range listenersForConn {
			for _, listener := range listenersForChannel {
				if listener.Query != nil {
					// whole table or filtered table update
					conn := listener.QueryExecution.Channel.Connection
					// want to just be like "clone this, with this different..."
					// like object spread operator in JS (also Elixir, Elm)
					newQuery := &Select{
						Live:       true,
						Many:       listener.Query.Many,
						One:        listener.Query.One, // ugh
						Selections: listener.Query.Selections,
						Table:      listener.Query.Table,
						Where: &Where{
							ColumnName: list.Table.PrimaryKey,
							Value:      event.NewRecord.GetField(list.Table.PrimaryKey).StringVal,
						}, // TODO: doesn't work if there was already a query... need AND support
					}
					go func() {
						result, selectErr := conn.ExecuteQueryForTableListener(
							newQuery, int(listener.QueryExecution.ID), listener.QueryExecution.Channel,
						)
						if selectErr != nil {
							log.Println("failed to execute query for table listener statement id", listener.QueryExecution.ID)
						}
						listener.QueryExecution.Channel.WriteTableUpdate(&TableUpdate{
							QueryPath: listener.QueryPath,
							Selection: result,
						})
					}()
				} else {
					// record update
					listener.QueryExecution.Channel.WriteRecordUpdate(event, listener.QueryPath)
				}
			}
		}
	}
}
