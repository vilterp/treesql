package lang

import (
	"testing"
)

func TestParser(t *testing.T) {
	cases := []string{
		//var
		`blerp`,
		`42`,
		// member access
		`foo.bar`,
		// func call
		`foo()`,
		`foo(2, 3)`,
		`foo(bar, baz)`,
		// obj lit
		`{}`,
		`{
  bloop: 2
}`,
		`{
  bloop: 2,
  gloop: 3
}`,
		// lambda
		`(): int => 2`,
		`(): int => plus(foo, bar)`,
		`(foo: int, bar: int): int => plus(foo, bar)`,
		// TODO: handle type aliases... ugh
		`map(blog_posts.by_id, (post: string): int => {
  id: post.id,
  title: post.title
})`,
	}

	for idx, testCase := range cases {
		resExpr, err := Parse(testCase)
		if err != nil {
			t.Errorf("case %d: `%s` err: %v", idx, testCase, err)
			continue
		}

		if resExpr.Format().String() != testCase {
			t.Errorf("case %d: expected `%v`; got `%v`", idx, testCase, resExpr.Format().String())
		}
	}
}
