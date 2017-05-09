package main

import (
	"flag"
	"fmt"
	"log"

	"os"
	"os/signal"
	"syscall"

	"net/http"

	"github.com/gorilla/websocket"
	treesql "github.com/vilterp/treesql/package"
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
		resp.Write([]byte("Hello from TreeSQL. The action is at /ws"))
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
		conn.WriteMessage(websocket.TextMessage, []byte("hello on a websocket"))
		_, data, messageErr := conn.ReadMessage()
		if messageErr != nil {
			log.Println("websocket message error:", messageErr)
		}
		log.Println("WS message received:", string(data))
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
