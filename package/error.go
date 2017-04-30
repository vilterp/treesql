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
