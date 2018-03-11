package lang

import (
	"fmt"

	"strconv"

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

	"member_access": p.Map(
		p.Sequence([]p.Rule{
			p.Ref("var"),
			p.Keyword("."),
			p.Ident,
		}),
		func(tree *p.TraceTree) interface{} {
			recordExpr, ok := tree.ItemTraces[0].GetMapRes().(Expr)
			if !ok {
				panic(fmt.Sprintf("failed to cast %T to expr", recordExpr))
			}
			member := tree.ItemTraces[2].RegexMatch
			return NewMemberAccess(recordExpr, member)
		},
	),

	// Primitives.
	"var": p.Map(p.Ident, func(tt *p.TraceTree) interface{} {
		return NewVar(tt.RegexMatch)
	}),
	"string_lit": p.Map(p.StringLit, func(tree *p.TraceTree) interface{} {
		return NewStringLit(tree.RegexMatch)
	}),
	"signed_int_lit": p.Map(p.SignedIntLit, func(tree *p.TraceTree) interface{} {
		val, err := strconv.Atoi(tree.RegexMatch)
		if err != nil {
			panic(fmt.Sprintf("err parsing int: %v", err))
		}
		return NewIntLit(val)
	}),

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

func Parse(input string) (Expr, error) {
	tree, err := Grammar.Parse("expr", input)
	if err != nil {
		return nil, err
	}

	mapRes := tree.GetMapRes()
	expr, ok := mapRes.(Expr)
	if !ok {
		fmt.Printf("TRACE: %+v\n", tree)
		return nil, fmt.Errorf("failed to cast %T to Expr", mapRes)
	}
	return expr, nil
}
