package treesql

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestSelect(t *testing.T) {
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
		//{
		//	stmt: `CREATE TABLE empty (id string PRIMARYKEY)`
		//}
		// Test select.
		{
			query: `MANY blog_posts { id, title }`,
			initialResult: `[
  {
    "id": "0",
    "title": "hello world"
  },
  {
    "id": "1",
    "title": "hello again world"
  }
]`,
		},
		// TODO: re-enable WHERE
		//		{
		//			query: "MANY blog_posts WHERE id = '1' { title }",
		//			initialResult: `[
		//  {
		//    "title": "hello again world"
		//  }
		//]`,
		//		},
		{
			query: "MANY blog_posts { id, title, comments: MANY comments { id, body } }",
			initialResult: `[
  {
    "comments": [
      {
        "body": "hello yourself!",
        "id": "0"
      }
    ],
    "id": "0",
    "title": "hello world"
  },
  {
    "comments": [
      {
        "body": "sup",
        "id": "1"
      },
      {
        "body": "so creative",
        "id": "2"
      }
    ],
    "id": "1",
    "title": "hello again world"
  }
]`,
		},
		// TODO: test validation errors

		// TODO: re-enable WHERE
		//		{
		//			query: `MANY blog_posts WHERE id = "0" { title }`,
		//			initialResult: `[
		//  {
		//    "title": "hello world"
		//  }
		//]`,
		//		},
	})
	tsr.Close()
}

func BenchmarkSelect(t *testing.B) {
	numAuthors := 5
	numPosts := 100
	commentsPerPost := 20

	server, client, err := NewTestServer(testServerArgs{})
	if err != nil {
		t.Fatal(err)
	}
	defer server.close()

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

	// Now for the actual benchmark: run queries.
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := client.Query(`MANY blog_posts { id, comments: MANY comments { id } }`)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.StopTimer()
}
