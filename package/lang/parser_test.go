package lang

import (
	"testing"
)

func TestParser(t *testing.T) {
	cases := []string{
		// var
		`blerp`,
		`42`,
		// member access
		`foo.bar`,
		//// func call
		//{`foo(bar, baz)`},
		//// obj lit
		//{`{bloop:2}`},
		//{`{bloop: 2}`},
		//{`{ bloop: 2 }`},
		//{`{ bloop: 2, gloop: "bloop" }`},
		//// lambda
		//{`(foo) => plus(foo, bar)`},
		//{`(foo, bar) => plus(foo, bar)`},
		//{`map(blog_posts.by_id, (post) => {
		//  id: post.id,
		//  title: post.title
		//})`},
	}

	for idx, testCase := range cases {
		resExpr, err := Parse(testCase)
		if err != nil {
			t.Errorf("case %d: err: %v", idx, err)
		}

		if resExpr.Format().String() != testCase {
			t.Errorf("case %d: expected `%v`; got `%v`", idx, testCase, resExpr.Format().String())
		}
	}
}
