package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/pzhin/go-sophia"
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
	env := newEnvironment()
	dbs := openDatabases(env, *dataDir)
	fmt.Printf("opened data directory: %s\n", *dataDir)

	// listen & handle connections
	listeningSock, _ := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	fmt.Printf("listening on port %d\n", *port)

	connectionID := 0
	for {
		conn, _ := listeningSock.Accept()
		go handleConnection(conn, connectionID, env, dbs)
		connectionID++
	}
}

func handleConnection(conn net.Conn, connID int, env *sophia.Environment, dbs *map[string]*sophia.Database) {
	fmt.Printf("connection id %d from %s\n", connID, conn.RemoteAddr())
	for {
		// will listen for message to process ending in newline (\n)
		message, err := bufio.NewReader(conn).ReadString('\n')

		if err != nil {
			fmt.Printf("conn id %d terminated: %v\n", connID, err)
			return
		}

		// output message received
		fmt.Print("Message Received:", string(message))
		// sample process for string received
		newmessage := strings.ToUpper(message)
		// send new string back to client
		conn.Write([]byte(newmessage + "\n"))
	}
}

func newEnvironment() *sophia.Environment {
	env, _ := sophia.NewEnvironment()
	return env
}

func openDatabases(env *sophia.Environment, dataDir string) *map[string]*sophia.Database {
	env.Set("sophia.path", dataDir)

	// define hardcoded schemas
	// (in future will load these from some other DB)
	blogPostsSchema := &sophia.Schema{}
	blogPostsSchema.AddKey("id", sophia.FieldTypeUInt32)
	blogPostsSchema.AddValue("title", sophia.FieldTypeString)
	blogPostsSchema.AddValue("author_id", sophia.FieldTypeUInt32)
	blogPostsSchema.AddValue("body", sophia.FieldTypeString) // too bad Sophia doesn't have that Toast

	commentsSchema := &sophia.Schema{}
	commentsSchema.AddKey("id", sophia.FieldTypeUInt32)
	commentsSchema.AddValue("post_id", sophia.FieldTypeUInt32)
	commentsSchema.AddValue("author_id", sophia.FieldTypeUInt32)
	commentsSchema.AddValue("body", sophia.FieldTypeString)

	authorsSchema := &sophia.Schema{}
	authorsSchema.AddKey("id", sophia.FieldTypeUInt32)
	authorsSchema.AddValue("name", sophia.FieldTypeString)

	// open dbs
	dbs := make(map[string]*sophia.Database)

	blogPostsDB, _ := env.NewDatabase(&sophia.DatabaseConfig{
		Name:   "blog_posts",
		Schema: blogPostsSchema,
	})
	dbs["blog_posts"] = blogPostsDB

	commentsDB, _ := env.NewDatabase(&sophia.DatabaseConfig{
		Name:   "comments",
		Schema: commentsSchema,
	})
	dbs["comments"] = commentsDB

	authorsDB, _ := env.NewDatabase(&sophia.DatabaseConfig{
		Name:   "authors",
		Schema: authorsSchema,
	})
	dbs["authors"] = authorsDB

	env.Open()
	return &dbs
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
