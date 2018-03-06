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
			Concat([]Doc{Text("foo"), Text("["), Newline, Nest(Text("bar"), 2), Newline, Text("]")}),
			`foo[
  bar
]`,
		},
	}

	for idx, testCase := range cases {
		actual := testCase.in.Render()
		if actual != testCase.out {
			t.Fatalf("case %d:\nEXPECTED\n\n%s\n\nGOT\n\n%s", idx, testCase.out, actual)
		}
	}
}
