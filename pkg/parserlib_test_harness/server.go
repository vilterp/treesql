package parserlib_test_harness

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vilterp/treesql/pkg/parserlib"
)

type completionsRequest struct {
	Input     string
	CursorPos int // TODO: line/col?
}

type completionsResponse struct {
	Trace       *parserlib.TraceTree
	PSITree     parserlib.PSINode
	Completions []string
	Err         string
}

// TODO: use some logging middleware
// which prints statuses, urls, and times

type server struct {
	language          parserlib.Language
	serializedGrammar *parserlib.SerializedGrammar
	startRule         string

	mux *http.ServeMux
}

func NewServer(l parserlib.Language, startRule string) *server {
	mux := http.NewServeMux()

	// Serve UI static files.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("/index.html")
		http.ServeFile(w, r, "pkg/parserlib_test_harness/build/index.html")
	})

	fileServer := http.FileServer(http.Dir("pkg/parserlib_test_harness/build/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	server := &server{
		language:          l,
		startRule:         startRule,
		serializedGrammar: l.Grammar.Serialize(),
		mux:               mux,
	}

	// Serve grammar and completions.
	http.HandleFunc("/grammar", server.handleGrammar)
	http.HandleFunc("/completions", server.handleCompletions)

	return server
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *server) handleGrammar(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(s.serializedGrammar); err != nil {
		log.Println("err encoding json:", err)
	}
	end := time.Now()
	log.Println("/grammar responded in", end.Sub(start))
}

func (s *server) handleCompletions(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if r.Method != "POST" {
		log.Println("/completions: expecting GET")
		http.Error(w, "expecting GET", 400)
		return
	}
	// Decode request.
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var cr completionsRequest
	err := decoder.Decode(&cr)
	if err != nil {
		log.Printf("/completions error: %v", err)
		http.Error(w, fmt.Sprintf("error parsing request body: %v", err), 400)
		return
	}

	var resp completionsResponse

	// Parse it.
	trace, err := s.language.Grammar.Parse(s.startRule, cr.Input, cr.CursorPos)
	resp.Trace = trace
	if err != nil {
		resp.Err = err.Error()
		log.Println("/completions parse error: ", err.Error())
	}
	if trace != nil {
		// Get PSI tree.
		resp.PSITree = s.language.ParseTreeToPSI(trace)
		// Get completions.
		completions, err := trace.GetCompletions()
		if err != nil {
			resp.Err = err.Error()
			log.Println("/completions completions error: ", err.Error())
		}
		resp.Completions = completions
	}

	// Respond.
	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		log.Println("err encoding json:", err)
		http.Error(w, err.Error(), 500)
	}

	end := time.Now()
	log.Println("/completions responded in", end.Sub(start))
}
