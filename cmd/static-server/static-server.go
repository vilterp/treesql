package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"encoding/json"

	"github.com/vilterp/treesql/package"
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
	clientConn, connErr := treesql.NewClientConn(*mothershipUrl)
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
	channel := clientConn.LiveQuery(getFilesQuery(*appID))
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
