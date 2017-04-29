package main

import (
	"flag"
	"fmt"
	"net"
	"time"

	"log"

	"github.com/pzhin/go-sophia"
	treesql "github.com/vilterp/treesql/package"
)

const (
	KeyTemplate   = "key%v"
	ValueTemplate = "value%v"

	RecordsCount = 500000

	RecordsCountBench = 5000000
)

func main() {
	fmt.Println("TreeSQL server")

	// get cmdline flags
	var port = flag.Int("port", 6000, "port to listen for connections on")
	var dataDir = flag.String("data-dir", "data", "data directory")
	flag.Parse()

	// open Sophia storage layer
	database, err := treesql.Open(*dataDir)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data directory: %s\n", *dataDir)

	insertTestData(database)

	// listen & handle connections
	listeningSock, _ := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	log.Printf("listening on port %d\n", *port)

	connectionID := 0
	for {
		conn, _ := listeningSock.Accept()
		connection := &treesql.Connection{
			ClientConn: conn,
			ID:         connectionID,
			Database:   database,
		}
		connectionID++
		go treesql.HandleConnection(connection)
	}
}

// I never know whether I'm supposed to be passing by value or reference

func insertTestData(db *treesql.Database) {
	fmt.Println("writing test data")
	blogPosts := db.Dbs["blog_posts"]

	blogPost := blogPosts.Document()
	blogPost.Set("id", fmt.Sprintf("%d", 10000))
	blogPost.Set("title", "Hello world")
	blogPost.Set("body", "whew, making a db is hard work")
	// spew.Dump(blogPost)
	err := blogPosts.Set(blogPost)
	if err != nil {
		fmt.Println("error writing post:", err)
	}
	// for i := 0; i < 1; i++ {
	// 	blogPost := blogPosts.Document()
	// 	blogPost.SetInt("id", int64(i))
	// 	blogPost.SetString("title", "Hello world")
	// 	blogPost.SetString("body", "whew, making a db is hard work")
	// 	// spew.Dump(blogPost)
	// 	err := blogPosts.Set(blogPost)
	// 	if err != nil {
	// 		fmt.Println("error writing post:", err)
	// 	}
	// }
	fmt.Println("done writing test data")
}

func doConcurrentTx(env *sophia.Environment, db *sophia.Database) {
	fmt.Println("starting initial writes")
	for i := 0; i < RecordsCount; i++ {
		doc := db.Document()
		doc.Set("key", fmt.Sprintf(KeyTemplate, i))
		doc.Set("value", fmt.Sprintf(ValueTemplate, i))

		db.Set(doc)
		doc.Free()
	}
	fmt.Println("finished initial writes")

	tx1, _ := env.BeginTx()
	tx2, _ := env.BeginTx()

	go func() {
		fmt.Println("starting tx1 writes")
		for i := 0; i < RecordsCount; i++ {
			doc := db.Document()
			value := fmt.Sprintf(ValueTemplate, i+1)
			doc.Set("key", fmt.Sprintf(KeyTemplate, i))
			doc.Set("value", value)

			tx1.Set(doc)
			doc.Free()
		}
		tx1.Commit()

		fmt.Println("finished tx1 writes")
	}()

	go func() {
		fmt.Println("starting tx2 writes")
		for i := 0; i < RecordsCount; i++ {
			doc := db.Document()
			doc.Set("key", fmt.Sprintf(KeyTemplate, i))
			value := fmt.Sprintf(ValueTemplate, i+2)
			doc.Set("value", value)

			tx2.Set(doc)
			doc.Free()
		}
		tx2.Commit()

		fmt.Println("finished tx2 writes")
	}()

	fmt.Println("sleeping for 30s")
	time.Sleep(time.Duration(30) * time.Second)

	fmt.Println("reading")
	var size int
	for i := 0; i < RecordsCount; i++ {
		doc := db.Document()
		doc.Set("key", fmt.Sprintf(KeyTemplate, i))

		d, _ := db.Get(doc)
		value := d.GetString("value", &size)
		fmt.Printf("read %s\n", value)
		doc.Free()
		d.Free()
		d.Destroy()
	}
	fmt.Println("done reading")
}
