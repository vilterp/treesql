package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"log"

	"github.com/chzyer/readline"
	"github.com/robertkrimen/isatty"
	"github.com/vilterp/treesql/pkg"
)

var url = flag.String("url", "ws://localhost:9000/ws", "URL of TreeSQL server to connect to")

func main() {
	// get cmdline flags
	flag.Parse()

	// connect to server
	client, connErr := treesql.NewClient(*url)
	if connErr != nil {
		fmt.Println("couldn't connect:", connErr)
		os.Exit(1)
		return
	}
	defer client.Close()

	// Wait for server closing
	go waitForServerClose(client)

	// check if is TTY
	isInputTty := isatty.Check(os.Stdin.Fd())

	if isInputTty {
		fmt.Println("TreeSQL shell")
		fmt.Println("\\h for help")
	}

	// initialize readline
	prompt := ""
	if isInputTty {
		prompt = fmt.Sprintf("%s> ", *url)
	}
	l, err := readline.NewEx(&readline.Config{
		Prompt:            prompt,
		HistoryFile:       "/tmp/.treesql-history",
		InterruptPrompt:   "^C",
		EOFPrompt:         "bye!",
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		line, readlineErr := l.Readline()
		if readlineErr != nil {
			fmt.Println("bye!")
			os.Exit(0)
		}

		// TODO: factor these out into a commands dict or something
		if line == `\h` {
			fmt.Println(`\h	help`)
			fmt.Println(`\d	describe schema`)
			continue
		}
		if line == `\d` { // describe schema
			runStatement(client, TablesQuery)
			continue
		}

		if len(strings.Trim(line, "\t ")) == 0 {
			continue
		}

		if strings.HasSuffix(line, "live") {
			runLiveQuery(client, line)
		} else {
			runStatement(client, line)
		}
	}
}

func waitForServerClose(client *treesql.Client) {
	<-client.ServerClosed
	log.Println("server closed the connection")
	// TODO: just reset the connection
	os.Exit(0)
}

func runLiveQuery(client *treesql.Client, query string) {
	initialResult, channel, err := client.LiveQuery(query)
	if err != nil {
		fmt.Println("error:", err)
	}
	printJSON("init", initialResult.Value)
	go handleMessages(channel)
}

func runStatement(client *treesql.Client, stmt string) {
	channel := client.RunStatement(stmt)
	firstUpdate := <-channel.Updates
	printMessage(channel, firstUpdate)
	go handleMessages(channel)
}

func handleMessages(channel *treesql.ClientChannel) {
	for message := range channel.Updates {
		printMessage(channel, message)
	}
}

func printMessage(channel *treesql.ClientChannel, msg *treesql.BasicMessageToClient) {
	fmt.Printf("chan %d: ", channel.StatementID)
	if msg.AckMessage != nil {
		fmt.Println("ack", *msg.AckMessage)
		return
	}
	if msg.ErrorMessage != nil {
		fmt.Println("error", *msg.ErrorMessage)
		return
	}
	if msg.InitialResultMessage != nil {
		printJSON("init", msg.InitialResultMessage.Value)
		return
	}
	//if msg.RecordUpdateMessage != nil {
	//	printJSON("record_update", msg.RecordUpdateMessage)
	//	return
	//}
	//if msg.TableUpdateMessage != nil {
	//	printJSON("table_update", msg.TableUpdateMessage)
	//	return
	//}
}

func printJSON(tag string, thing interface{}) {
	indented, _ := json.MarshalIndent(thing, "", "  ")
	fmt.Printf("%s:\n%s\n", tag, indented)
}

const TablesQuery = `
	many __tables__ {
    name,
    primary_key
  }
`
