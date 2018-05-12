package treesql

import (
	"testing"

	"fmt"

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
			`map(scan(tables.blog_posts.id), (row1) => {
  id: row1.id
})`,
			"",
		},
		{
			`MANY blog_posts WHERE id = 'foo' { id }`,
			`map(filter(scan(tables.blog_posts.id), (row1) => strEq(row1.id, "foo")), (row1) => {
  id: row1.id
})`,
			"",
		},
		{
			`MANY blog_posts { id, comments: MANY comments { id, blog_post_id } }`,
			`map(scan(tables.blog_posts.id), (row1) => {
  comments: map(filter(scan(tables.comments.id), (row2) => strEq(row2.blog_post_id, row1.id)), (row2) => {
    blog_post_id: row2.blog_post_id,
    id: row2.id
  }),
  id: row1.id
})`,
			"",
		},
	}

	for idx, testCase := range cases {
		t.Run(fmt.Sprintf("case_%d", idx), func(t *testing.T) {
			parsedQuery, err := Parse(testCase.in)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}

			db := tsr.server.db
			boltTxn, err := db.boltDB.Begin(false)
			if err != nil {
				t.Fatal(err)
			}
			txn := &txn{
				boltTxn: boltTxn,
				db:      db,
			}

			_, typeScope := db.schema.toScope(txn)

			expr, err := tsr.server.db.schema.planSelect(parsedQuery.Select, typeScope)
			if util.AssertError(t, idx, testCase.err, err) {
				return
			}

			formatted := expr.Format().String()
			if formatted != testCase.out {
				t.Fatalf("EXPECTED:\n\n%s\n\nGOT:\n\n%s\n\n", testCase.out, formatted)
			}
		})
	}
}
