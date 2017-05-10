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

type TableAlreadyExists struct {
	TableName string
}

func (e *TableAlreadyExists) Error() string {
	return fmt.Sprintf("table already exists: %s", e.TableName)
}

type NonexistentType struct {
	TypeName string
}

func (e *NonexistentType) Error() string {
	return fmt.Sprintf("nonexistent type:", e.TypeName)
}

type WrongNoPrimaryKey struct {
	Count int
}

func (e *WrongNoPrimaryKey) Error() string {
	return fmt.Sprintf("tables should have exactly one column marked \"primary key\"; given %d", e.Count)
}

type NoReferenceForJoin struct {
	FromTable string
	ToTable   string
}

func (e *NoReferenceForJoin) Error() string {
	return fmt.Sprintf("query requires a column in table `%s` referencing table `%s`; none found", e.FromTable, e.ToTable)
}
