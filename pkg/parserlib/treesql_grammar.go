package parserlib

var treeSQLGrammarRules = map[string]Rule{
	"select": Sequence([]Rule{
		Choice([]Rule{
			Keyword("ONE"),
			Keyword("MANY"),
		}),
		Whitespace,
		Ref("table_name"),
		Whitespace,
		Opt(Ref("where_clause")),
		OptWhitespace,
		Ref("selection"),
	}),
	"table_name":  Ident,
	"column_name": Ident,
	"where_clause": Sequence([]Rule{
		Keyword("WHERE"),
		Whitespace,
		Ref("column_name"),
		OptWhitespace,
		Keyword("="),
		OptWhitespace,
		Ref("expr"),
	}),
	"selection": Sequence([]Rule{
		Keyword("{"),
		OptWhitespaceSurround(
			Ref("selection_fields"),
		),
		Keyword("}"),
	}),
	// TODO: intercalate combinator (??)
	"selection_fields": ListRule(
		"selection_field",
		"selection_fields",
		Sequence([]Rule{Keyword(","), OptWhitespace}),
	),
	"selection_field": Sequence([]Rule{
		Ref("column_name"),
		Opt(Sequence([]Rule{
			Keyword(":"),
			OptWhitespace,
			Ref("select"),
		})),
	}),
	"expr": Choice([]Rule{
		Ident,
		StringLit,
		SignedIntLit,
	}),
}

func TestTreeSQLGrammar() (*Grammar, error) {
	return NewGrammar(treeSQLGrammarRules)
}
