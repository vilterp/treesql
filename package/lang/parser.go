package lang

import (
	"fmt"

	"strconv"

	p "github.com/vilterp/treesql/package/parserlib"
)

type recordKVPair struct {
	key   string
	value Expr
}

var rules = map[string]p.Rule{
	// Func call.
	"func_call": p.Map(
		p.Sequence([]p.Rule{
			p.Ref("var"),
			p.Keyword("("),
			p.Ref("arg_list"),
			p.Keyword(")"),
		}),
		func(tree *p.TraceTree) interface{} {
			return NewFuncCall("foo", []Expr{})
		},
	),
	"arg_list": p.ListRule(
		"expr",
		"arg_list",
		p.Sequence([]p.Rule{p.Keyword(","), p.OptWhitespace}),
	),

	"record_literal": p.Map(
		p.Sequence([]p.Rule{
			p.Keyword("{"),
			p.OptWhitespaceSurround(p.Ref("record_kv_pairs")),
			p.Keyword("}"),
		}),
		func(tree *p.TraceTree) interface{} {
			listT := tree.ItemTraces[1]
			exprs := map[string]Expr{}
			for _, kvInterface := range listT.ItemTraces[1].GetListRes() {
				kv := kvInterface.(*recordKVPair)
				exprs[kv.key] = kv.value
			}
			return &ERecordLit{
				exprs: exprs,
			}
		},
	),
	"record_kv_pairs": p.Sequence([]p.Rule{
		p.ListRule("record_kv_pair", "record_kv_pairs", p.CommaWhitespace),
	}),
	"record_kv_pair": p.Map(
		p.Sequence([]p.Rule{
			p.Ident,
			ColonWhitespace,
			p.Ref("expr"),
		}),
		func(tree *p.TraceTree) interface{} {
			return &recordKVPair{
				key:   tree.ItemTraces[0].RegexMatch,
				value: tree.ItemTraces[2].GetMapRes().(Expr),
			}
		},
	),

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
		p.Ref("record_literal"),
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
		return nil, fmt.Errorf("failed to cast %T to Expr", mapRes)
	}
	return expr, nil
}
