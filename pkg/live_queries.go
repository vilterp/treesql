package treesql

import (
	"sync"
	"time"

	clog "github.com/vilterp/treesql/pkg/log"
)

// liveQueryInfo lives in a table...
type liveQueryInfo struct {
	// input channels
	TableEvents              chan *tableEvent
	RecordSubscriptionEvents chan *recordSubscriptionEvent
	TableSubscriptionEvents  chan *tableSubscriptionEvent
	// subscribers

	mu struct {
		sync.RWMutex

		TableListeners      map[columnName]map[string]*listenerList // column name => value => listener
		WholeTableListeners *listenerList
		RecordListeners     map[string]*listenerList
	}
}

func (table *tableDescriptor) newLiveQueryInfo() *liveQueryInfo {
	lqi := &liveQueryInfo{
		TableEvents:              make(chan *tableEvent),
		TableSubscriptionEvents:  make(chan *tableSubscriptionEvent),
		RecordSubscriptionEvents: make(chan *recordSubscriptionEvent),
	}
	lqi.mu.TableListeners = make(map[columnName]map[string]*listenerList)
	lqi.mu.WholeTableListeners = table.newListenerList()
	lqi.mu.RecordListeners = make(map[string]*listenerList)
	return lqi
}

type tableEvent struct {
	TableName string
	OldRecord *record
	NewRecord *record

	channel *channel
}

type tableSubscriptionEvent struct {
	QueryExecution *selectExecution
	QueryPath      *queryPath
	SubQuery       *Select // where we are in the query
	// vv this and value null => subscribe to whole table w/ no filter
	ColumnName *string
	Value      *value

	channel *channel
}

type recordSubscriptionEvent struct {
	QueryExecution *selectExecution
	Value          *value
	QueryPath      *queryPath

	channel *channel
}

func (table *tableDescriptor) removeListenersForConn(id connectionID) {
	liveInfo := table.liveQueryInfo
	liveInfo.mu.Lock()
	defer liveInfo.mu.Unlock()

	liveInfo.mu.WholeTableListeners.removeListenersForConn(id)
	for _, listenersForCol := range liveInfo.mu.TableListeners {
		for _, listenersForVal := range listenersForCol {
			listenersForVal.removeListenersForConn(id)
		}
	}
	// TODO: this is O(num vals being listened on)
	// Index it by conn.
	for _, list := range liveInfo.mu.RecordListeners {
		list.removeListenersForConn(id)
	}
}

func (table *tableDescriptor) handleEvents() {
	// PERF: I guess all writes and (live) reads are serialized through here
	// that seems bad for perf
	// you'd have to shard the channels themselves somehow... e.g. for p.k. listeners,
	// each record has its own goroutine...
	// TODO (safety): all these long-lived values are making me nervous
	// Bolt may recycle the underlying memory. fuck
	liveInfo := table.liveQueryInfo
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

func (table *tableDescriptor) handleTableSub(evt *tableSubscriptionEvent) {
	liveInfo := table.liveQueryInfo
	liveInfo.mu.Lock()
	defer liveInfo.mu.Unlock()

	if evt.ColumnName == nil {
		// whole table listener
		liveInfo.mu.WholeTableListeners.addQueryListener(
			evt.QueryExecution, evt.SubQuery, evt.QueryPath,
		)
	} else {
		// filtered listener
		columnName := columnName(*evt.ColumnName)
		// initialize listeners for this column (could be done at table create/load)
		// but that would leave us open when new columns are added
		listenersForColumn := liveInfo.mu.TableListeners[columnName]
		if listenersForColumn == nil {
			listenersForColumn = map[string]*listenerList{}
			liveInfo.mu.TableListeners[columnName] = listenersForColumn
		}
		// initialize listeners for this value in this column
		listenersForValue := listenersForColumn[evt.Value.stringVal]
		if listenersForValue == nil {
			listenersForValue = table.newListenerList()
			listenersForColumn[evt.Value.stringVal] = listenersForValue
		}
		listenersForValue.addQueryListener(
			evt.QueryExecution, evt.SubQuery, evt.QueryPath,
		)
	}
}

func (table *tableDescriptor) handleRecordSub(evt *recordSubscriptionEvent) {
	liveInfo := table.liveQueryInfo
	liveInfo.mu.Lock()
	defer liveInfo.mu.Unlock()

	listenersForValue := liveInfo.mu.RecordListeners[evt.Value.stringVal]
	if listenersForValue == nil {
		listenersForValue = table.newListenerList()
		liveInfo.mu.RecordListeners[evt.Value.stringVal] = listenersForValue
	}
	listenersForValue.addRecordListener(evt.QueryExecution, evt.QueryPath)
}

func (table *tableDescriptor) handleTableEvent(evt *tableEvent) {
	startTime := time.Now()
	liveInfo := table.liveQueryInfo
	liveInfo.mu.Lock()
	defer liveInfo.mu.Unlock()

	if evt.NewRecord != nil && evt.OldRecord == nil {
		// clog.Println(evt.channel, "pushing insert event to table listeners")
		// whole table listeners
		liveInfo.mu.WholeTableListeners.sendEvent(evt)
		// filtered table listeners
		for columnName, listenersForColumn := range liveInfo.mu.TableListeners {
			valueForColumn := evt.NewRecord.GetField(string(columnName)).stringVal
			listenersForValue := listenersForColumn[valueForColumn]
			if listenersForValue != nil {
				listenersForValue.sendEvent(evt)
			}
		}
	} else if evt.OldRecord != nil && evt.NewRecord != nil {
		clog.Println(evt.channel, "pushing update event to table listeners")
		// record listeners
		primaryKeyValue := evt.NewRecord.GetField(table.primaryKey).stringVal
		recordListeners := liveInfo.mu.RecordListeners[primaryKeyValue]
		if recordListeners != nil {
			recordListeners.sendEvent(evt)
		}
	} else if evt.OldRecord != nil && evt.NewRecord == nil {
		clog.Println(evt.channel, "TODO: handle delete events")
	}
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	// TODO: get metrics more directly (i.e. not through the event)
	metrics := evt.channel.connection.database.metrics
	metrics.liveQueryPushLatency.Observe(float64(duration.Nanoseconds()))
}
