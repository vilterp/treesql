package treesql

import (
	"testing"
)

func TestParser(t *testing.T) {
	testCases := []string{
		`CREATETABLE blog_posts (id STRING PRIMARYKEY, title STRING, author_id STRING REFERENCESTABLE blog_posts)`,

		`MANY blog_posts { id, body, comments: MANY comments { id, body } }`,
		`ONE blog_posts WHERE id = "5" { id, title }`,

		`UPDATE blog_posts SET title = "bloop" WHERE id = "5"`,

		`INSERT INTO blog_posts VALUES ("5", "bloop_doop")`,
	}

	for _, testCase := range testCases {
		statement, err := Parse(testCase)
		formatted := statement.Format()
		if err != nil {
			t.Fatal("expected it to parse; got error:", err)
		}
		if formatted != testCase {
			t.Fatalf(`parsed "%s" and it formatted back to "%s"`, testCase, formatted)
		}
	}
}
