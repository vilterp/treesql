package lang

import (
	"fmt"

	p "github.com/vilterp/treesql/package/parserlib"
)

var rules = map[string]p.Rule{
	// Func call.
	"func_call": p.Sequence([]p.Rule{
		p.Ref("var"),
		p.Keyword("("),
		p.Ref("arg_list"),
		p.Keyword(")"),
	}),
	"arg_list": p.ListRule(
		"expr",
		"arg_list",
		p.Sequence([]p.Rule{p.Keyword(","), p.OptWhitespace}),
	),

	"object_literal": p.Sequence([]p.Rule{
		p.Keyword("{"),
		p.OptWhitespaceSurround(p.Ref("obj_kv_pairs")),
		p.Keyword("}"),
	}),
	"obj_kv_pairs": p.Sequence([]p.Rule{
		p.ListRule("obj_kv_pair", "obj_kv_pairs", p.CommaWhitespace),
	}),
	"obj_kv_pair": p.Sequence([]p.Rule{
		p.Ident,
		ColonWhitespace,
		p.Ref("expr"),
	}),

	// Lambda.
	"lambda": p.Sequence([]p.Rule{
		p.Keyword("("),
		p.Ref("param_list"),
		p.Keyword(") => "),
		p.Ref("expr"),
	}),
	"param":      p.Ref("var"),
	"param_list": p.ListRule("param", "param_list", p.CommaWhitespace),

	"member_access": p.Sequence([]p.Rule{
		p.Ref("var"),
		p.Keyword("."),
		p.Ident,
	}),

	// Primitives.
	"var":            p.Ident,
	"string_lit":     p.StringLit,
	"signed_int_lit": p.SignedIntLit,

	// Expression.
	"expr": p.Choice([]p.Rule{
		p.Ref("func_call"),
		p.Ref("member_access"),
		p.Ref("var"),
		p.Ref("object_literal"),
		p.Ref("lambda"),
		p.Ref("string_lit"),
		p.Ref("signed_int_lit"),
	}),
}

var ColonWhitespace = p.Sequence([]p.Rule{p.Keyword(":"), p.OptWhitespace})

var Grammar *p.Grammar

func init() {
	g, err := p.NewGrammar(rules)
	if err != nil {
		panic(fmt.Sprintf("grammar error: %v", err))
	}
	Grammar = g
}

func Parse(input string) (*p.TraceTree, error) {
	return Grammar.Parse("expr", input)
}
