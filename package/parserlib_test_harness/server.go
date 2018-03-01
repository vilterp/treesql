package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/vilterp/treesql/package/parserlib"
)

var port = flag.String("port", "9999", "port to serve on")

type completionsRequest struct {
	Input     string
	CursorPos int // TODO: line/col?
}

type completionsResponse struct {
	Trace *parserlib.TraceTree
	Err   *parserlib.ParseError
}

func main() {
	flag.Parse()

	tsg, err := parserlib.TestTreeSQLGrammar()
	if err != nil {
		log.Fatal("error loading grammar:", err)
	}
	tsgSerialized := tsg.Serialize()

	// TODO: parameterize this server so it can be started up with other grammars

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("serving index.html")
		http.ServeFile(w, r, "index.html")
	})
	http.HandleFunc("/grammar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		log.Println("serving grammar")
		if err := json.NewEncoder(w).Encode(&tsgSerialized); err != nil {
			log.Println("err encoding json:", err)
		}
	})
	http.HandleFunc("/completions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		if r.Method != "POST" {
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
		trace, err := tsg.Parse("select", cr.Input)
		log.Println("/completions", trace, err)
		resp.Trace = trace
		if err != nil {
			switch tErr := err.(type) {
			case *parserlib.ParseError:
				resp.Err = tErr
			default:
				http.Error(w, fmt.Sprintf("error parsing: %v", err), 500)
			}
		}

		// Respond.
		if err := json.NewEncoder(w).Encode(&resp); err != nil {
			log.Println("err encoding json:", err)
			http.Error(w, err.Error(), 500)
		}
	})

	addr := fmt.Sprintf(":%s", *port)
	log.Printf("serving on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
