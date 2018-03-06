package treesql

import (
	"testing"
)

func TestCreateTable(t *testing.T) {
	tsr := runSimpleTestScript(t, []simpleTestStmt{
		// validate that there's a primary key
		{
			stmt:  "CREATETABLE foo (id int)",
			error: `validation error: tables should have exactly one column marked "primary key"; given 0`,
		},
		// validate that references exist
		{
			stmt:  "CREATETABLE bar (id int PRIMARYKEY, blog_post_id string REFERENCESTABLE blog_posts)",
			error: `validation error: no such table: blog_posts`,
		},
		// happy path:
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
	tsr.Close()
}
