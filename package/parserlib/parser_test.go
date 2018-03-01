package parserlib

import (
	"testing"
)

// So, what does the parser actually return?
// at minimum, it just returns true/false...
// beyond that, it returns a representation of what
// path we took through the grammar railroad...
// it returns its state.

func TestParse(t *testing.T) {
	// TODO: DRY this up
	tsg, err := TestTreeSQLGrammar()
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		rule  string
		input string
		error string
	}{
		{
			"selection_field",
			"id",
			"",
		},
		{
			"selection_fields",
			"id, body",
			"",
		},
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
			"ONE blog_posts WHERE id = 1 { title }",
			"",
		},
		{
			"select",
			"MANY 09notatable {SELECTION}",
			`line 1, col 6: no match found for regex [a-zA-Z_][a-zA-Z0-9_-]*
MANY 09notatable {SELECTION}
     ^`,
		},
	}
	for caseIdx, testCase := range cases {
		_, err := tsg.Parse(testCase.rule, testCase.input)
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
	tsg, err := TestTreeSQLGrammar()
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := tsg.Parse("select", `MANY blog_posts {
	id,
	body,
	comments: MANY comments {
		id,
		body
	}
}`)
		if err != nil {
			b.Fatal(err)
		}
	}
}
