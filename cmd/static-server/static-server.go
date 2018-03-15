package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/vilterp/treesql/pkg"
)

func main() {
	// cmdline flags
	mothershipUrl := flag.String("mothership-url", "ws://localhost:9000/ws", "URL of mothership to connect to")
	appID := flag.String("app-id", "", "id of the app to serve assets for")
	flag.Parse()

	if *appID == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// connect to server
	clientConn, connErr := treesql.NewClient(*mothershipUrl)
	if connErr != nil {
		fmt.Println("failed to connect:", connErr)
		return
	}

	// graceful shutdown on Ctrl-C
	ctrlCChan := make(chan os.Signal, 1)
	signal.Notify(ctrlCChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctrlCChan
		os.Exit(0)
	}()

	// open files LQ
	res, channel, err := clientConn.LiveQuery(getFilesQuery(*appID))
	log.Println("initial files:", res.Value)
	if err != nil {
		log.Fatal(err)
	}
	for {
		update := <-channel.Updates
		parsed, _ := json.MarshalIndent(update, "", "  ")
		fmt.Println("update:", string(parsed))
	}
}

func getFilesQuery(appID string) string {
	return fmt.Sprintf(`
		one apps where id = "%s" {
			id,
			versions: many versions {
				id,
				timestamp,
				files: many files {
					id,
					path,
					contents
				}
			}
		}
		live
	`, appID)
}
