package treesql

import "testing"

func TestInsert(t *testing.T) {
	tsr := runSimpleTestScript(t, []simpleTestStmt{
		{
			stmt: "CREATETABLE blog_posts (id string PRIMARYKEY, body string)",
			ack:  "CREATE TABLE",
		},
		{
			stmt: `INSERT INTO blog_posts VALUES ("0", "hello world")`,
			ack:  "INSERT 1",
		},
		// Verify that primary key uniqueness violations are checked.
		{
			stmt:  `INSERT INTO blog_posts VALUES ("0", "another hello world")`,
			error: "executing insert: record already exists with primary key id=0",
		},
		// Verify that number of columns is checked.
		{
			stmt:  `INSERT INTO blog_posts VALUES ("0")`,
			error: "validation error: table blog_posts has 2 columns, but insert statement provided 1",
		},
		{
			stmt:  `INSERT INTO blog_posts VALUES ("0", "bloop", "doop")`,
			error: "validation error: table blog_posts has 2 columns, but insert statement provided 3",
		},
	})
	tsr.Close()
}
