package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorilla/websocket"
	treesql "github.com/vilterp/treesql/package"
)

func main() {
	fmt.Println("TreeSQL server")

	// get cmdline flags
	var port = flag.Int("port", 9000, "port to listen for connections on")
	var dataFile = flag.String("data-file", "treesql.data", "data file")
	flag.Parse()

	// open storage layer
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

	// set up HTTP server for static files
	fileServer := http.FileServer(http.Dir("webui/build/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		log.Println("serving index.html")
		http.ServeFile(resp, req, "webui/build/index.html")
	})

	// set up web server
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *http.Request) bool { return true }, // TODO: security... only do this in dev mode (...)
	}
	http.HandleFunc("/ws", func(resp http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			log.Println(err)
			return
		}
		database.NewConnection(conn).HandleStatements()
	})
	log.Println("serving HTTP at", fmt.Sprintf("http://localhost:%d/", *port))
	listenErr := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if listenErr != nil {
		log.Fatal("error listening:", listenErr)
	}
}
