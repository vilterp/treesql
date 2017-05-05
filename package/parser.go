package treesql

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

var (
	sqlLexer = lexer.Unquote(
		lexer.Upper(
			lexer.Must(
				lexer.Regexp(`(\s+)`+
					`|(?P<Keyword>(?i)LIVE|SELECT|INSERT|INTO|VALUES|CREATETABLE|PRIMARYKEY|REFERENCESTABLE|ONE|MANY|FROM|TOP|DISTINCT|ALL|WHERE|GROUP|BY|HAVING|UNION|MINUS|EXCEPT|INTERSECT|ORDER|LIMIT|OFFSET|TRUE|FALSE|NULL|IS|NOT|ANY|SOME|BETWEEN|AND|OR|LIKE|AS)`+
					`|(?P<Ident>[a-zA-Z_][a-zA-Z0-9_]*)`+
					`|(?P<Number>[-+]?\d*\.?\d+([eE][-+]?\d+)?)`+
					`|(?P<String>'[^']*'|"[^"]*")`+
					`|(?P<Operators><>|!=|<=|>=|[-+*/%,.()\{\}=<>:])`,
				),
			),
			"Keyword",
		),
		"String",
	)
	sqlParser = participle.MustBuild(&Statement{}, sqlLexer)
)

type Statement struct {
	Select      *Select      `  @@`
	Insert      *Insert      `| @@`
	CreateTable *CreateTable `| @@`
}

type CreateTable struct {
	Name    string               `"CREATETABLE" @Ident` // parser can't distinguish idents and keywords
	Columns []*CreateTableColumn `"(" @@ { "," @@ } ")"`
}

type CreateTableColumn struct {
	Name       string  `@Ident`
	TypeName   string  `@Ident`
	PrimaryKey bool    `[ @"PRIMARYKEY"`              // parser can't distinguish idents and keywords
	References *string `| "REFERENCESTABLE" @Ident ]` // parser can't distinguish idents and keywords
}

type Insert struct {
	Table  string   `"INSERT" "INTO" @Ident`
	Values []string `"VALUES" "(" @String { "," @String } ")"`
}

type Select struct {
	Live       bool         `[ @"LIVE" ]`
	Many       bool         `( @"MANY"`
	One        bool         `| @"ONE" )`
	Table      string       `@Ident`
	Where      *Where       `[ "WHERE" @@ ]`
	Selections []*Selection `"{" @@ { "," @@ } "}"` // TODO: * for all columns
}

type Where struct {
	ColumnName string `@Ident "="`
	Value      string `@String`
}

type Selection struct {
	Name      string  `@Ident`
	SubSelect *Select `[ ":" @@ ]`
}

// Parse parses sql
func Parse(sql string) (*Statement, error) {
	result := &Statement{}
	err := sqlParser.ParseString(sql, result)
	return result, err
}
