package lang

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vilterp/treesql/pkg/lang/typecheck"
	"github.com/vilterp/treesql/pkg/util"
)

func TestExprGetType(t *testing.T) {
	// Create scope.
	scope := BuiltinsScope.NewChildScope()

	blogPostType := &TRecord{
		types: map[string]Type{
			"id":    TInt,
			"title": TString,
		},
	}

	scope.Add("blog_post", NewVRecord(map[string]Value{
		"id":    NewVInt(2),
		"title": NewVString("hello world"),
	}))
	scope.Add("blog_posts", NewVIteratorRef(nil, blogPostType))

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
			NewMemberAccess(NewVar("blog_post"), "id"),
			"",
			"int",
		},
		{
			NewFuncCall("map", []Expr{
				NewVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewMemberAccess(NewVar("post"), "id"),
					TString,
				),
			}),
			"lambda declared as returning string; body is of type int",
			"",
		},
		{
			NewFuncCall("map", []Expr{
				NewVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewRecordLit(map[string]Expr{
						"id": NewMemberAccess(NewVar("post"), "id"),
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
			NewFuncCall("filter", []Expr{
				NewVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewFuncCall("intEq", []Expr{
						NewMemberAccess(NewVar("post"), "id"),
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
			NewFuncCall("filter", []Expr{
				NewFuncCall("map", []Expr{
					NewVar("blog_posts"),
					NewELambda(
						[]Param{{"post", blogPostType}},
						NewRecordLit(map[string]Expr{
							"id": NewMemberAccess(NewVar("post"), "id"),
						}),
						NewTRecord(map[string]Type{
							"id": TInt,
						}),
					),
				}),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewFuncCall("intEq", []Expr{
						NewMemberAccess(NewVar("post"), "id"),
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
	}

	typeScope := scope.toTypeScope()
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

func TestExprHashable(t *testing.T) {
	blogPostType := &TRecord{
		types: map[string]Type{
			"id":    TInt,
			"title": TString,
		},
	}

	testCases := []struct {
		in       Expr
		hashable HashableExpr
		hashed   typecheck.Hash
	}{
		{
			NewMemberAccess(
				&ERecordLit{exprs: map[string]Expr{"x": NewIntLit(5)}},
				"x",
			),
			"",
			123,
		},
		{
			NewMemberAccess(NewVar("blog_post"), "id"),
			"",
			456,
		},
		{
			NewFuncCall("map", []Expr{
				NewVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewMemberAccess(NewVar("post"), "id"),
					TString,
				),
			}),
			"",
			789,
		},
		{
			NewFuncCall("map", []Expr{
				NewVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewRecordLit(map[string]Expr{
						"id": NewMemberAccess(NewVar("post"), "id"),
					}),
					NewTRecord(map[string]Type{
						"id": TInt,
					}),
				),
			}),
			"",
			012,
		},
		{
			NewFuncCall("filter", []Expr{
				NewVar("blog_posts"),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewFuncCall("intEq", []Expr{
						NewMemberAccess(NewVar("post"), "id"),
						NewIntLit(5),
					}),
					TBool,
				),
			}),
			"",
			345,
		},
		{
			NewFuncCall("filter", []Expr{
				NewFuncCall("map", []Expr{
					NewVar("blog_posts"),
					NewELambda(
						[]Param{{"post", blogPostType}},
						NewRecordLit(map[string]Expr{
							"id": NewMemberAccess(NewVar("post"), "id"),
						}),
						NewTRecord(map[string]Type{
							"id": TInt,
						}),
					),
				}),
				NewELambda(
					[]Param{{"post", blogPostType}},
					NewFuncCall("intEq", []Expr{
						NewMemberAccess(NewVar("post"), "id"),
						NewIntLit(5),
					}),
					TBool,
				),
			}),
			"",
			678,
		},
	}

	for idx, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			hashable := testCase.in.Hashable()
			require.Equal(t, testCase.hashable, hashable)
			require.Equal(t, testCase.hashed, hashable.Hash())
		})
	}
}
