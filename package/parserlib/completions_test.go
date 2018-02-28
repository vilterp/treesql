package parserlib

import (
	"reflect"
	"sort"
	"testing"
)

func TestCompletions(t *testing.T) {
	g, err := NewGrammar(map[string]Rule{
		"a_or_b": Choice([]Rule{Keyword("A"), Keyword("B")}),
		"c_or_d": Choice([]Rule{Keyword("C"), Keyword("D")}),
		"ab_then_cd": Sequence([]Rule{
			Choice([]Rule{Keyword("A"), Keyword("B")}),
			Choice([]Rule{Keyword("C"), Keyword("D")}),
		}),
		"ab_then_cd_refs": Sequence([]Rule{
			Ref("a_or_b"),
			Ref("c_or_d"),
		}),
	})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		grammar     *Grammar
		rule        string
		input       string
		completions []string
	}{
		{
			g,
			"a_or_b",
			"",
			[]string{"A", "B"},
		},
		{
			g,
			"ab_then_cd",
			"",
			[]string{"A", "B"},
		},
		{
			g,
			"ab_then_cd",
			"A",
			[]string{"C", "D"},
		},
		{
			g,
			"ab_then_cd_refs",
			"",
			[]string{"A", "B"},
		},
		{
			g,
			"ab_then_cd_refs",
			"A",
			[]string{"C", "D"},
		},
		{
			TestTreeSQLGrammar,
			"selection",
			"",
			[]string{"{"},
		},
		//{
		//	TestTreeSQLGrammar,
		//	"selection",
		//	"{foo",
		//	[]string{",", "}"},
		//},
	}
	for caseIdx, testCase := range cases {
		completions, err := testCase.grammar.GetCompletions(testCase.rule, testCase.input)
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(completions)
		sort.Strings(testCase.completions)
		if !reflect.DeepEqual(completions, testCase.completions) {
			t.Errorf("case %d: expected %v; got %v", caseIdx, testCase.completions, completions)
		}
	}
}
