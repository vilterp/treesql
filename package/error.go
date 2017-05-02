package treesql

import "fmt"

type NoSuchTable struct {
	TableName string
}

func (e *NoSuchTable) Error() string {
	return fmt.Sprintf("no such table: %s", e.TableName)
}

type NoSuchColumn struct {
	TableName  string
	ColumnName string
}

func (e *NoSuchColumn) Error() string {
	return fmt.Sprintf("no such column in table %s: %s", e.TableName, e.ColumnName)
}

type BuiltinWriteAttempt struct {
	TableName string
}

func (e *BuiltinWriteAttempt) Error() string {
	return fmt.Sprintf("attemtped to write to %s, but builtin tables are read-only", e.TableName)
}

type InsertWrongNumFields struct {
	TableName string
	Wanted    int
	Got       int
}

func (e *InsertWrongNumFields) Error() string {
	return fmt.Sprintf("table %s has %d columns, but insert statement provided %d", e.TableName, e.Wanted, e.Got)
}
