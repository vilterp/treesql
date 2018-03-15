package treesql

import "fmt"

// TODO: these aren't always used. Remove them or always use them.

type noSuchTable struct {
	TableName string
}

func (e *noSuchTable) Error() string {
	return fmt.Sprintf("no such table: %s", e.TableName)
}

type noSuchColumn struct {
	TableName  string
	ColumnName string
}

func (e *noSuchColumn) Error() string {
	return fmt.Sprintf("no such column in table %s: %s", e.TableName, e.ColumnName)
}

type builtinWriteAttempt struct {
	TableName string
}

func (e *builtinWriteAttempt) Error() string {
	return fmt.Sprintf("attemtped to write to %s, but builtin tables are read-only", e.TableName)
}

type insertWrongNumFields struct {
	TableName string
	Wanted    int
	Got       int
}

func (e *insertWrongNumFields) Error() string {
	return fmt.Sprintf("table %s has %d columns, but insert statement provided %d", e.TableName, e.Wanted, e.Got)
}

type tableAlreadyExists struct {
	TableName string
}

func (e *tableAlreadyExists) Error() string {
	return fmt.Sprintf("table already exists: %s", e.TableName)
}

type nonexistentType struct {
	TypeName string
}

func (e *nonexistentType) Error() string {
	return fmt.Sprintf("nonexistent type: %s", e.TypeName)
}

type wrongNoPrimaryKey struct {
	Count int
}

func (e *wrongNoPrimaryKey) Error() string {
	return fmt.Sprintf("tables should have exactly one column marked \"primary key\"; given %d", e.Count)
}

type noReferenceForJoin struct {
	FromTable string
	ToTable   string
}

func (e *noReferenceForJoin) Error() string {
	return fmt.Sprintf("query requires a column in table `%s` referencing table `%s`; none found", e.FromTable, e.ToTable)
}

// TODO: maybe just use errors.Wrap for these

type parseError struct {
	error error
}

func (e *parseError) Error() string {
	return fmt.Sprintf("parse error: %s", e.error.Error())
}

type validationError struct {
	error error
}

func (e *validationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.error.Error())
}

type recordAlreadyExists struct {
	ColName string
	Val     string
}

func (e *recordAlreadyExists) Error() string {
	return fmt.Sprintf("record already exists with primary key %s=%s", e.ColName, e.Val)
}
