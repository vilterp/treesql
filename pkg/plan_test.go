package treesql

import (
	"testing"

	"github.com/vilterp/treesql/pkg/util"
)

func TestPlan(t *testing.T) {
	tsr := runSimpleTestScript(t, []simpleTestStmt{
		// Create blog post schema.
		{
			stmt: `
				CREATETABLE blog_posts (
					id string PRIMARYKEY,
					title string
				)
			`,
			ack: "CREATE TABLE",
		},
		{
			stmt: `
				CREATETABLE comments (
					id string PRIMARYKEY,
					blog_post_id string REFERENCESTABLE blog_posts,
					body string
				)
			`,
			ack: "CREATE TABLE",
		},
	})

	cases := []struct {
		in  string
		out string
		err string
	}{
		{
			`MANY blog_posts { id }`,
			`map(tables.blog_posts.id.scan, (row) => {
  id: row.id
})`,
			"",
		},
		{
			`MANY blog_posts WHERE id = 'foo' { id }`,
			`map(filter(tables.blog_posts.id.scan, (row) => strEq(row.id, "foo")), (row) => {
  id: row.id
})`,
			"",
		},
	}

	for idx, testCase := range cases {
		parsedQuery, err := Parse(testCase.in)
		if err != nil {
			t.Errorf("parse error: %v", err)
			continue
		}

		expr, err := tsr.server.db.schema.planSelect(parsedQuery.Select)
		if util.AssertError(t, idx, testCase.err, err) {
			continue
		}

		formatted := expr.Format().String()
		if formatted != testCase.out {
			t.Errorf("expected:\n\n%s\n\ngot:\n\n%s\n", testCase.out, formatted)
		}
	}
}
