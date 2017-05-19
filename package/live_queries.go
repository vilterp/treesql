package treesql

import (
	"log"
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

type TableEvent struct {
	TableName string
	OldRecord *Record
	NewRecord *Record
}

type TableSubscriptionEvent struct {
	QueryExecution *QueryExecution
	QueryPath      *QueryPath
	SubQuery       *Select // where we are in the query
	// vv this and value null => subscribe to whole table w/ no filter
	ColumnName *string
	Value      *Value
}

type RecordSubscriptionEvent struct {
	QueryExecution *QueryExecution
	Value          *Value
	QueryPath      *QueryPath
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
				// whole table listener
				liveInfo.WholeTableListeners.AddQueryListener(
					tableSubEvent.QueryExecution, tableSubEvent.SubQuery, tableSubEvent.QueryPath,
				)
			} else {
				// filtered listener
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
				listenersForValue.AddQueryListener(
					tableSubEvent.QueryExecution, tableSubEvent.SubQuery, tableSubEvent.QueryPath,
				)
			}

		case recordSubEvent := <-liveInfo.RecordSubscriptionEvents:
			listenersForValue := liveInfo.RecordListeners[recordSubEvent.Value.StringVal]
			if listenersForValue == nil {
				listenersForValue = table.NewListenerList()
				liveInfo.RecordListeners[recordSubEvent.Value.StringVal] = listenersForValue
			}
			listenersForValue.AddRecordListener(recordSubEvent.QueryExecution, recordSubEvent.QueryPath)

		case tableEvent := <-liveInfo.TableEvents:
			if tableEvent.NewRecord != nil && tableEvent.OldRecord == nil {
				log.Println("pushing insert event to table listeners")
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
			} else if tableEvent.OldRecord != nil && tableEvent.NewRecord != nil {
				log.Println("pushing update event to table listeners")
				// record listeners
				primaryKeyValue := tableEvent.NewRecord.GetField(table.PrimaryKey).StringVal
				recordListeners := liveInfo.RecordListeners[primaryKeyValue]
				if recordListeners != nil {
					recordListeners.SendEvent(tableEvent)
				}
			} else if tableEvent.OldRecord != nil && tableEvent.NewRecord == nil {
				log.Println("TODO: handle delete events")
			}
		}
	}
}
