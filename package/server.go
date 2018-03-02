package treesql

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	db         *Database
	httpServer *http.Server
}

func NewServer(dataFile string, port int) *Server {
	// open storage layer
	database, err := NewDatabase(dataFile)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data file: %s\n", dataFile)

	serveMux := http.NewServeMux()

	// Serve static files for web console.
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

	// Serve metrics.
	serveMux.Handle(
		"/metrics",
		promhttp.HandlerFor(database.Metrics.registry, promhttp.HandlerOpts{}),
	)

	// Serve WebSocket endpoint for DB traffic.
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
		database.AddConnection(conn)
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
