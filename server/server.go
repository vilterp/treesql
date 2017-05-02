package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"os"
	"os/signal"
	"syscall"

	treesql "github.com/vilterp/treesql/package"
)

func main() {
	fmt.Println("TreeSQL server")

	// get cmdline flags
	var port = flag.Int("port", 6000, "port to listen for connections on")
	var dataFile = flag.String("data-file", "treesql.data", "data file")
	flag.Parse()

	// open Sophia storage layer
	database, err := treesql.Open(*dataFile)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data file: %s\n", *dataFile)

	// graceful shutdown on Ctrl-C (hopefully this will stop the routine corruption??)
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctrlCChan
		database.Close()
		os.Exit(1) // is 1 the proper exit code for Ctrl-C?
	}()

	// listen for connections
	listeningSock, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if listenErr != nil {
		log.Fatalln("failed to listen for connections:", listenErr)
	}
	log.Printf("listening on port %d\n", *port)

	// accept & handle connections
	connectionID := 0
	for {
		conn, _ := listeningSock.Accept()
		connection := &treesql.Connection{
			ClientConn: conn,
			ID:         connectionID,
			Database:   database,
		}
		connectionID++
		go connection.Run()
	}
}
