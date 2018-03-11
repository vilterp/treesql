package lang

import "testing"

func TestParser(t *testing.T) {
	cases := []string{
		// var
		`blerp`,
		// member access
		`foo.bar`,
		// func call
		`foo(bar, baz)`,
		// obj lit
		`{bloop:2}`,
		`{bloop: 2}`,
		`{ bloop: 2 }`,
		`{ bloop: 2, gloop: "bloop" }`,
		// lambda
		`() => plus(foo, bar)`,
		`(foo, bar) => plus(foo, bar)`,
	}

	for idx, testCase := range cases {
		_, err := Parse(testCase)
		if err != nil {
			t.Errorf("case %d: err: %v", idx, err)
		}
		// TODO: format a trace tree back to its original
	}
}
