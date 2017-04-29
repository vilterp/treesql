package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	treesql "github.com/vilterp/treesql/package"
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

	// insertTestData(database)

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
