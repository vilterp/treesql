package treesql

type Statement struct {
	Select      *Select
	Insert      *Insert
	Update      *Update
	CreateTable *CreateTable
}

type CreateTable struct {
	Name    string
	Columns []*CreateTableColumn
}

type CreateTableColumn struct {
	Name       string
	TypeName   string
	PrimaryKey bool
	References *string
}

type Insert struct {
	Table  string
	Values []string
}

type Update struct {
	Table           string
	ColumnName      string
	Value           string
	WhereColumnName string
	EqualsValue     string
}

type Select struct {
	Many       bool
	Table      string
	Where      *Where
	Selections []*Selection
	Live       bool
}

type Where struct {
	ColumnName string
	Value      string
}

type Selection struct {
	Name      string
	SubSelect *Select
}
