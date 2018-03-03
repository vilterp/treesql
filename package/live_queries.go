package treesql

import (
	"time"

	clog "github.com/vilterp/treesql/package/log"
)

// LiveQueryInfo lives in a table...
type LiveQueryInfo struct {
	// input channels
	TableEvents              chan *TableEvent
	RecordSubscriptionEvents chan *RecordSubscriptionEvent
	TableSubscriptionEvents  chan *TableSubscriptionEvent
	// subscribers
	TableListeners      map[ColumnName]map[string]*ListenerList // column name => value => listener
	WholeTableListeners *ListenerList
	RecordListeners     map[string]*ListenerList
}

func (table *TableDescriptor) NewLiveQueryInfo() *LiveQueryInfo {
	return &LiveQueryInfo{
		TableEvents:              make(chan *TableEvent),
		TableSubscriptionEvents:  make(chan *TableSubscriptionEvent),
		RecordSubscriptionEvents: make(chan *RecordSubscriptionEvent),
		TableListeners:           make(map[ColumnName]map[string]*ListenerList),
		WholeTableListeners:      table.NewListenerList(),
		RecordListeners:          make(map[string]*ListenerList),
	}
}

type TableEvent struct {
	TableName string
	OldRecord *Record
	NewRecord *Record

	channel *Channel
}

type TableSubscriptionEvent struct {
	QueryExecution *SelectExecution
	QueryPath      *QueryPath
	SubQuery       *Select // where we are in the query
	// vv this and value null => subscribe to whole table w/ no filter
	ColumnName *string
	Value      *Value

	channel *Channel
}

type RecordSubscriptionEvent struct {
	QueryExecution *SelectExecution
	Value          *Value
	QueryPath      *QueryPath

	channel *Channel
}

func (table *TableDescriptor) removeListenersForConn(id ConnectionID) {
	liveInfo := table.LiveQueryInfo
	liveInfo.WholeTableListeners.removeListenersForConn(id)
	for _, listenersForCol := range liveInfo.TableListeners {
		for _, listenersForVal := range listenersForCol {
			listenersForVal.removeListenersForConn(id)
		}
	}
	// TODO: this is O(num vals being listened on)
	// Index it by conn.
	for _, list := range liveInfo.RecordListeners {
		list.removeListenersForConn(id)
	}
}

func (table *TableDescriptor) HandleEvents() {
	// PERF: I guess all writes and (live) reads are serialized through here
	// that seems bad for perf
	// you'd have to shard the channels themselves somehow... e.g. for p.k. listeners,
	// each record has its own goroutine...
	// TODO (safety): all these long-lived values are making me nervous
	// Bolt may recycle the underlying memory. fuck
	liveInfo := table.LiveQueryInfo
	for {
		select {
		case tableSubEvent := <-liveInfo.TableSubscriptionEvents:
			table.handleTableSub(tableSubEvent)

		case recordSubEvent := <-liveInfo.RecordSubscriptionEvents:
			table.handleRecordSub(recordSubEvent)

		case tableEvent := <-liveInfo.TableEvents:
			table.handleTableEvent(tableEvent)
		}
	}
}

func (table *TableDescriptor) handleTableSub(evt *TableSubscriptionEvent) {
	liveInfo := table.LiveQueryInfo
	if evt.ColumnName == nil {
		// whole table listener
		liveInfo.WholeTableListeners.AddQueryListener(
			evt.QueryExecution, evt.SubQuery, evt.QueryPath,
		)
	} else {
		// filtered listener
		columnName := ColumnName(*evt.ColumnName)
		// initialize listeners for this column (could be done at table create/load)
		// but that would leave us open when new columns are added
		listenersForColumn := liveInfo.TableListeners[columnName]
		if listenersForColumn == nil {
			listenersForColumn = map[string]*ListenerList{}
			liveInfo.TableListeners[columnName] = listenersForColumn
		}
		// initialize listeners for this value in this column
		listenersForValue := listenersForColumn[evt.Value.StringVal]
		if listenersForValue == nil {
			listenersForValue = table.NewListenerList()
			listenersForColumn[evt.Value.StringVal] = listenersForValue
		}
		listenersForValue.AddQueryListener(
			evt.QueryExecution, evt.SubQuery, evt.QueryPath,
		)
	}
}

func (table *TableDescriptor) handleRecordSub(evt *RecordSubscriptionEvent) {
	liveInfo := table.LiveQueryInfo
	listenersForValue := liveInfo.RecordListeners[evt.Value.StringVal]
	if listenersForValue == nil {
		listenersForValue = table.NewListenerList()
		liveInfo.RecordListeners[evt.Value.StringVal] = listenersForValue
	}
	listenersForValue.AddRecordListener(evt.QueryExecution, evt.QueryPath)
}

func (table *TableDescriptor) handleTableEvent(evt *TableEvent) {
	startTime := time.Now()
	liveInfo := table.LiveQueryInfo
	if evt.NewRecord != nil && evt.OldRecord == nil {
		// clog.Println(evt.channel, "pushing insert event to table listeners")
		// whole table listeners
		liveInfo.WholeTableListeners.SendEvent(evt)
		// filtered table listeners
		for columnName, listenersForColumn := range liveInfo.TableListeners {
			valueForColumn := evt.NewRecord.GetField(string(columnName)).StringVal
			listenersForValue := listenersForColumn[valueForColumn]
			if listenersForValue != nil {
				listenersForValue.SendEvent(evt)
			}
		}
	} else if evt.OldRecord != nil && evt.NewRecord != nil {
		clog.Println(evt.channel, "pushing update event to table listeners")
		// record listeners
		primaryKeyValue := evt.NewRecord.GetField(table.PrimaryKey).StringVal
		recordListeners := liveInfo.RecordListeners[primaryKeyValue]
		if recordListeners != nil {
			recordListeners.SendEvent(evt)
		}
	} else if evt.OldRecord != nil && evt.NewRecord == nil {
		clog.Println(evt.channel, "TODO: handle delete events")
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	// TODO: get metrics more directly (i.e. not through the event)
	metrics := evt.channel.Connection.Database.Metrics
	metrics.liveQueryPushLatency.Observe(float64(duration.Nanoseconds()))
}
