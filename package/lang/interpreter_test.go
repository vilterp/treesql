package lang

import "testing"

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
	})

	cases := []struct {
		expr    Expr
		typ     Type
		val     string
		typErr  string
		evalErr string
	}{
		// Basic func call
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
			typErr: "not in scope: foo",
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
			typErr: "not in scope: bloop",
		},
		// Lambda call
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

	// lord this error checking code is tedious
	for idx, testCase := range cases {
		interp := newInterpreter(userRootScope, testCase.expr)
		// Typecheck
		typ, typErr := testCase.expr.GetType(userRootScope)
		if typErr == nil {
			if testCase.typErr != "" {
				t.Errorf(`case %d: expected type error "%s"; got none`, idx, testCase.typErr)
				continue
			}
		} else {
			if typErr.Error() != testCase.typErr {
				t.Errorf(`case %d: expected type error "%s"; got "%s"`, idx, testCase.typErr, typErr)
				continue
			}
			// typeErr not nil; matches case's error
			continue
		}
		if typ.Format().Render() != testCase.typ.Format().Render() {
			t.Errorf(
				`case %d: expected type "%s"; got "%s"`,
				idx, testCase.typ.Format().Render(), typ.Format().Render(),
			)
			continue
		}
		// Evaluate
		val, evalErr := interp.interpret()
		if evalErr == nil {
			if testCase.evalErr != "" {
				t.Errorf(`case %d: expected eval error "%s"; got none`, idx, evalErr.Error())
				continue
			}
		} else if evalErr.Error() != testCase.evalErr {
			t.Errorf(`case %d: expected eval error "%s"; got "%s"`, idx, testCase.evalErr, evalErr)
			continue
		}
		if val.Format().Render() != testCase.val {
			t.Errorf(
				`case %d: expected value "%s"; got "%s"`,
				idx, testCase.val, val.Format().Render(),
			)
		}
	}
}
