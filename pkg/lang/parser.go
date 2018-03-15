package lang

import (
	"fmt"

	"strconv"

	p "github.com/vilterp/treesql/pkg/parserlib"
)

type recordKVPair struct {
	key   string
	value Expr
}

var rules = map[string]p.Rule{
	// Func call.
	"func_call": p.Map(
		p.Sequence([]p.Rule{
			p.Ident,
			p.Keyword("("),
			p.OptWhitespaceSurround(p.Ref("arg_list")),
			p.Keyword(")"),
		}),
		func(tree *p.TraceTree) interface{} {
			// Get name.
			name := tree.ItemTraces[0].RegexMatch
			// Get param list.
			inParens := tree.ItemTraces[2]
			inWhitespace := inParens.OptWhitespaceSurroundRes()
			exprIs := inWhitespace.GetMapRes().([]interface{})
			exprs := make([]Expr, len(exprIs))
			for idx, exprI := range exprIs {
				// Don't understand why we can cast the individual but not the array...
				exprs[idx] = exprI.(Expr)
			}
			return NewFuncCall(name, exprs)
		},
	),
	"arg_list": p.Map(
		p.ListRule(
			"expr",
			"arg_list",
			p.CommaOptWhitespace,
		),
		func(tree *p.TraceTree) interface{} {
			return tree.GetListRes()
		},
	),

	// Record lit.
	"record_literal": p.Map(
		p.Sequence([]p.Rule{
			p.Keyword("{"),
			p.OptWhitespaceSurround(p.Ref("record_kv_pairs")),
			p.Keyword("}"),
		}),
		func(tree *p.TraceTree) interface{} {
			// Unwrap to get to list result.
			betweenCurlies := tree.ItemTraces[1]
			unwrapWS := betweenCurlies.OptWhitespaceSurroundRes()
			kvs := unwrapWS.GetMapRes().([]interface{})
			// Build map.
			exprs := map[string]Expr{}
			for _, kvInterface := range kvs {
				kv := kvInterface.(*recordKVPair)
				exprs[kv.key] = kv.value
			}
			return &ERecordLit{
				exprs: exprs,
			}
		},
	),
	"record_kv_pairs": p.Map(
		p.ListRule("record_kv_pair", "record_kv_pairs", p.CommaOptWhitespace),
		func(tree *p.TraceTree) interface{} {
			return tree.GetListRes()
		},
	),
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
	"lambda": p.Map(
		p.Sequence([]p.Rule{
			p.Keyword("("),
			p.Ref("param_list"),
			p.Keyword("): "),
			p.Ref("type"),
			p.Keyword(" => "),
			p.Ref("expr"),
		}),
		func(tree *p.TraceTree) interface{} {
			// Get param list.
			paramIs := tree.ItemTraces[1].RefTrace.GetMapRes().([]interface{})
			params := make(paramList, len(paramIs))
			for idx, paramI := range paramIs {
				params[idx] = paramI.(Param)
			}
			// Get type.
			typ := tree.ItemTraces[3].GetMapRes().(Type)
			// Get expr.
			expr := tree.ItemTraces[5].GetMapRes().(Expr)
			return NewELambda(params, expr, typ)
		},
	),
	"param": p.Map(
		p.Sequence([]p.Rule{
			p.Ident,
			p.Keyword(": "),
			p.Ref("type"),
		}),
		func(tree *p.TraceTree) interface{} {
			return Param{
				Name: tree.ItemTraces[0].RegexMatch,
				Typ:  tree.ItemTraces[2].GetMapRes().(Type),
			}
		},
	),
	"param_list": p.Map(
		p.ListRule("param", "param_list", p.CommaOptWhitespace),
		func(tree *p.TraceTree) interface{} {
			return tree.GetListRes()
		},
	),

	// Member access.
	"member_access": p.Map(
		p.Sequence([]p.Rule{
			p.Ref("var"),
			p.Keyword("."),
			p.Ident,
		}),
		func(tree *p.TraceTree) interface{} {
			recordExpr := tree.ItemTraces[0].GetMapRes().(Expr)
			member := tree.ItemTraces[2].RegexMatch
			return NewMemberAccess(recordExpr, member)
		},
	),

	// Primitives.
	"var": p.Map(
		p.Ident,
		func(tt *p.TraceTree) interface{} {
			return NewVar(tt.RegexMatch)
		},
	),
	"string_lit": p.Map(
		p.StringLit,
		func(tree *p.TraceTree) interface{} {
			return NewStringLit(tree.RegexMatch)
		},
	),
	"signed_int_lit": p.Map(
		p.SignedIntLit,
		func(tree *p.TraceTree) interface{} {
			val, err := strconv.Atoi(tree.RegexMatch)
			if err != nil {
				panic(fmt.Sprintf("err parsing int: %v", err))
			}
			return NewIntLit(val)
		},
	),

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

	// Type.
	// TODO: choice:
	// - simple_name
	// - iterator
	// - index
	// - object lit
	// (maybe some day)
	// - dicts (for group by)
	// - returns
	"type": p.Map(
		p.Ident,
		func(tree *p.TraceTree) interface{} {
			// TODO: return a type expression; resolve it later
			str := tree.RegexMatch
			switch str {
			case "int":
				return TInt
			case "string":
				return TString
			default:
				panic(fmt.Sprintf("cannot parse type %s", str))
			}
		},
	),
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
