package treesql

import "testing"

func TestUpdate(t *testing.T) {
	ts := runSimpleTestScript(t, []simpleTestStmt{
		{
			stmt: "CREATETABLE blog_posts (id string PRIMARYKEY, body string)",
			ack:  "CREATE TABLE",
		},
		{
			stmt: `INSERT INTO blog_posts VALUES ("0", "hello world")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `UPDATE blog_posts SET body = "goodbye world" WHERE id = "0"`,
			ack:  "UPDATE 1",
		},
		{
			query: "MANY blog_posts { id, body }",
			initialResult: `[
  {
    "body": "goodbye world",
    "id": "0"
  }
]`,
		},
	})
	ts.Close()

	// TODO: run a query after the update to test that indices got updated
}
