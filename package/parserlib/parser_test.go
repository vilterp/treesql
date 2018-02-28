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
		"select": WhitespaceSeq([]Rule{
			Choice([]Rule{
				Keyword("ONE"),
				Keyword("MANY"),
			}),
			Ref("table_name"),
			Ref("selection"),
		}),
		"table_name": Regex(regexp.MustCompile("[a-zA-Z_][a-zA-Z0-9_-]+")),
		"selection": Sequence([]Rule{
			Keyword("{"),
			Opt(Whitespace),
			Ref("selection_fields"),
			Opt(Whitespace),
			Keyword("}"),
		}),
		// TODO: intercalate combinator (??)
		"selection_fields": ListRule(
			"selection_field",
			"selection_fields",
			Sequence([]Rule{Keyword(","), Opt(Whitespace)}),
		),
		"selection_field": Sequence([]Rule{
			Ident,
			Opt(Sequence([]Rule{
				Keyword(":"),
				Opt(Whitespace),
				Ref("select"),
			})),
		}),
	},
}

func TestParse(t *testing.T) {
	cases := []struct {
		rule  string
		input string
		error string
	}{
		{
			"select",
			"MANY comments {id}",
			"",
		},
		{
			"select",
			"MANY comments {id,body}",
			"",
		},
		{
			"select",
			"MANY blog_posts {id, body, comments: MANY comments {id}}",
			"",
		},
		{
			"select",
			"MANY blog_posts {id, body, comments: MANY comments { id }}",
			"",
		},
		{
			"select",
			`MANY blog_posts {
	id,
	body,
	comments: MANY comments {
		id,
		body
	}
}`,
			"",
		},
		{
			"select",
			"MANY 09notatable {SELECTION}",
			`line 1, col 6: no match found for regex [a-zA-Z_][a-zA-Z0-9_-]+
MANY 09notatable {SELECTION}
     ^`,
		},
	}
	for caseIdx, testCase := range cases {
		_, err := Parse(TestTreeSQLGrammar, testCase.rule, testCase.input)
		// TODO: I love you traces; will get back to you when I do completion
		if err == nil {
			if testCase.error != "" {
				t.Errorf(`case %d: got no error; expected "%s"`, caseIdx, testCase.error)
			}
			continue
		}
		switch parseErr := err.(type) {
		case *ParseError:
			inContext := parseErr.ShowInContext()
			if inContext != testCase.error {
				t.Errorf(`case %d: expected err "%s"; got "%s"`, caseIdx, testCase.error, inContext)
			}
		default:
			if err.Error() != testCase.error {
				t.Errorf(`case %d: expected err "%s"; got "%s"`, caseIdx, testCase.error, err)
			}
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
