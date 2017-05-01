package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	"encoding/json"

	"bytes"

	"github.com/chzyer/readline"
	"github.com/robertkrimen/isatty"
)

func main() {
	// get cmdline flags
	var host = flag.String("host", "localhost", "host to connect to")
	var port = flag.Int("port", 6000, "port to connect to")
	flag.Parse()
	isInputTty := isatty.Check(os.Stdin.Fd())

	if isInputTty {
		fmt.Println("TreeSQL client")
	}

	// connect to server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		fmt.Printf("failed to connect to %s:%d\n", *host, *port)
		os.Exit(1)
	}

	// initialize readline
	prompt := ""
	if isInputTty {
		prompt = fmt.Sprintf("%s:%d> ", *host, *port)
	}
	l, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     "/tmp/.treesql-history",
		InterruptPrompt: "^C",
		EOFPrompt:       "bye!",

		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	for {
		line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		conn.Write([]byte(line + "\n"))
		readResult(conn)
	}
}

func readResult(conn net.Conn) {
	reader := bufio.NewReader(conn)
	message, err := reader.ReadBytes('\n')
	if err != nil {
		fmt.Println("connection error:", err)
		os.Exit(0)
	}
	if string(message) == "done\n" {
		return
	}
	var dstBuffer bytes.Buffer
	json.Indent(&dstBuffer, message, "", "  ")
	dstBuffer.WriteTo(os.Stdout)
}

func readFromPrompt() string {
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("bye!")
		os.Exit(0)
	}
	return text
}
