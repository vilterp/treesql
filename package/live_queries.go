package treesql

import (
	"fmt"
)

func (db *Database) MakeTableListeners() {
	for _, table := range db.Schema.Tables {
		fmt.Println("making table listener for", table.Name)
		db.AddTableListener(table)
	}
}

type TableListener struct {
	Table                *Table
	TableEvents          chan *TableEvent
	SubscriberEvents     chan *SubscriberEvent
	ColumnValueListeners map[string](map[string]*ColumnValueListener) // column name => value => listener
	WholeTableListeners  []*WholeTableListener
}

type TableEvent struct {
	TableName string
	OldRecord *Record
	NewRecord *Record
}

type SubscriberEvent struct {
	ColumnName     string
	Value          *Value
	QueryExecution *QueryExecution
}

type WholeTableListener struct {
	Table       *Table
	Query       *Select
	LiveQueries []*QueryExecution
}

func (db *Database) AddTableListener(table *Table) {
	listener := &TableListener{
		Table:                table,
		TableEvents:          make(chan *TableEvent),
		SubscriberEvents:     make(chan *SubscriberEvent),
		ColumnValueListeners: map[string](map[string]*ColumnValueListener){},
	}
	// yet another thing to migrate when schema of this table changes
	for _, column := range table.Columns {
		listener.ColumnValueListeners[column.Name] = map[string](*ColumnValueListener){}
	}
	db.TableListeners[table.Name] = listener
	go tableListenerLoop(listener)
}

func tableListenerLoop(listener *TableListener) {
	for {
		select {
		case subEvent := <-listener.SubscriberEvents:
			fmt.Println("sub event for", listener.Table.Name, ":", subEvent)
			columnListeners := listener.ColumnValueListeners[subEvent.ColumnName]
			columnValueListener, ok := columnListeners[subEvent.Value.StringVal]
			if !ok {
				columnValueListener = newColumnValueListener(listener.Table.Name, subEvent)
				columnListeners[subEvent.Value.StringVal] = columnValueListener
			}
			columnValueListener.LiveQueries = append(columnValueListener.LiveQueries, subEvent.QueryExecution)
		case tableEvent := <-listener.TableEvents:
			fmt.Println("table event for", listener.Table.Name, ":", tableEvent)
			for columnName, columnValueListeners := range listener.ColumnValueListeners {
				fmt.Println("column value listeners for table", listener.Table.Name, "on column", columnName, "are", columnValueListeners)
				// TODO: integers, someday
				columnValueListener, ok := columnValueListeners[tableEvent.NewRecord.GetField(columnName).StringVal]
				if ok {
					columnValueListener.TableEvents <- tableEvent
				}
			}
		}
	}
}

type ColumnValueListener struct {
	TableEvents chan *TableEvent
	TableName   string
	ColumnName  string
	EqualsValue *Value
	LiveQueries []*QueryExecution
}

func newColumnValueListener(tableName string, subEvt *SubscriberEvent) *ColumnValueListener {
	listener := &ColumnValueListener{
		TableEvents: make(chan *TableEvent),
		TableName:   tableName,
		ColumnName:  subEvt.ColumnName,
		EqualsValue: subEvt.Value,
		LiveQueries: make([]*QueryExecution, 0),
	}
	fmt.Println("new column value listener", listener)
	go columnValueListenerLoop(listener)
	return listener
}

func columnValueListenerLoop(listener *ColumnValueListener) {
	for {
		tableEvent := <-listener.TableEvents
		fmt.Println("listener", listener, "event", tableEvent)
		for _, liveQuery := range listener.LiveQueries {
			liveQuery.Channel.WriteUpdateMessage(tableEvent)
		}
	}
}
