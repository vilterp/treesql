package treesql

import (
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/davecgh/go-spew/spew"
)

type TableIterator interface {
	Next() *Record
	Get(key string) (*Record, error)
	Close()
}

func (ex *QueryExecution) getTableIterator(tableName string) (TableIterator, error) {
	if tableName == "__tables__" {
		return newTablesIterator(ex.Connection.Database)
	} else if tableName == "__columns__" {
		return newColumnsIterator(ex.Connection.Database)
	}
	return newBoltIterator(ex, tableName)
}

// sophia iterator

type BoltIterator struct {
	cursor        *bolt.Cursor
	seekedToFirst bool
	table         *Table
}

func newBoltIterator(ex *QueryExecution, tableName string) (*BoltIterator, error) {
	tableSchema := ex.Connection.Database.Schema.Tables[tableName]
	cursor := ex.Transaction.Bucket([]byte(tableName)).Cursor()
	return &BoltIterator{
		table:         tableSchema,
		seekedToFirst: false,
		cursor:        cursor,
	}, nil
}

func (it *BoltIterator) Next() *Record {
	fmt.Println("======= Next =======")
	var key []byte
	var rawRecord []byte
	if !it.seekedToFirst {
		fmt.Println("about to call first")
		key, rawRecord = it.cursor.First()
		fmt.Println("first key", key)
		it.seekedToFirst = true
	} else {
		fmt.Println("about to call next")
		key, rawRecord = it.cursor.Next()
		fmt.Println("next key", key)
	}
	if key == nil {
		fmt.Println("key is nil")
		return nil
	} else {
		fmt.Println("key not nil")
		record := it.table.RecordFromBytes(rawRecord)
		spew.Dump(record)
		return record
	}
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
	tablesTable *Table
	tablesArray []*Table
	tablesMap   map[string]*Table
	idx         int
}

func newTablesIterator(db *Database) (*SchemaTablesIterator, error) {
	tables := make([]*Table, len(db.Schema.Tables))
	i := 0
	for _, table := range db.Schema.Tables {
		tables[i] = table
		i++
	}
	return &SchemaTablesIterator{
		tablesTable: db.Schema.Tables["__tables__"],
		tablesArray: tables,
		tablesMap:   db.Schema.Tables,
		idx:         0,
	}, nil
}

func (it *SchemaTablesIterator) Next() *Record {
	if it.idx == len(it.tablesArray) {
		return nil
	}
	table := it.tablesArray[it.idx]
	it.idx++
	return it.tableToRecord(table)
}

func (it *SchemaTablesIterator) Get(key string) (*Record, error) {
	return it.tableToRecord(it.tablesMap[key]), nil
}

func (it *SchemaTablesIterator) Close() {}

func (it *SchemaTablesIterator) tableToRecord(tableSpec *Table) *Record {
	return &Record{
		Table: it.tablesTable,
		Values: []Value{
			Value{ // name
				Type:      TypeString,
				StringVal: tableSpec.Name,
			},
			Value{ // primary_key
				Type:      TypeString,
				StringVal: tableSpec.PrimaryKey,
			},
		},
	}
}

// schema columns iterator

type SchemaColumnsIterator struct {
	columns []*Record
	idx     int
}

func newColumnsIterator(db *Database) (*SchemaColumnsIterator, error) {
	columnsTable := db.Schema.Tables["__columns__"]
	columns := make([]*Record, 0)
	for _, table := range db.Schema.Tables {
		for _, column := range table.Columns {
			columnDoc := columnToRecord(columnsTable, column, table)
			columns = append(columns, columnDoc)
		}
	}
	return &SchemaColumnsIterator{
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

func columnToRecord(columnsTable *Table, column *Column, memberOfTable *Table) *Record {
	var referencesColumn string // jeez, give me a freaking ternary already
	if column.ReferencesColumn != nil {
		referencesColumn = column.ReferencesColumn.TableName
	}
	return &Record{
		Table: columnsTable,
		Values: []Value{
			Value{ // name
				Type:      TypeString,
				StringVal: column.Name,
			},
			Value{ // table_name
				Type:      TypeString,
				StringVal: memberOfTable.Name,
			},
			Value{ // references
				Type:      TypeString,
				StringVal: referencesColumn,
			},
		},
	}
}
