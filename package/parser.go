package treesql

import (
	"github.com/alecthomas/participle"
	"github.com/alecthomas/participle/lexer"
)

var (
	sqlLexer = lexer.Unquote(lexer.Upper(lexer.Must(lexer.Regexp(`(\s+)`+
		`|(?P<Keyword>(?i)SELECT|FROM|TOP|DISTINCT|ALL|WHERE|GROUP|BY|HAVING|UNION|MINUS|EXCEPT|INTERSECT|ORDER|LIMIT|OFFSET|TRUE|FALSE|NULL|IS|NOT|ANY|SOME|BETWEEN|AND|OR|LIKE|AS|IN)`+
		`|(?P<Ident>[a-zA-Z_][a-zA-Z0-9_]*)`+
		`|(?P<Number>[-+]?\d*\.?\d+([eE][-+]?\d+)?)`+
		`|(?P<String>'[^']*'|"[^"]*")`+
		`|(?P<Operators><>|!=|<=|>=|[-+*/%,.()=<>])`,
	)), "Keyword"), "String")
	sqlParser = participle.MustBuild(&Select{}, sqlLexer)
)

type Select struct {
	From *From `"SELECT" "*" "FROM" @@`
}

type From struct {
	Table string `@Ident`
}

// Parse parses sql
func Parse(sql string) (*Select, error) {
	result := &Select{}
	err := sqlParser.ParseString(sql, result)
	return result, err
}