package parserlib

import (
	"testing"
)

func TestIntercalate(t *testing.T) {
	t.Skip("skipping until repetition is a thing")
	g, err := NewGrammar(map[string]Rule{
		"derps": Intercalate(
			&Keyword{Value: "derp"},
			&Keyword{Value: ","},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(g)
	if _, err := Parse(g, "derps", `derp, derp, derp`); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(g, "derps", `derp`); err != nil {
		t.Fatal(err)
	}
}

func TestOpt(t *testing.T) {
	g, err := NewGrammar(map[string]Rule{
		"optbar": Opt(&Keyword{Value: "bar"}),
		"foo_optbar_baz": &Sequence{
			Items: []Rule{
				&Keyword{Value: "foo"},
				&Ref{Name: "optbar"},
				&Keyword{Value: "baz"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(g)

	allShouldSucceed(t, g, []succeedCase{
		{"optbar", "bar"},
		{"optbar", ""},
		{"foo_optbar_baz", "foobarbaz"},
		{"foo_optbar_baz", "foobaz"},
	})
}

type succeedCase struct {
	rule  string
	input string
}

func allShouldSucceed(t *testing.T, g *Grammar, cases []succeedCase) {
	for caseIdx, testCase := range cases {
		if _, err := Parse(g, testCase.rule, testCase.input); err != nil {
			t.Errorf("case %d: expected success for rule %s; got %v", caseIdx, testCase.rule, err)
		}
	}
}
