package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/google/uuid"
	treesql "github.com/vilterp/treesql/package"
)

func main() {
	// flags
	mothershipUrl := flag.String("mothership-url", "ws://treesql.com:9000/ws", "URL of mothership to connect to")
	appID := flag.String("app-id", "", "id of the app to serve assets for")
	dir := flag.String("dir", ".", "directory to push")
	flag.Parse()

	// connect to mothership
	clientConn, connErr := treesql.NewClientConn(*mothershipUrl)
	if connErr != nil {
		fmt.Println("failed to connect:", connErr)
		return
	}
	fmt.Println("connected to", *mothershipUrl, "for app", *appID)

	// insert new version
	newVersionID := uuid.New()
	fmt.Println("new version:", newVersionID)

	newVersionStmt := fmt.Sprintf("insert into versions values ('%s', '%s', '%v')", newVersionID, *appID, time.Now())
	newVersionChannel := clientConn.SendStatement(newVersionStmt) // TODO: sendStatementSync or something
	go func() {
		update := <-newVersionChannel.Updates
		fmt.Println("new version", update)
	}()

	filepath.Walk(*dir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			fmt.Println("inserting", path)
			newFileID := uuid.New()
			contents, readErr := ioutil.ReadFile(path)
			if readErr != nil {
				fmt.Println("couldn't read file", path, ":", readErr)
			}
			newFileStmt := fmt.Sprintf(
				"insert into files values ('%s', '%s', '%s', %s)",
				newFileID, path, newVersionID, strconv.Quote(string(contents)),
			)
			newFileChannel := clientConn.SendStatement(newFileStmt)
			go func(ch *treesql.ClientChannel) {
				update := <-ch.Updates
				fmt.Println("new file", update)
			}(newFileChannel)
		}
		return nil
	})
}
