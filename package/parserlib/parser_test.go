package parserlib

import (
	"regexp"
	"testing"
)

// So, what does the parser actually return?
// at minimum, it just returns true/false...
// beyond that, it returns a representation of what
// path we took through the grammar railroad...
// it returns its state.

var TestTreeSQLGrammar = &Grammar{
	rules: map[string]Rule{
		"select": &Sequence{
			Items: []Rule{
				&Choice{
					Choices: []Rule{
						&Keyword{Value: "ONE"},
						&Keyword{Value: "MANY"},
					},
				},
				&Ref{Name: "table_name"},
				&Keyword{Value: "{"},
				&Ref{Name: "selection"},
				&Keyword{Value: "}"},
			},
		},
		"table_name": &Regex{Regex: regexp.MustCompile("[a-zA-Z_][a-zA-Z0-9_-]+")},
		"selection":  Intercalate(&Keyword{Value: "SELECTION"}, &Keyword{","}),
	},
}

func TestParse(t *testing.T) {
	cases := []struct {
		input string
		trace string
		error string
	}{
		{
			"MANYTABLENAME{SELECTION}",
			`<SEQ [<CHOICE 1 <KW "MANY" => 1:5> => 1:5>, <REF table_name <REGEX "TABLENAME" => 1:14> => 1:14>, <KW "{" => 1:15>, <REF selection <CHOICE 1 <KW "SELECTION" => 1:24> => 1:24> => 1:24>, <KW "}" => 1:25>] => 1:25>`,
			"",
		},
		{
			"MANY09notatable{SELECTION}",
			``,
			`no match for sequence item 1: no match for rule "table_name": no match found for regex [a-zA-Z_][a-zA-Z0-9_-]+`,
		},
	}
	for caseIdx, testCase := range cases {
		trace, err := Parse(TestTreeSQLGrammar, "select", testCase.input)
		if err == nil {
			if testCase.error != "" {
				t.Fatalf(`case %d: got no error; expected "%s"`, caseIdx, testCase.error)
			}
			if testCase.trace != trace.String() {
				t.Fatalf(`case %d: expected trace "%s"; got "%s"`, caseIdx, testCase.trace, trace.String())
			}
			continue
		}
		if err.Error() != testCase.error {
			t.Fatalf(`case %d: expected "%s"; got "%s"`, caseIdx, testCase.error, err.Error())
		}
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Parse(TestTreeSQLGrammar, "select", "MANYTABLENAME{SELECTION}")
		if err != nil {
			b.Fatal(err)
		}
	}
}
