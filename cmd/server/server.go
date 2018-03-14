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

func main() {
	fmt.Println("TreeSQL server")

	// get cmdline flags
	var port = flag.Int("port", 9000, "port to listen for connections on")
	var dataFile = flag.String("data-file", "treesql.data", "data file")
	flag.Parse()

	server := treesql.NewServer(*dataFile, *port)

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
