package treesql

import (
	"bufio"
)

func (db *Database) MakeTableListeners() {
	for _, table := range db.Schema.Tables {
		db.AddTableListener(table)
	}
}

type TableListener struct {
	Table            *Table
	TableEvents      chan *TableEvent
	SubscriberEvents chan *SubscriberEvent
	// PointListeners map[Value]*PointListener // this is just a field listener on the primary key...
	ColumnValueListeners map[string](map[string]*ColumnValueListener) // column name => value => listener
}

type TableEvent struct {
	OldRecord *Record
	NewRecord *Record
}

type SubscriberEvent struct {
	ColumnName     string
	Value          *Value
	QueryExecution *QueryExecution
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
			columnListeners := listener.ColumnValueListeners[subEvent.ColumnName]
			columnValueListener, ok := columnListeners[subEvent.Value.StringVal]
			if !ok {
				columnValueListener = newColumnValueListener(subEvent)
				columnListeners[subEvent.Value.StringVal] = columnValueListener
			}
			columnValueListener.LiveQueries = append(columnValueListener.LiveQueries, subEvent.QueryExecution)
		case tableEvent := <-listener.TableEvents:
			for columnName, columnValueListeners := range listener.ColumnValueListeners {
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
	ColumnName  string
	EqualsValue *Value
	LiveQueries []*QueryExecution
}

func newColumnValueListener(subEvt *SubscriberEvent) *ColumnValueListener {
	listener := &ColumnValueListener{
		TableEvents: make(chan *TableEvent),
		ColumnName:  subEvt.ColumnName,
		EqualsValue: subEvt.Value,
		LiveQueries: make([]*QueryExecution, 0),
	}
	go columnValueListenerLoop(listener)
	return listener
}

func columnValueListenerLoop(listener *ColumnValueListener) {
	for {
		tableEvent := <-listener.TableEvents
		for _, liveQuery := range listener.LiveQueries {
			writeNotificationAsJson(tableEvent, liveQuery.ResultWriter)
		}
	}
}

func writeNotificationAsJson(event *TableEvent, writer *bufio.Writer) {
	writer.WriteString("{\"old_record\":")
	if event.OldRecord == nil {
		writer.WriteString("null")
	} else {
		writer.WriteString(event.OldRecord.ToJson())
	}
	writer.WriteString(",\"new_record\":")
	if event.NewRecord == nil {
		writer.WriteString("null")
	} else {
		writer.WriteString(event.NewRecord.ToJson())
	}
	writer.WriteString("}\n")
	writer.Flush()
}
