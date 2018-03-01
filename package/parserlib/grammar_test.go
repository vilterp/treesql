package parserlib

import "testing"

var partialTreeSQLGrammarRules = map[string]Rule{
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
}

func TestFormat(t *testing.T) {
	actual := partialTreeSQLGrammarRules["select"].String()
	expected := `["ONE" | "MANY", table_name, "{", selection, "}"]`
	if actual != expected {
		t.Fatalf("expected `%s`; got `%s`", expected, actual)
	}
}

func TestValidate(t *testing.T) {
	_, actual := NewGrammar(partialTreeSQLGrammarRules)
	expected := `in rule "select": in seq item 1: ref not found: "table_name"`
	if actual.Error() != expected {
		t.Fatalf("expected `%v`; got `%v`", expected, actual)
	}
}

func TestRuleIDs(t *testing.T) {
	g, err := TestTreeSQLGrammar()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.ruleForID) == 0 || len(g.idForRule) == 0 {
		t.Fatal("rule maps seem to be empty")
	}
}
