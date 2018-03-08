package treesql

import (
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
)

// TODO: just make a common arrayIterator for internal tables

type TableIterator interface {
	Next() *Record
	Get(key string) (*Record, error)
	Close()
}

func (ex *SelectExecution) getTableIterator(tableName string) (TableIterator, error) {
	if tableName == "__tables__" {
		return newTablesIterator(ex.Channel.Connection.Database)
	}
	if tableName == "__columns__" {
		return newColumnsIterator(ex.Channel.Connection.Database)
	}
	if tableName == "__record_listeners__" {
		return newRecordListenersIterator(ex.Channel.Connection.Database)
	}
	return newBoltIterator(ex, tableName)
}

// bolt iterator

type BoltIterator struct {
	cursor        *bolt.Cursor
	seekedToFirst bool
	table         *TableDescriptor
}

func newBoltIterator(ex *SelectExecution, tableName string) (*BoltIterator, error) {
	tableSchema := ex.Channel.Connection.Database.Schema.Tables[tableName]
	cursor := ex.Transaction.Bucket([]byte(tableName)).Cursor()
	return &BoltIterator{
		table:         tableSchema,
		seekedToFirst: false,
		cursor:        cursor,
	}, nil
}

func (it *BoltIterator) Next() *Record {
	var key []byte
	var rawRecord []byte
	if !it.seekedToFirst {
		key, rawRecord = it.cursor.First()
		it.seekedToFirst = true
	} else {
		key, rawRecord = it.cursor.Next()
	}
	if key == nil {
		return nil
	}
	record := it.table.RecordFromBytes(rawRecord)
	return record
}

func (it *BoltIterator) Get(key string) (*Record, error) {
	_, rawRecord := it.cursor.Seek([]byte(key))
	return it.table.RecordFromBytes(rawRecord), nil
}

func (it *BoltIterator) Close() {
	// I guess closing this is not a thing
}

// schema tables iterator

// oof, what if these change out from underneath the iterators?
// how do I clone the tables?
type SchemaTablesIterator struct {
	db          *Database
	tablesArray []*TableDescriptor
	idx         int
}

func newTablesIterator(db *Database) (*SchemaTablesIterator, error) {
	tables := make([]*TableDescriptor, len(db.Schema.Tables))
	i := 0
	for _, table := range db.Schema.Tables {
		tables[i] = table
		i++
	}
	return &SchemaTablesIterator{
		db:          db,
		tablesArray: tables,
		idx:         0,
	}, nil
}

func (it *SchemaTablesIterator) Next() *Record {
	if it.idx == len(it.tablesArray) {
		return nil
	}
	table := it.tablesArray[it.idx]
	it.idx++
	return table.ToRecord(it.db)
}

func (it *SchemaTablesIterator) Get(key string) (*Record, error) {
	table := it.db.Schema.Tables[key]
	return table.ToRecord(it.db), nil
}

func (it *SchemaTablesIterator) Close() {}

// schema columns iterator

type SchemaColumnsIterator struct {
	db      *Database
	columns []*Record
	idx     int
}

func newColumnsIterator(db *Database) (*SchemaColumnsIterator, error) {
	columns := make([]*Record, 0)
	for _, table := range db.Schema.Tables {
		for _, column := range table.Columns {
			columnDoc := column.ToRecord(table.Name, db)
			columns = append(columns, columnDoc)
		}
	}
	return &SchemaColumnsIterator{
		db:      db,
		columns: columns,
		idx:     0,
	}, nil
}

func (it *SchemaColumnsIterator) Next() *Record {
	if it.idx == len(it.columns) {
		return nil
	}
	columnDoc := it.columns[it.idx]
	it.idx++
	return columnDoc
}

func (it *SchemaColumnsIterator) Get(key string) (*Record, error) {
	// BUG: this is not stable if columns are dropped
	// need real OIDs
	// need sequences as a first-class DB object O_o
	idx, err := strconv.Atoi(key)
	if err != nil {
		return nil, nil
	}
	return it.columns[idx], nil
}

func (it *SchemaColumnsIterator) Close() {}

// record listeners iterator

type RecordListenersIterator struct {
	db        *Database
	listeners []*Record
	idx       int
}

func newRecordListenersIterator(db *Database) (*RecordListenersIterator, error) {
	listenersTable := db.Schema.Tables["__record_listeners__"]
	listeners := make([]*Record, 0)
	i := 0
	for _, table := range db.Schema.Tables {
		table.LiveQueryInfo.mu.RLock()
		defer table.LiveQueryInfo.mu.RUnlock()

		for pkVal, listenerList := range table.LiveQueryInfo.mu.RecordListeners {
			for connID, listenersForConn := range listenerList.Listeners {
				for statementID, listenersForStatement := range listenersForConn {
					for _, listener := range listenersForStatement {
						record := listenersTable.NewRecord()
						record.SetString("id", fmt.Sprintf("%d", i)) // uh yeah these are not stable
						record.SetString("connection_id", fmt.Sprintf("%d", connID))
						record.SetString("channel_id", fmt.Sprintf("%d", statementID))
						record.SetString("table_name", table.Name)
						record.SetString("pk_value", pkVal)
						record.SetString("query_path", listener.QueryPath.String())
						listeners = append(listeners, record)
						i++
					}
				}
			}
		}
	}
	return &RecordListenersIterator{
		db:        db,
		listeners: listeners,
		idx:       0,
	}, nil
}

func (it *RecordListenersIterator) Next() *Record {
	if it.idx == len(it.listeners) {
		return nil
	}
	columnDoc := it.listeners[it.idx]
	it.idx++
	return columnDoc
}

func (it *RecordListenersIterator) Get(key string) (*Record, error) {
	// BUG: this is not stable if columns are dropped
	// need real OIDs
	// need sequences as a first-class DB object O_o
	idx, err := strconv.Atoi(key)
	if err != nil {
		return nil, nil
	}
	return it.listeners[idx], nil
}

func (it *RecordListenersIterator) Close() {}

// records iterator
