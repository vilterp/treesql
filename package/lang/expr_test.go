package lang

import (
	"testing"

	"github.com/vilterp/treesql/package/util"
)

func TestExprGetType(t *testing.T) {
	// Create scope.
	scope := NewScope(BuiltinsScope)

	blogPostType := &TRecord{
		Types: map[string]Type{
			"id": TInt,
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
			"Iterator<int>",
		},
		// TODO: func call
		// TODO: func call of generic func
	}

	typeScope := scope.ToTypeScope()
	for idx, testCase := range testCases {
		actual, err := testCase.in.GetType(typeScope)
		if util.AssertError(t, idx, testCase.error, err) {
			continue
		}
		if actual.Format().String() != testCase.out {
			t.Errorf("case %d: expected type %s; got %s", idx, testCase.out, actual.Format())
		}
	}
}
