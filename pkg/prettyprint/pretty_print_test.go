package prettyprint

import "testing"

func TestPrettyPrint(t *testing.T) {
	cases := []struct {
		in  Doc
		out string
	}{
		{
			Seq([]Doc{Text("foo"), Text(" "), Text("bar")}),
			`foo bar`,
		},
		{
			Seq([]Doc{Text("foo"), Text("["), Newline, Indent(2, Text("bar")), Newline, Text("]")}),
			`foo[
  bar
]`,
		},
		{
			Seq([]Doc{
				Text("["), Newline,
				Indent(2, Join([]Doc{
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
