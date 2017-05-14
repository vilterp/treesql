package treesql

import (
	"fmt"
)

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

func EmptyLiveQueryInfo() *LiveQueryInfo {
	return &LiveQueryInfo{
		TableEvents:              make(chan *TableEvent),
		TableSubscriptionEvents:  make(chan *TableSubscriptionEvent),
		RecordSubscriptionEvents: make(chan *RecordSubscriptionEvent),
		TableListeners:           map[ColumnName](map[string]*ListenerList){},
		WholeTableListeners:      NewListenerList(),
		RecordListeners:          map[string]*ListenerList{},
	}
}

// type ListenerList map[ConnectionID]([]*QueryExecution)
type ListenerList struct {
	Listeners []*Listener
}

type Listener struct {
	QueryExecution *QueryExecution
	Query          *Select // nil for record listeners
}

func NewListenerList() *ListenerList {
	return &ListenerList{
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
	fmt.Println("send event", event, list.Listeners)
	for _, listener := range list.Listeners {
		fmt.Println("\tlistener", listener)
		if listener.Query != nil {
			// conn := listener.QueryExecution.Channel.Connection
			fmt.Println("\t\texecuting sub query", listener.Query)
			listener.QueryExecution.Channel.WriteUpdateMessage(listener.Query)
			// conn.ExecuteQuery(listener.Query, int(listener.QueryExecution.ID), listener.QueryExecution.Channel)
		} else {
			fmt.Println("\t\trecord update")
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
			fmt.Println("table sub event for", table.Name, ":", tableSubEvent)
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
					listenersForValue = NewListenerList()
					listenersForColumn[tableSubEvent.Value.StringVal] = listenersForValue
				}
				listenersForValue.AddQueryListener(tableSubEvent.QueryExecution, tableSubEvent.SubQuery)
			}

		case recordSubEvent := <-liveInfo.RecordSubscriptionEvents:
			fmt.Println("\trecord sub event for", table.Name, ":", recordSubEvent)
			listenersForValue := liveInfo.RecordListeners[recordSubEvent.Value.StringVal]
			if listenersForValue == nil {
				listenersForValue = NewListenerList()
				liveInfo.RecordListeners[recordSubEvent.Value.StringVal] = listenersForValue
			}
			listenersForValue.AddRecordListener(recordSubEvent.QueryExecution)

		case tableEvent := <-liveInfo.TableEvents:
			fmt.Println("table event for", table.Name, ":", tableEvent)
			// whole table listeners
			fmt.Println("whole table event", tableEvent)
			liveInfo.WholeTableListeners.SendEvent(tableEvent)
			// filtered table listeners
			fmt.Println("tableListeners for", table.Name, ":", liveInfo.TableListeners)
			for columnName, listenersForColumn := range liveInfo.TableListeners {
				fmt.Println("\tLFC", columnName, listenersForColumn)
				valueForColumn := tableEvent.NewRecord.GetField(string(columnName)).StringVal
				listenersForValue := listenersForColumn[valueForColumn]
				fmt.Println("\tLFV", listenersForValue)
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
