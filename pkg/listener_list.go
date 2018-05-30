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
	channel *channel
	// vv nil for record listeners
	query     *Select
	queryPath *queryPath
}

func (table *tableDescriptor) newListenerList() *listenerList {
	return &listenerList{
		Table:     table,
		Listeners: map[connectionID]map[channelID][]*Listener{},
	}
}

func (list *listenerList) addListener(listener *Listener) {
	stmtID := channelID(listener.channel.id)
	connID := connectionID(listener.channel.connection.id)
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

func (list *listenerList) addQueryListener(c *channel, query *Select, queryPath *queryPath) {
	list.addListener(&Listener{
		channel:   c,
		query:     query,
		queryPath: queryPath,
	})
}

func (list *listenerList) addRecordListener(c *channel, queryPath *queryPath) {
	list.addListener(&Listener{
		channel:   c,
		queryPath: queryPath,
	})
}

func (list *listenerList) sendEvent(event *tableEvent) {
	for _, listenersForConn := range list.Listeners {
		for _, listenersForChannel := range listenersForConn {
			for _, listener := range listenersForChannel {
				if listener.query != nil {
					// whole table or filtered table update
					conn := listener.channel.connection
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
						Many:       listener.query.Many,
						One:        listener.query.One, // ugh
						Selections: listener.query.Selections,
						Table:      listener.query.Table,
						Where: &Where{
							ColumnName: list.Table.primaryKey,
							Value:      whereValString,
						}, // TODO: doesn't work if there was already a query... need AND support
					}
					go func() {
						result, selectErr := conn.executeQueryForTableListener(
							newQuery, int(listener.channel.id), listener.channel,
						)
						if selectErr != nil {
							log.Println("failed to execute query for table listener statement id", listener.channel.id)
						}
						listener.channel.writeTableUpdate(&TableUpdate{
							QueryPath: listener.queryPath.flatten(),
							Selection: result,
						})
					}()
				} else {
					// record update
					listener.channel.writeRecordUpdate(event, listener.queryPath)
				}
			}
		}
	}
}

func valueMustBeString(val lang.Value) (string, error) {
	vString, ok := val.(*lang.VString)
	if !ok {
		return "", fmt.Errorf("value in Listener where clause must be string")
	}
	return string(*vString), nil
}
