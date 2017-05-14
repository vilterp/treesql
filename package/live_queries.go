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

type TableListener struct {
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

func (table *Table) HandleSubscriptionEvents() {
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

func (table *Table) HandleTableEvents() {
	for {
		tableEvent := <-table.TableEvents
		fmt.Println("table event for", table, ":", tableEvent)
	}
}
