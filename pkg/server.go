package treesql

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	db         *Database
	httpServer *http.Server
}

func NewServer(dataFile string, host string, port int) *Server {
	database, handler := newServerInternal(dataFile)

	httpServer := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port), Handler: handler}

	return &Server{
		db:         database,
		httpServer: httpServer,
	}
}

func newServerInternal(dataFile string) (*Database, http.Handler) {
	// open database
	database, err := NewDatabase(dataFile)
	if err != nil {
		log.Fatalln("failed to open database:", err)
	}
	log.Printf("opened data file: %s\n", dataFile)

	// set up HTTP server
	mux := http.NewServeMux()

	// Serve static files for web console.
	fileServer := http.FileServer(http.Dir("webui/build/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))
	mux.HandleFunc("/favicon-96x96.png", func(resp http.ResponseWriter, req *http.Request) {
		log.Println("serving favicon.-96x96.png")
		http.ServeFile(resp, req, "webui/build/favicon-96x96.png")
	})
	mux.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		log.Println("serving index.html")
		http.ServeFile(resp, req, "webui/build/index.html")
	})

	// Serve metrics.
	mux.Handle(
		"/metrics",
		promhttp.HandlerFor(database.metrics.registry, promhttp.HandlerOpts{}),
	)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Serve WebSocket endpoint for DB traffic.
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(_ *http.Request) bool { return true }, // TODO: security... only do this in dev mode (...)
	}
	mux.HandleFunc("/ws", func(resp http.ResponseWriter, req *http.Request) {
		conn, err := upgrader.Upgrade(resp, req, nil)
		if err != nil {
			log.Println(err)
			return
		}
		database.addConnection(conn)
	})

	return database, mux
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
