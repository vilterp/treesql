package treesql

import (
	"fmt"
)

type TableEvent struct {
	TableName string
	OldRecord *Record
	NewRecord *Record
}

type TableSubscriptionEvent struct {
	ColumnName     string
	Value          *Value
	QueryExecution *QueryExecution
}

type RecordSubscriptionEvent struct {
	Value          *Value
	QueryExecution *QueryExecution
}

type ColumnValueListener struct {
	TableEvents chan *TableEvent
	TableName   string
	ColumnName  string
	EqualsValue *Value
	LiveQueries []*QueryExecution
}

type RecordListener struct {
	Table       *Table
	Value       *Value // value of primary key
	LiveQueries map[ConnectionID]*QueryExecution
}

func (table *Table) tableListenerLoop() {
	for {
		select {
		case tableSubEvent := <-table.TableSubscriptionEvents:
			fmt.Println("table sub event for", table.Name, ":", tableSubEvent)
		case recordSubEvent := <-table.RecordSubscriptionEvents:
			fmt.Println("record sub event for", table.Name, ":", recordSubEvent)
		case tableEvent := <-table.TableEvents:
			fmt.Println("table event for", table.Name, ":", tableEvent)
		}
	}
}

// func newColumnValueListener(tableName string, subEvt *SubscriberEvent) *ColumnValueListener {
// 	listener := &ColumnValueListener{
// 		TableEvents: make(chan *TableEvent),
// 		TableName:   tableName,
// 		ColumnName:  subEvt.ColumnName,
// 		EqualsValue: subEvt.Value,
// 		LiveQueries: make([]*QueryExecution, 0),
// 	}
// 	fmt.Println("new column value listener", listener)
// 	go columnValueListenerLoop(listener)
// 	return listener
// }

func columnValueListenerLoop(listener *ColumnValueListener) {
	for {
		tableEvent := <-listener.TableEvents
		fmt.Println("listener", listener, "event", tableEvent)
		for _, liveQuery := range listener.LiveQueries {
			liveQuery.Channel.WriteUpdateMessage(tableEvent)
		}
	}
}
