package treesql

import "testing"

func TestLangExec(t *testing.T) {
	tsr := runSimpleTestScript(t, []simpleTestStmt{
		// TODO: maybe dedup this with SelectTest?
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
		// Insert data.
		{
			stmt: `INSERT INTO blog_posts VALUES ("0", "hello world")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO blog_posts VALUES ("1", "hello again world")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO comments VALUES ("0", "0", "hello yourself!")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO comments VALUES ("1", "1", "sup")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO comments VALUES ("2", "1", "so creative")`,
			ack:  "INSERT 1",
		},
	})
	defer tsr.Close()

	db := tsr.server.db

	userRootScope := db.Schema.toScope()

	// TODO: get a table iterator
}
