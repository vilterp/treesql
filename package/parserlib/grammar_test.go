package parserlib

import "testing"

var TreeSQLGrammarPartial = Grammar{
	rules: map[string]Rule{
		"select": Sequence([]Rule{
			Choice([]Rule{
				&keyword{value: "ONE"},
				&keyword{value: "MANY"},
			}),
			Ref("table_name"),
			Keyword("{"),
			Ref("selection"),
			Keyword("}"),
		}),
	},
}

func TestFormat(t *testing.T) {
	actual := TreeSQLGrammarPartial.rules["select"].String()
	expected := `["ONE" | "MANY", table_name, "{", selection, "}"]`
	if actual != expected {
		t.Fatalf("expected `%s`; got `%s`", expected, actual)
	}
}

func TestValidate(t *testing.T) {
	actual := TreeSQLGrammarPartial.Validate().Error()
	expected := `in rule "select": in seq item 1: ref not found: "table_name"`
	if actual != expected {
		t.Fatalf("expected `%v`; got `%v`", expected, actual)
	}
}
