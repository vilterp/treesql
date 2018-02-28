package parserlib

import "testing"

var TreeSQLGrammar = Grammar{
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
	},
}

func TestFormat(t *testing.T) {
	actual := TreeSQLGrammar.rules["select"].String()
	expected := `["ONE" | "MANY", table_name, "{", selection, "}"]`
	if actual != expected {
		t.Fatalf("expected `%s`; got `%s`", expected, actual)
	}
}

func TestValidate(t *testing.T) {
	actual := TreeSQLGrammar.Validate().Error()
	expected := `in rule "select": in seq item 1: ref not found: "table_name"`
	if actual != expected {
		t.Fatalf("expected `%v`; got `%v`", expected, actual)
	}
}
