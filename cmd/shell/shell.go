package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chzyer/readline"
	"github.com/robertkrimen/isatty"
	"github.com/vilterp/treesql/package"
)

func main() {
	// get cmdline flags
	var url = flag.String("url", "ws://localhost:9000/ws", "URL of TreeSQL server to connect to")
	flag.Parse()

	// connect to server
	client, connErr := treesql.NewClient(*url)
	if connErr != nil {
		fmt.Println("couldn't connect:", connErr)
		os.Exit(1)
		return
	}
	defer client.Close()

	// check if is TTY
	isInputTty := isatty.Check(os.Stdin.Fd())

	if isInputTty {
		fmt.Println("TreeSQL client")
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

		// TODO: factor these out into a commands struct or something
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

		runStatement(client, line)
	}
}

func runStatement(client *treesql.Client, stmt string) {
	channel := client.LiveQuery(stmt)
	firstUpdate := <-channel.Updates
	printMessage(channel, firstUpdate)
	go handleMessages(channel)
}

func handleMessages(channel *treesql.ClientChannel) {
	for {
		message := <-channel.Updates
		printMessage(channel, message)
	}
}

func printMessage(channel *treesql.ClientChannel, msg *treesql.MessageToClient) {
	fmt.Printf("chan %d: ", channel.StatementID)
	if msg.AckMessage != nil {
		fmt.Println("ack:", *msg.AckMessage)
		return
	}
	if msg.ErrorMessage != nil {
		fmt.Println("error:", *msg.ErrorMessage)
		return
	}
	if msg.InitialResultMessage != nil {
		printJSON("init", msg.InitialResultMessage.Data)
		return
	}
	if msg.RecordUpdateMessage != nil {
		printJSON("record_update", msg.RecordUpdateMessage)
		return
	}
	if msg.TableUpdateMessage != nil {
		printJSON("table_update", msg.TableUpdateMessage)
		return
	}
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
