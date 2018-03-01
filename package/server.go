package treesql

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type Server struct {
	db         *Database
	httpServer *http.Server
}

func NewServer(dataFile string, port int) *Server {
	// open storage layer
	database, err := Open(dataFile)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data file: %s\n", dataFile)

	serveMux := http.NewServeMux()

	// set up HTTP server for static files
	fileServer := http.FileServer(http.Dir("webui/build/static"))
	serveMux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	serveMux.HandleFunc("/favicon-96x96.png", func(resp http.ResponseWriter, req *http.Request) {
		log.Println("serving favicon.-96x96.png")
		http.ServeFile(resp, req, "webui/build/favicon-96x96.png")
	})
	serveMux.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		log.Println("serving index.html")
		http.ServeFile(resp, req, "webui/build/index.html")
	})

	// set up web server
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *http.Request) bool { return true }, // TODO: security... only do this in dev mode (...)
	}
	serveMux.HandleFunc("/ws", func(resp http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			log.Println(err)
			return
		}
		database.NewConnection(conn).HandleStatements()
	})

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: serveMux}

	return &Server{
		db:         database,
		httpServer: httpServer,
	}
}

func (s *Server) ListenAndServe() error {
	log.Println("serving HTTP at", fmt.Sprintf("http://%s/", s.httpServer.Addr))
	return s.httpServer.ListenAndServe()
}

func (s *Server) Close() error {
	log.Println("closing storage layer...")
	if err := s.db.Close(); err != nil {
		return err
	}
	log.Println("closing http server...")
	if err := s.httpServer.Close(); err != nil {
		return err
	}
	log.Println("bye!")
	return nil
}
