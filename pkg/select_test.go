package treesql

import "testing"

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
		{
			stmt: `CREATETABLE map (id string PRIMARYKEY)`,
			ack:  "CREATE TABLE",
		},
		{
			stmt: `INSERT INTO map VALUES ("1")`,
			ack:  "INSERT 1",
		},
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
		{
			query: `MANY map { id }`,
			initialResult: `[
  {
    "id": "1"
  }
]`,
		},
		// TODO: test validation errors

		// TODO: sort output so we don't have indeterminant map iteration flakiness.
		//		{
		//			query: `
		//				MANY blog_posts {
		//					id,
		//					title,
		//					comments: MANY comments {
		//						id,
		//						body
		//					}
		//				}
		//			`,
		//			initialResult: `[
		//  {
		//    "comments": [
		//      {
		//        "body": "hello yourself!",
		//        "id": "0"
		//      }
		//    ],
		//    "id": "0",
		//    "title": "hello world"
		//  },
		//  {
		//    "comments": [
		//      {
		//        "body": "sup",
		//        "id": "1"
		//      },
		//      {
		//        "body": "so creative",
		//        "id": "2"
		//      }
		//    ],
		//    "id": "1",
		//    "title": "hello again world"
		//  }
		//]`,
		//		},
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
