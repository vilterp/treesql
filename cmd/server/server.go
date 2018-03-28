package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vilterp/treesql/pkg"
)

var port = flag.Int("port", 9000, "port to listen on")
var host = flag.String("host", "0.0.0.0", "host to listen on")
var dataFile = flag.String("data-file", "treesql.data", "data file")

func main() {
	// get cmdline flags
	flag.Parse()

	fmt.Println("TreeSQL server")

	server := treesql.NewServer(*dataFile, *host, *port)

	// graceful shutdown on Ctrl-C
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctrlCChan
		if err := server.Close(); err != nil {
			log.Println("error closing:", err)
		}
		os.Exit(0)
	}()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("error listening:", err)
	}
}
