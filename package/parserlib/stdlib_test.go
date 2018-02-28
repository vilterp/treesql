package parserlib

import (
	"testing"
)

func TestOpt(t *testing.T) {
	g, err := NewGrammar(map[string]Rule{
		"optbar": Opt(Keyword("bar")),
		"foo_optbar_baz": Sequence([]Rule{
			Keyword("foo"),
			Ref("optbar"),
			Keyword("baz"),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	allShouldSucceed(t, g, []succeedCase{
		{"optbar", "bar"},
		{"optbar", ""},
		{"foo_optbar_baz", "foobarbaz"},
		{"foo_optbar_baz", "foobaz"},
	})
}

func TestRegexes(t *testing.T) {
	g, err := NewGrammar(map[string]Rule{
		"int_lit":    SignedIntLit,
		"str_lit":    StringLit,
		"ident":      Ident,
		"whitespace": Whitespace,
	})
	if err != nil {
		t.Fatal(err)
	}

	allShouldSucceed(t, g, []succeedCase{
		{"int_lit", "0"},
		{"int_lit", "123"},
		{"int_lit", "-123"},
		{"str_lit", `"hello world"`},
		{"str_lit", `"he said \"hello world\" blerp blerp"`},
		{"ident", "some_name2"},
		{"ident", "SomeName"},
		{"whitespace", " "},
		{"whitespace", "  "},
		{"whitespace", "\t"},
		{"whitespace", "\t\n\t"},
	})
}

func TestWhitespaceSeq(t *testing.T) {
	g, err := NewGrammar(map[string]Rule{
		"whitespace_seq": WhitespaceSeq([]Rule{
			Keyword("a"),
			Keyword("b"),
			Keyword("c"),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	allShouldSucceed(t, g, []succeedCase{
		{"whitespace_seq", "a b c"},
		{"whitespace_seq", "a    b c"},
		{"whitespace_seq", "a    b\n\tc"},
	})
}

type succeedCase struct {
	rule  string
	input string
}

func allShouldSucceed(t *testing.T, g *Grammar, cases []succeedCase) {
	for caseIdx, testCase := range cases {
		if _, err := g.Parse(testCase.rule, testCase.input); err != nil {
			t.Errorf("case %d: rule=%s, input=%s, err=%v", caseIdx, testCase.rule, testCase.input, err)
		}
	}
}
