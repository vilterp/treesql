package treesql_lang

import p "github.com/vilterp/treesql/pkg/parserlib"

var Grammar *p.Grammar

func init() {
	grammar, err := p.NewGrammar(grammarRules)
	if err != nil {
		panic(err)
	}
	Grammar = grammar
}

var grammarRules = map[string]p.Rule{
	p.StartRuleName: p.Ref("select"),
	"select": p.Sequence([]p.Rule{
		p.Choice([]p.Rule{
			p.Keyword("ONE"),
			p.Keyword("MANY"),
		}),
		p.Whitespace,
		p.Ref("table_name"),
		p.Whitespace,
		p.Opt(p.Ref("where_clause")),
		p.OptWhitespace,
		p.Ref("selection"),
	}),
	"table_name":  p.Ident,
	"column_name": p.Ident,
	"where_clause": p.Sequence([]p.Rule{
		p.Keyword("WHERE"),
		p.Whitespace,
		p.Ref("column_name"),
		p.OptWhitespace,
		p.Keyword("="),
		p.OptWhitespace,
		p.Ref("expr"),
	}),
	"selection": p.Sequence([]p.Rule{
		p.Keyword("{"),
		p.OptWhitespaceSurround(
			p.Ref("selection_fields"),
		),
		p.Keyword("}"),
	}),
	// TODO: intercalate combinator (??)
	"selection_fields": p.ListRule(
		"selection_field",
		"selection_fields",
		p.Sequence([]p.Rule{p.Keyword(","), p.OptWhitespace}),
	),
	"selection_field": p.Sequence([]p.Rule{
		p.Ref("column_name"),
		p.Opt(p.Sequence([]p.Rule{
			p.Keyword(":"),
			p.OptWhitespace,
			p.Ref("select"),
		})),
	}),
	"expr": p.Choice([]p.Rule{
		p.Ident,
		p.StringLit,
		p.SignedIntLit,
	}),
}
