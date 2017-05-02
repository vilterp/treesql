package main

import (
	"flag"
	"fmt"
	"log"

	treesql "github.com/vilterp/treesql/package"
)

func main() {
	var dataDir = flag.String("data-dir", "data", "data directory")
	flag.Parse()

	database, err := treesql.Open(*dataDir)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data directory: %s\n", *dataDir)

	insertTestData(database)
}

func insertTestData(db *treesql.Database) {
	fmt.Println("writing test data")

	blogPosts := db.Tables["blog_posts"]
	for i := 0; i < 3; i++ {
		blogPost := blogPosts.Document()
		blogPost.Set("id", fmt.Sprintf("derp%d", i))
		blogPost.Set("title", "Hello world")
		blogPost.Set("body", "whew, making a db is hard work")
		err := blogPosts.Set(blogPost)
		if err != nil {
			fmt.Println("error writing post:", err)
		}
	}

	comments := db.Tables["comments"]
	for i := 0; i < 3; i++ {
		comment := comments.Document()
		comment.Set("id", fmt.Sprintf("derp%d", i))
		comment.Set("post_id", fmt.Sprintf("derp%d", i))
		comment.Set("body", "fa la la comment")
		err := comments.Set(comment)
		if err != nil {
			fmt.Println("error writing post:", err)
		}
	}

	users := db.Tables["users"]
	user1 := users.Document()
	user1.Set("id", fmt.Sprintf("derp%d", 1))
	user1.Set("name", "Pete")
	err1 := users.Set(user1)
	if err1 != nil {
		fmt.Println("error writing post:", err1)
	}

	user2 := users.Document()
	user2.Set("id", fmt.Sprintf("derp%d", 2))
	user2.Set("name", "Steve")
	err2 := users.Set(user2)
	if err2 != nil {
		fmt.Println("error writing post:", err2)
	}

	fmt.Println("done writing test data")
}
