package lang

import (
	"testing"

	"github.com/vilterp/treesql/package/util"
)

func TestInterpreter(t *testing.T) {
	userRootScope := NewScope(BuiltinsScope)
	userRootScope.Add("a", NewVInt(2))
	userRootScope.Add("b", NewVInt(3))
	userRootScope.Add("hello", NewVString("world"))
	userRootScope.Add("plus5", &vLambda{
		definedInScope: userRootScope,
		def: &ELambda{
			retType: TInt,
			params:  []Param{{"a", TInt}},
			body: &EFuncCall{
				funcName: "plus",
				args: []Expr{
					&EVar{name: "a"},
					NewIntLit(5),
				},
			},
		},
		// A little annoying that you have to repeat this, but...
		typ: &tFunction{
			params:  []Param{{"a", TInt}},
			retType: TInt,
		},
	})

	cases := []struct {
		expr    Expr
		typ     Type
		val     string
		typErr  string
		evalErr string
	}{
		// Basic func Call
		{
			expr: &EFuncCall{
				funcName: "plus",
				args: []Expr{
					&EVar{name: "a"},
					&EVar{name: "b"},
				},
			},
			typ: TInt,
			val: "5",
		},
		// Wrong arg #
		{
			expr: &EFuncCall{
				funcName: "plus",
				args: []Expr{
					&EVar{name: "a"},
				},
			},
			typ:    TInt,
			typErr: "plus: expected 2 args; given 1",
		},
		// Wrong arg types
		{
			expr: &EFuncCall{
				funcName: "plus",
				args: []Expr{
					&EVar{name: "hello"},
					NewStringLit("bla"),
				},
			},
			typErr: "call to plus, param 0: have string; want int",
		},
		// Nonexistent func
		{
			expr: &EFuncCall{
				funcName: "foo",
				args: []Expr{
					&EVar{name: "hello"},
					NewStringLit("bla"),
				},
			},
			typErr: "not in type scope: foo",
		},
		// Nonexistent arg
		{
			expr: &EFuncCall{
				funcName: "plus",
				args: []Expr{
					&EVar{name: "bloop"},
					NewStringLit("bla"),
				},
			},
			typErr: "not in type scope: bloop",
		},
		// Lambda Call
		{
			expr: &EFuncCall{
				funcName: "plus5",
				args: []Expr{
					&EVar{name: "a"},
				},
			},
			typ: TInt,
			val: "7",
		},
	}

	typeScope := userRootScope.ToTypeScope()

	// lord this error checking code is tedious
	for idx, testCase := range cases {
		interp := NewInterpreter(userRootScope, testCase.expr)
		// Typecheck
		typ, typErr := testCase.expr.GetType(typeScope)
		if util.AssertError(t, idx, testCase.typErr, typErr) {
			continue
		}
		if typ.Format().String() != testCase.typ.Format().String() {
			t.Errorf(
				`case %d: expected type "%s"; got "%s"`,
				idx, testCase.typ.Format(), typ.Format(),
			)
			continue
		}
		// Evaluate
		val, evalErr := interp.Interpret()
		if util.AssertError(t, idx, testCase.evalErr, evalErr) {
			continue
		}
		if val.Format().String() != testCase.val {
			t.Errorf(
				`case %d: expected value "%s"; got "%s"`,
				idx, testCase.val, val.Format(),
			)
		}
	}
}
