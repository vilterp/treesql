package treesql

import (
	"fmt"
	"math/rand"
	"testing"
)

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
			error: `executing insert: record already exists with primary key id="0"`,
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

func BenchmarkInsert(t *testing.B) {
	numAuthors := 5
	numPosts := 100
	commentsPerPost := 20

	for i := 0; i < t.N; i++ {
		server, client, err := NewTestServer(testServerArgs{})
		if err != nil {
			t.Fatal(err)
		}
		// Create schema.
		schemaStmts := []string{
			`CREATETABLE authors (
				id string primarykey,
				name string
			)`,
			`CREATETABLE blog_posts (
				id string primarykey,
				author_id string referencestable authors,
				title string
			)`,
			`CREATETABLE comments (
				id string primarykey,
				author_id string referencestable authors,
				post_id string referencestable blog_posts,
				body string
			)`,
		}

		for _, stmt := range schemaStmts {
			if _, err := client.Exec(stmt); err != nil {
				t.Fatal(err)
			}
		}

		t.StartTimer()

		// Insert authors.
		for i := 0; i < numAuthors; i++ {
			insertStmt := fmt.Sprintf(`INSERT INTO authors VALUES ("%d", "Author %d")`, i, i)
			if _, err := client.Exec(insertStmt); err != nil {
				t.Fatal(err)
			}
		}

		insertComments := func(client *Client, postID int) {
			// Insert comments on post.
			for commentID := 0; commentID < commentsPerPost; commentID++ {
				commentAuthorID := rand.Intn(numAuthors)
				insertStmt := fmt.Sprintf(
					`INSERT INTO comments VALUES ("%d-%d", "%d", "%d", "Bla bla bla bla bla")`,
					postID, commentID, commentAuthorID, postID,
				)
				if _, err := client.Exec(insertStmt); err != nil {
					t.Fatal(err)
				}
			}
		}

		// Insert posts.
		for postID := 1; postID < numPosts; postID++ {
			authorID := rand.Intn(numAuthors)
			insertStmt := fmt.Sprintf(
				`INSERT INTO blog_posts VALUES ("%d", "%d", "Bla bla bla bla bla")`, postID, authorID,
			)
			if _, err := client.Exec(insertStmt); err != nil {
				t.Fatal(err)
			}
			insertComments(client, postID)
		}

		t.StopTimer()

		server.close()
	}
}
