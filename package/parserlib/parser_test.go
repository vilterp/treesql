package parserlib

import "testing"

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
		"table_name": &Keyword{Value: "TABLENAME"},
		"selection":  Intercalate(&Keyword{Value: "SELECTION"}, &Keyword{","}),
	},
}

func TestParse(t *testing.T) {
	cases := []struct {
		input  string
		output string
	}{
		{"MANYTABLENAME{SELECTION}", ""},
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
