package lang

import (
	"testing"
)

func TestExprGetType(t *testing.T) {
	scope := NewScope(BuiltinsScope)

	blogPostType := &TObject{
		Types: map[string]Type{
			"id": TInt,
		},
	}

	scope.Add("blog_post", NewVObject(map[string]Value{
		"id":    NewVInt(2),
		"title": NewVString("hello world"),
	}))
	scope.Add("blog_posts", NewVIteratorRef(nil, blogPostType))

	testCases := []struct {
		in  Expr
		out string
	}{
		{
			NewMemberAccess(
				&EObjectLit{exprs: map[string]Expr{"x": NewIntLit(5)}},
				"x",
			),
			"int",
		},
		{
			NewMemberAccess(NewVar("blog_post"), "id"),
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
			"Iterator<int>",
		},
		// TODO: func call
		// TODO: func call of generic func
	}

	typeScope := scope.ToTypeScope()
	for idx, testCase := range testCases {
		actual, err := testCase.in.GetType(typeScope)
		// TODO: test errors
		if err != nil {
			t.Errorf("case %d: %v", idx, err)
			continue
		}
		if actual.Format().Render() != testCase.out {
			t.Errorf("case %d: expected type %s; got %s", idx, testCase.out, actual.Format().Render())
		}
	}
}
