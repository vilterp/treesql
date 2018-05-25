package lang

import (
	"testing"

	"github.com/vilterp/treesql/pkg/util"
)

func TestExprGetType(t *testing.T) {
	// Create scope.
	scope := BuiltinsScope.NewChildScope()

	blogPostType := NewTRecord(map[string]Type{
		"id":    TInt,
		"title": TString,
	})

	scope.Add("blog_post", NewVRecord(map[string]Value{
		"id":    NewVInt(2),
		"title": NewVString("hello world"),
	}))
	scope.Add("blog_posts", NewVIteratorRef(nil, blogPostType))
	scope.Add("plus", &VBuiltin{
		Name:    "plus",
		RetType: TInt,
		Params: []Param{
			{"a", TInt},
			{"b", TInt},
		},
		Impl: func(interp Caller, args []Value) (Value, error) {
			return NewVInt(5), nil // not actually plus
		},
	})

	// Cases.
	testCases := []struct {
		in    Expr
		error string
		out   string
	}{
		{
			NewMemberAccess(
				&ERecordLit{exprs: map[string]Expr{"x": NewIntLit(5)}},
				"x",
			),
			"",
			"int",
		},
		{
			NewMemberAccess(NewEVar("blog_post"), "id"),
			"",
			"int",
		},
		{
			NewEFuncCall("map", []Expr{
				NewEVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewMemberAccess(NewEVar("post"), "id"),
					TString,
				),
			}),
			"lambda declared as returning string; body is of type int",
			"",
		},
		{
			NewEFuncCall("map", []Expr{
				NewEVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewERecord(map[string]Expr{
						"id": NewMemberAccess(NewEVar("post"), "id"),
					}),
					NewTRecord(map[string]Type{
						"id": TInt,
					}),
				),
			}),
			"",
			`Iterator<{
  id: int,
}>`,
		},
		{
			NewEFuncCall("filter", []Expr{
				NewEVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewEFuncCall("intEq", []Expr{
						NewMemberAccess(NewEVar("post"), "id"),
						NewIntLit(5),
					}),
					TBool,
				),
			}),
			"",
			`Iterator<{
  id: int,
  title: string,
}>`,
		},
		{
			NewEFuncCall("filter", []Expr{
				NewEFuncCall("map", []Expr{
					NewEVar("blog_posts"),
					NewELambda(
						[]Param{{"post", blogPostType}},
						NewERecord(map[string]Expr{
							"id": NewMemberAccess(NewEVar("post"), "id"),
						}),
						NewTRecord(map[string]Type{
							"id": TInt,
						}),
					),
				}),
				NewELambda(
					[]Param{
						{
							"post",
							NewTRecord(map[string]Type{
								"id": TInt,
							}),
						},
					},
					NewEFuncCall("intEq", []Expr{
						NewMemberAccess(NewEVar("post"), "id"),
						NewIntLit(5),
					}),
					TBool,
				),
			}),
			"",
			`Iterator<{
  id: int,
}>`,
		},
		{
			NewEDoBlock(
				[]DoBinding{
					{
						"",
						NewEFuncCall("blerp", []Expr{}),
					},
				},
				NewIntLit(5),
			),
			"not in type scope: blerp",
			"",
		},
		{
			NewEDoBlock(
				[]DoBinding{
					{
						"",
						NewEFuncCall("plus", []Expr{
							NewIntLit(5), NewStringLit("bloop"),
						}),
					},
				},
				NewIntLit(5),
			),
			"call to plus, param 1: have string; want int",
			"",
		},
	}

	typeScope := scope.GetTypeScope()
	for idx, testCase := range testCases {
		actual, err := testCase.in.GetType(typeScope)
		if util.AssertError(t, idx, testCase.error, err) {
			continue
		}
		if actual.Format().String() != testCase.out {
			t.Errorf("case %d: expected:\n\n%s\n\ngot:\n\n%s", idx, testCase.out, actual.Format())
		}
	}
}
