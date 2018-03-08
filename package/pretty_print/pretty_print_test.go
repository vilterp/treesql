package pretty_print

import "testing"

func TestPrettyPrint(t *testing.T) {
	cases := []struct {
		in  Doc
		out string
	}{
		{
			Concat([]Doc{Text("foo"), Text(" "), Text("bar")}),
			`foo bar`,
		},
		{
			Concat([]Doc{Text("foo"), Text("["), Newline, Nest(2, Text("bar")), Newline, Text("]")}),
			`foo[
  bar
]`,
		},
		{
			Concat([]Doc{
				Text("["), Newline,
				Nest(2, Join([]Doc{
					Text("foo: bar,"),
					Text("baz: bin,"),
				}, Newline)),
				Newline, Text("]"),
			}),
			`[
  foo: bar,
  baz: bin,
]`,
		},
	}

	for idx, testCase := range cases {
		actual := testCase.in.String()
		if actual != testCase.out {
			t.Fatalf("case %d:\nEXPECTED\n\n%s\n\nGOT\n\n%s", idx, testCase.out, actual)
		}
	}
}
