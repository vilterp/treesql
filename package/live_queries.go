package treesql

import "fmt"

// LiveQueryInfo lives in a table...
type LiveQueryInfo struct {
	// input channels
	TableEvents              chan *TableEvent
	RecordSubscriptionEvents chan *RecordSubscriptionEvent
	TableSubscriptionEvents  chan *TableSubscriptionEvent
	// subscribers
	TableListeners      map[ColumnName](map[string]*ListenerList) // column name => value => listener
	WholeTableListeners *ListenerList
	RecordListeners     map[string]*ListenerList
}

func (table *Table) EmptyLiveQueryInfo() *LiveQueryInfo {
	return &LiveQueryInfo{
		TableEvents:              make(chan *TableEvent),
		TableSubscriptionEvents:  make(chan *TableSubscriptionEvent),
		RecordSubscriptionEvents: make(chan *RecordSubscriptionEvent),
		TableListeners:           map[ColumnName](map[string]*ListenerList){},
		WholeTableListeners:      table.NewListenerList(),
		RecordListeners:          map[string]*ListenerList{},
	}
}

// type ListenerList map[ConnectionID]([]*QueryExecution)
type ListenerList struct {
	Table     *Table
	Listeners []*Listener
}

type Listener struct {
	QueryExecution *QueryExecution
	Query          *Select // nil for record listeners
}

func (table *Table) NewListenerList() *ListenerList {
	return &ListenerList{
		Table:     table,
		Listeners: make([]*Listener, 0),
	}
}

func (list *ListenerList) AddQueryListener(ex *QueryExecution, query *Select) {
	list.Listeners = append(list.Listeners, &Listener{
		QueryExecution: ex,
		Query:          query,
	})
}

func (list *ListenerList) AddRecordListener(ex *QueryExecution) {
	list.Listeners = append(list.Listeners, &Listener{
		QueryExecution: ex,
	})
}

func (list *ListenerList) SendEvent(event *TableEvent) {
	for _, listener := range list.Listeners {
		if listener.Query != nil {
			conn := listener.QueryExecution.Channel.Connection
			// TODO: make up a query
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
			fmt.Println("\texecuting new query")
			go conn.ExecuteQuery(
				newQuery, int(listener.QueryExecution.ID), listener.QueryExecution.Channel,
			)
		} else {
			listener.QueryExecution.Channel.WriteUpdateMessage(event)
		}
	}
}

type TableEvent struct {
	TableName string
	OldRecord *Record
	NewRecord *Record
}

type TableSubscriptionEvent struct {
	QueryExecution *QueryExecution
	// QueryPath      *QueryPath
	SubQuery *Select // where we are in the query
	// vv this and value null => subscribe to whole table w/ no filter
	ColumnName *string
	Value      *Value
}

type RecordSubscriptionEvent struct {
	Value          *Value
	QueryExecution *QueryExecution
}

func (table *Table) HandleEvents() {
	// PERF: I guess all (live) reads and writes are serialized through here
	// that seems bad for perf
	// you'd have to shard the channels themselves somehow... e.g. for p.k. listeners,
	// each record has its own goroutine...
	// TODO (safety): all these long-lived values are making me nervous
	// Bolt may recycle the underlying memory. fuck
	liveInfo := table.LiveQueryInfo
	for {
		select {
		case tableSubEvent := <-liveInfo.TableSubscriptionEvents:
			if tableSubEvent.ColumnName == nil {
				liveInfo.WholeTableListeners.AddQueryListener(
					tableSubEvent.QueryExecution, tableSubEvent.SubQuery,
				)
			} else {
				columnName := ColumnName(*tableSubEvent.ColumnName)
				// initialize listeners for this column (could be done at table create/load)
				// but that would leave us open when new columns are added
				listenersForColumn := liveInfo.TableListeners[columnName]
				if listenersForColumn == nil {
					listenersForColumn = map[string]*ListenerList{}
					liveInfo.TableListeners[columnName] = listenersForColumn
				}
				// initialize listeners for this value in this column
				listenersForValue := listenersForColumn[tableSubEvent.Value.StringVal]
				if listenersForValue == nil {
					listenersForValue = table.NewListenerList()
					listenersForColumn[tableSubEvent.Value.StringVal] = listenersForValue
				}
				listenersForValue.AddQueryListener(tableSubEvent.QueryExecution, tableSubEvent.SubQuery)
			}

		case recordSubEvent := <-liveInfo.RecordSubscriptionEvents:
			listenersForValue := liveInfo.RecordListeners[recordSubEvent.Value.StringVal]
			if listenersForValue == nil {
				listenersForValue = table.NewListenerList()
				liveInfo.RecordListeners[recordSubEvent.Value.StringVal] = listenersForValue
			}
			listenersForValue.AddRecordListener(recordSubEvent.QueryExecution)

		case tableEvent := <-liveInfo.TableEvents:
			fmt.Println("table event")
			// whole table listeners
			liveInfo.WholeTableListeners.SendEvent(tableEvent)
			// filtered table listeners
			for columnName, listenersForColumn := range liveInfo.TableListeners {
				valueForColumn := tableEvent.NewRecord.GetField(string(columnName)).StringVal
				listenersForValue := listenersForColumn[valueForColumn]
				if listenersForValue != nil {
					listenersForValue.SendEvent(tableEvent)
				}
			}
			// record listeners
			primaryKeyValue := tableEvent.NewRecord.GetField(table.PrimaryKey).StringVal
			recordListeners := liveInfo.RecordListeners[primaryKeyValue]
			if recordListeners != nil {
				recordListeners.SendEvent(tableEvent)
			}
			// TODO: handle deletes someday, heh
		}
	}
}
