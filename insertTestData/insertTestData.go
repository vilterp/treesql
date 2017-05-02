package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
	treesql "github.com/vilterp/treesql/package"
)

func main() {
	var dataDir = flag.String("data-file", "treesql.data", "data file")
	flag.Parse()

	database, err := treesql.Open(*dataDir)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data directory: %s\n", *dataDir)

	insertTestData(database)
}

func insertTestData(db *treesql.Database) {
	log.Println("writing test data")

	err := db.BoltDB.Update(func(tx *bolt.Tx) error {
		// posts
		postsBucket := tx.Bucket([]byte("blog_posts"))
		postsTable := db.Schema.Tables["blog_posts"]
		for i := 0; i < 3; i++ {
			record := postsTable.NewRecord()
			record.SetString("id", fmt.Sprintf("%d", i))
			record.SetString("author_id", "0")
			record.SetString("title", "hello world")
			record.SetString("body", "phew making a database is hard work")
			bytes := record.ToBytes()
			err := postsBucket.Put([]byte(fmt.Sprintf("%d", i)), bytes)
			if err != nil {
				return err
			}
		}

		// comments
		commentsBucket := tx.Bucket([]byte("comments"))
		commentsTable := db.Schema.Tables["comments"]
		for i := 0; i < 3; i++ {
			record := commentsTable.NewRecord()
			record.SetString("id", fmt.Sprintf("%d", i))
			record.SetString("author_id", "0")
			record.SetString("post_id", "0")
			record.SetString("body", "fa la la la la comment")
			bytes := record.ToBytes()
			err := commentsBucket.Put([]byte(fmt.Sprintf("%d", i)), bytes)
			if err != nil {
				return err
			}
		}

		// authors
		authorsBucket := tx.Bucket([]byte("users"))
		authorsTable := db.Schema.Tables["users"]
		for i := 0; i < 1; i++ {
			record := authorsTable.NewRecord()
			record.SetString("id", fmt.Sprintf("%d", i))
			record.SetString("name", "Pete")
			bytes := record.ToBytes()
			err := authorsBucket.Put([]byte(fmt.Sprintf("%d", i)), bytes)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		log.Println("error writing test data:", err)
	}

	db.Close()

	log.Println("done writing test data")
}
