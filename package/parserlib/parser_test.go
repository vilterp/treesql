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
		input  string
		output string
	}{
		{"MANYTABLENAME{SELECTION}", ""},
		{"MANY09notatable{SELECTION}", `no match for sequence item 1: no match for rule "table_name": no match found for regex [a-zA-Z_][a-zA-Z0-9_-]+`},
	}
	for caseIdx, testCase := range cases {
		err := Parse(TestTreeSQLGrammar, "select", testCase.input)
		if err == nil {
			if testCase.output != "" {
				t.Fatalf(`case %d: got no error; expected "%s"`, caseIdx, testCase.output)
			}
			continue
		}
		if err.Error() != testCase.output {
			t.Fatalf(`case %d: expected "%s"; got "%s"`, caseIdx, testCase.output, err.Error())
		}
	}
}
