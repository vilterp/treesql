package treesql

import (
	"strconv"

	sophia "github.com/pzhin/go-sophia"
)

type TableIterator interface {
	Next() *sophia.Document
	Get(key string) (*sophia.Document, error)
	Close()
}

func (db *Database) getTableIterator(tableName string) (TableIterator, error) {
	if tableName == "__tables__" {
		return newTablesIterator(db)
	} else if tableName == "__columns__" {
		return newColumnsIterator(db)
	}
	return newSophiaIterator(db, tableName)
}

// sophia iterator

type SophiaIterator struct {
	table  *sophia.Database
	cursor sophia.Cursor
}

func newSophiaIterator(db *Database, tableName string) (*SophiaIterator, error) {
	table := db.Dbs[tableName]
	doc := table.Document()
	cursor, err := table.Cursor(doc)
	return &SophiaIterator{
		table:  table,
		cursor: cursor,
	}, err
}

func (it *SophiaIterator) Next() *sophia.Document {
	result := it.cursor.Next()
	return result
}

func (it *SophiaIterator) Get(key string) (*sophia.Document, error) {
	doc := &sophia.Document{}
	return it.table.Get(doc)
}

func (it *SophiaIterator) Close() {
	it.cursor.Close()
}

// schema tables iterator

// oof, what if these change out from underneath the iterators?
// how do I clone the tables?
type SchemaTablesIterator struct {
	table       *sophia.Database
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
		tablesArray: tables,
		tablesMap:   db.Schema.Tables,
		idx:         0,
		table:       db.Dbs["__tables__"],
	}, nil
}

func (it *SchemaTablesIterator) Next() *sophia.Document {
	if it.idx == len(it.tablesArray) {
		return nil
	}
	table := it.tablesArray[it.idx]
	it.idx++
	return tableToDocument(it.table, table)
}

func (it *SchemaTablesIterator) Get(key string) (*sophia.Document, error) {
	return tableToDocument(it.table, it.tablesMap[key]), nil
}

func (it *SchemaTablesIterator) Close() {}

func tableToDocument(table *sophia.Database, tableSpec *Table) *sophia.Document {
	doc := table.Document()
	doc.SetString("name", tableSpec.Name)
	doc.SetString("primary_key", tableSpec.PrimaryKey)
	return doc
}

// schema columns iterator

type SchemaColumnsIterator struct {
	table   *sophia.Database
	columns []*sophia.Document
	idx     int
}

func newColumnsIterator(db *Database) (*SchemaColumnsIterator, error) {
	columns := make([]*sophia.Document, 0)
	for _, table := range db.Schema.Tables {
		for _, column := range table.Columns {
			columnDoc := columnToDocument(db.Dbs["__columns__"], column, table)
			columns = append(columns, columnDoc)
		}
	}
	return &SchemaColumnsIterator{
		columns: columns,
		idx:     0,
		table:   db.Dbs["__columns__"],
	}, nil
}

func (it *SchemaColumnsIterator) Next() *sophia.Document {
	if it.idx == len(it.columns) {
		return nil
	}
	columnDoc := it.columns[it.idx]
	it.idx++
	return columnDoc
}

func (it *SchemaColumnsIterator) Get(key string) (*sophia.Document, error) {
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

func columnToDocument(table *sophia.Database, column *Column, tableSpec *Table) *sophia.Document {
	doc := table.Document()
	doc.SetString("name", column.Name)
	if column.ReferencesColumn != nil {
		doc.SetString("references", column.ReferencesColumn.TableName)
	}
	// BUG: not having this sets it as empty string, which is not exactly what we want
	doc.SetString("table_name", tableSpec.Name)
	return doc
}
