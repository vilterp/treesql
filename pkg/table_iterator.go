package treesql

import (
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
)

// TODO: just make a common arrayIterator for internal tables

type TableIterator interface {
	Next() *record
	Get(key string) (*record, error)
	Close()
}

func (ex *selectExecution) getTableIterator(tableName string) (TableIterator, error) {
	if tableName == "__tables__" {
		return newTablesIterator(ex.Channel.connection.database)
	}
	if tableName == "__columns__" {
		return newColumnsIterator(ex.Channel.connection.database)
	}
	if tableName == "__record_listeners__" {
		return newRecordListenersIterator(ex.Channel.connection.database)
	}
	return newBoltIterator(ex, tableName)
}

// bolt iterator

type boltIterator struct {
	cursor        *bolt.Cursor
	seekedToFirst bool
	table         *tableDescriptor
}

func newBoltIterator(ex *selectExecution, tableName string) (*boltIterator, error) {
	tableSchema := ex.Channel.connection.database.schema.tables[tableName]
	cursor := ex.Transaction.Bucket([]byte(tableName)).Cursor()
	return &boltIterator{
		table:         tableSchema,
		seekedToFirst: false,
		cursor:        cursor,
	}, nil
}

func (it *boltIterator) Next() *record {
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

func (it *boltIterator) Get(key string) (*record, error) {
	_, rawRecord := it.cursor.Seek([]byte(key))
	return it.table.RecordFromBytes(rawRecord), nil
}

func (it *boltIterator) Close() {
	// I guess closing this is not a thing
}

// schema tables iterator

// oof, what if these change out from underneath the iterators?
// how do I clone the tables?
type schemaTablesIterator struct {
	db          *Database
	tablesArray []*tableDescriptor
	idx         int
}

func newTablesIterator(db *Database) (*schemaTablesIterator, error) {
	tables := make([]*tableDescriptor, len(db.schema.tables))
	i := 0
	for _, table := range db.schema.tables {
		tables[i] = table
		i++
	}
	return &schemaTablesIterator{
		db:          db,
		tablesArray: tables,
		idx:         0,
	}, nil
}

func (it *schemaTablesIterator) Next() *record {
	if it.idx == len(it.tablesArray) {
		return nil
	}
	table := it.tablesArray[it.idx]
	it.idx++
	return table.toRecord(it.db)
}

func (it *schemaTablesIterator) Get(key string) (*record, error) {
	table := it.db.schema.tables[key]
	return table.toRecord(it.db), nil
}

func (it *schemaTablesIterator) Close() {}

// schema columns iterator

type schemaColumnsIterator struct {
	db      *Database
	columns []*record
	idx     int
}

func newColumnsIterator(db *Database) (*schemaColumnsIterator, error) {
	columns := make([]*record, 0)
	for _, table := range db.schema.tables {
		for _, column := range table.columns {
			columnDoc := column.toRecord(table.name, db)
			columns = append(columns, columnDoc)
		}
	}
	return &schemaColumnsIterator{
		db:      db,
		columns: columns,
		idx:     0,
	}, nil
}

func (it *schemaColumnsIterator) Next() *record {
	if it.idx == len(it.columns) {
		return nil
	}
	columnDoc := it.columns[it.idx]
	it.idx++
	return columnDoc
}

func (it *schemaColumnsIterator) Get(key string) (*record, error) {
	// BUG: this is not stable if columns are dropped
	// need real OIDs
	// need sequences as a first-class DB object O_o
	idx, err := strconv.Atoi(key)
	if err != nil {
		return nil, nil
	}
	return it.columns[idx], nil
}

func (it *schemaColumnsIterator) Close() {}

// record listeners iterator

type recordListenersIterator struct {
	db        *Database
	listeners []*record
	idx       int
}

func newRecordListenersIterator(db *Database) (*recordListenersIterator, error) {
	listenersTable := db.schema.tables["__record_listeners__"]
	listeners := make([]*record, 0)
	i := 0
	for _, table := range db.schema.tables {
		table.liveQueryInfo.mu.RLock()
		defer table.liveQueryInfo.mu.RUnlock()

		for pkVal, listenerList := range table.liveQueryInfo.mu.RecordListeners {
			for connID, listenersForConn := range listenerList.Listeners {
				for statementID, listenersForStatement := range listenersForConn {
					for _, listener := range listenersForStatement {
						record := listenersTable.NewRecord()
						record.setString("id", fmt.Sprintf("%d", i)) // uh yeah these are not stable
						record.setString("connection_id", fmt.Sprintf("%d", connID))
						record.setString("channel_id", fmt.Sprintf("%d", statementID))
						record.setString("table_name", table.name)
						record.setString("pk_value", pkVal)
						record.setString("query_path", listener.QueryPath.String())
						listeners = append(listeners, record)
						i++
					}
				}
			}
		}
	}
	return &recordListenersIterator{
		db:        db,
		listeners: listeners,
		idx:       0,
	}, nil
}

func (it *recordListenersIterator) Next() *record {
	if it.idx == len(it.listeners) {
		return nil
	}
	columnDoc := it.listeners[it.idx]
	it.idx++
	return columnDoc
}

func (it *recordListenersIterator) Get(key string) (*record, error) {
	// BUG: this is not stable if columns are dropped
	// need real OIDs
	// need sequences as a first-class DB object O_o
	idx, err := strconv.Atoi(key)
	if err != nil {
		return nil, nil
	}
	return it.listeners[idx], nil
}

func (it *recordListenersIterator) Close() {}

// records iterator
