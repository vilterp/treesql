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
	treesql "gitlab.com/vilterp/treesql/package"
)

func main() {
	fmt.Println("TreeSQL server")

	// get cmdline flags
	var port = flag.Int("port", 6000, "port to listen for connections on")
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

	// set up HTTP server
	http.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		log.Println("serving index.html")
		http.ServeFile(resp, req, "index.html")
	})
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	http.HandleFunc("/ws", func(resp http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			log.Println(err)
			return
		}
		database.NewConnection(conn).Run()
	})
	log.Println("serving HTTP at", fmt.Sprintf("http://localhost:%d/", *port))
	listenErr := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	if listenErr != nil {
		log.Fatal("error listening:", listenErr)
	}

	// // listen for connections
	// listeningSock, listenErr := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	// if listenErr != nil {
	// 	log.Fatalln("failed to listen for connections:", listenErr)
	// }
	// log.Printf("listening on port %d\n", *port)

	// // accept & handle connections
	// connectionID := 0
	// for {
	// 	conn, _ := listeningSock.Accept()
	// 	mx, _ := yamux.Server(conn, nil)
	// 	connection := &treesql.Connection{
	// 		ClientConn:  mx,
	// 		ID:          connectionID,
	// 		Database:    database,
	// 		NextQueryId: 0,
	// 	}
	// 	connectionID++
	// 	go connection.Run()
	// }
}
