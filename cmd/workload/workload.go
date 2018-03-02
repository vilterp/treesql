package main

import (
	"flag"
	"fmt"
	"log"

	"math/rand"

	"github.com/vilterp/treesql/package"
)

var load = flag.Bool("load", false, "load schema")
var url = flag.String("url", "ws://localhost:9000/ws", "url of treesql server to connect to")
var numLiveQueries = flag.Int("numLiveQueries", 5, "number of live queries to open")
var numAuthors = flag.Int("numAuthors", 5, "number of authors to create")
var posts = flag.Int("numPosts", 10000000000, "number of posts to insert")
var commentsPerPost = flag.Int("numCommentsPerPost", 10, "number of comments per post")

var schemaStmts = []string{
	`createtable authors (
		id string primarykey,
		name string
	)`,
	`createtable blog_posts (
		id string primarykey,
		author_id string referencestable authors,
		title string
	)`,
	`createtable comments (
		id string primarykey,
		author_id string referencestable authors,
		post_id string referencestable blog_posts,
		body string
	)`,
}

func main() {
	flag.Parse()

	client, err := treesql.NewClient(*url)
	if err != nil {
		log.Fatal(err)
	}

	// Load schema.
	if *load {
		log.Println("loading schema")
		for _, stmt := range schemaStmts {
			log.Println(stmt)
			if _, err := client.Exec(stmt); err != nil {
				log.Fatal(err)
			}
		}
	}

	// Open live queries.
	log.Println("opening live queries")
	for i := 0; i < *numLiveQueries; i++ {
		_, channel, err := client.LiveQuery(`
			MANY blog_posts {
				id,
				title,
				author: ONE authors {
					id,
					name
				},
				comments: MANY comments {
					id,
					body,
					author: ONE authors {
						id,
						name
					}
				}
			} live
		`)
		if err != nil {
			log.Fatal(err)
		}
		go func() {
			for {
				<-channel.Updates
			}
		}()
	}

	// Insert authors.
	log.Println("inserting authors")
	for i := 0; i < *numAuthors; i++ {
		insertStmt := fmt.Sprintf(`INSERT INTO authors VALUES ("%d", "Author %d")`, i, i)
		if _, err := client.Exec(insertStmt); err != nil {
			log.Fatal(err)
		}
	}

	// Insert posts and comments.
	log.Println("inserting posts")
	for postID := 0; postID < *posts; postID++ {
		authorID := rand.Intn(*numAuthors)
		insertStmt := fmt.Sprintf(
			`INSERT INTO blog_posts VALUES ("%d", "%d", "Bla bla bla bla bla")`, postID, authorID,
		)
		postID++
		if _, err := client.Exec(insertStmt); err != nil {
			log.Fatal(err)
		}
		// Insert comments on post.
		for commentID := 0; commentID < *commentsPerPost; commentID++ {
			commentAuthorID := rand.Intn(*numAuthors)
			insertStmt := fmt.Sprintf(
				`INSERT INTO comments VALUES ("%d-%d", "%d", "%d", "Bla bla bla bla bla")`,
				postID, commentID, commentAuthorID, postID,
			)
			if _, err := client.Exec(insertStmt); err != nil {
				log.Fatal(err)
			}
		}
		if postID%500 == 0 {
			log.Println("post id:", postID)
		}
	}
}
