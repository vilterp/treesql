package parserlib_test_harness

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/vilterp/treesql/package/parserlib"
)

type completionsRequest struct {
	Input     string
	CursorPos int // TODO: line/col?
}

type completionsResponse struct {
	Trace *parserlib.TraceTree
	Err   string
}

// TODO: use some logging middleware
// which prints statuses, urls, and times

func NewServer(port string, gram *parserlib.Grammar, startRule string) {
	gramSerialized := gram.Serialize()

	// Serve UI static files.

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("/index.html")
		http.ServeFile(w, r, "package/parserlib_test_harness/build/index.html")
	})

	fileServer := http.FileServer(http.Dir("package/parserlib_test_harness/build/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// Serve grammar and completions.

	http.HandleFunc("/grammar", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if err := json.NewEncoder(w).Encode(&gramSerialized); err != nil {
			log.Println("err encoding json:", err)
		}
		end := time.Now()
		log.Println("/grammar responded in", end.Sub(start))
	})

	http.HandleFunc("/completions", func(w http.ResponseWriter, r *http.Request) {
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
		trace, err := gram.Parse(startRule, cr.Input)
		resp.Trace = trace
		if err != nil {
			resp.Err = err.Error()
			log.Println("/completions parse error: ", err.Error())
		}

		// Respond.
		if err := json.NewEncoder(w).Encode(&resp); err != nil {
			log.Println("err encoding json:", err)
			http.Error(w, err.Error(), 500)
		}

		end := time.Now()
		log.Println("/completions responded in", end.Sub(start))
	})

	// Start 'er up.
	addr := fmt.Sprintf(":%s", port)
	log.Printf("serving on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
