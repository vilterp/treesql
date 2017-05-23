package main

import (
	"flag"
	"fmt"
	"os"

	"encoding/json"

	"github.com/chzyer/readline"
	"github.com/robertkrimen/isatty"
	treesql "github.com/vilterp/treesql/package"
)

func main() {
	// get cmdline flags
	var url = flag.String("url", "ws://localhost:9000/ws", "URL of TreeSQL server to connect to")
	flag.Parse()

	// connect to server
	client, connErr := treesql.NewClientConn(*url)
	if connErr != nil {
		fmt.Println("couldn't connect:", connErr)
		os.Exit(1)
		return
	}

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

		channel := client.SendStatement(line)
		go handleUpdates(channel)
	}
}

func handleUpdates(channel *treesql.ClientChannel) {
	for {
		update := <-channel.Updates
		indented, _ := json.MarshalIndent(update, "", "  ")
		fmt.Println("from channel", channel.StatementID, ":", string(indented))
	}
}
