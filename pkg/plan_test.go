package treesql

import (
	"fmt"
	"testing"

	"github.com/vilterp/treesql/pkg/lang"
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
		// TODO: put update listeners back in
		{
			`MANY blog_posts { id }`,
			`do {
  selection = (row1) => do {
    innerSelection = (row1) => {
      id: row1.id
    }
    addUpdateListener(tables.blog_posts.id, row1.id, innerSelection)
    innerSelection(row1)
  }
  addInsertListener(tables.blog_posts.id, selection)
  map(scan(tables.blog_posts.id), selection)
}`,
			"",
		},
		//		{
		//			`MANY blog_posts WHERE id = 'foo' { id }`,
		//			`do {
		//  addInsertListener(tables.blog_posts.id, "foo")
		//  map(filter(scan(tables.blog_posts.id), (row1) => strEq(row1.id, "foo")), (row1) => do {
		//    addUpdateListener(tables.blog_posts.id, row1.id)
		//    {
		//      id: row1.id
		//    }
		//  })
		//}`,
		//			"",
		//		},
		{
			`MANY blog_posts { id, comments: MANY comments { id, blog_post_id } }`,
			`do {
  selection = (row1) => do {
    innerSelection = (row1) => {
      comments: do {
        subIndex = get(tables.comments.blog_post_id, row1.id)
        selection = (row2Key) => do {
          row2 = get(tables.comments.id, row2Key)
          innerSelection = (row2) => {
            blog_post_id: row2.blog_post_id,
            id: row2.id
          }
          addUpdateListener(tables.comments.id, row2Key, innerSelection)
          innerSelection(row2)
        }
        addInsertListener(subIndex, selection)
        map(scan(subIndex), selection)
      },
      id: row1.id
    }
    addUpdateListener(tables.blog_posts.id, row1.id, innerSelection)
    innerSelection(row1)
  }
  addInsertListener(tables.blog_posts.id, selection)
  map(scan(tables.blog_posts.id), selection)
}`,
			"",
		},
		// TODO: many-to-one joins
		//{
		//	`MANY comments { id, blog_post: ONE blog_posts { id } }`,
		//	``,
		//	"",
		//},
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

			indexMap := db.schema.toIndexMap(txn)

			expr, _, err := tsr.server.db.schema.planSelect(parsedQuery.Select, lang.BuiltinsScope.GetTypeScope(), indexMap)
			if util.AssertError(t, idx, testCase.err, err) {
				return
			}

			formatted := expr.Format().String()
			if formatted != testCase.out {
				t.Fatalf(
					"QUERY:\n\n%s\n\nEXPECTED:\n\n%s\n\nGOT:\n\n%s\n\n",
					testCase.in, testCase.out, formatted,
				)
			}
		})
	}
}
