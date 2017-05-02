package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

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
		fmt.Println("\\h for help")
	}

	// initialize readline
	prompt := ""
	if isInputTty {
		prompt = fmt.Sprintf("%s:%d> ", *host, *port)
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
		// connect to server
		conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *host, *port))
		if err != nil {
			fmt.Printf("failed to connect to %s:%d\n", *host, *port)
			os.Exit(1)
		}
		fmt.Printf("connected to %s:%d\n", *host, *port)

		// once we're connected, repl it up
		replErr := repl(l, conn)

		// if we get disconnected, go around the loop again
		if replErr == io.EOF {
			fmt.Println("connection error; trying to reconnect...")
		}
	}
}

func repl(reader *readline.Instance, conn net.Conn) error {
	for {
		line, err := reader.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				os.Exit(0)
			} else {
				continue
			}
		} else if err == io.EOF {
			os.Exit(0)
		}
		// TODO: factor special commands out to somewhere
		line, commandErr := translateOrExecuteCommand(line)
		if commandErr != nil {
			fmt.Println(commandErr)
		} else if len(line) > 0 {
			conn.Write([]byte(line + "\n"))
			readErr := readResult(conn)
			if readErr != nil {
				return readErr
			}
		}
	}
}

func translateOrExecuteCommand(line string) (string, error) {
	if line == "\\h" {
		fmt.Println("Help:")
		fmt.Println("  \\d:               list tables")
		fmt.Println("  \\d <table name>:  describe <table name>")
		return "", nil
	} else if strings.HasPrefix(line, "\\d") {
		if line == "\\d" {
			return "many __tables__ { name, primary_key }", nil
		} else {
			segments := strings.Split(line, " ")
			if len(segments) == 2 {
				return fmt.Sprintf(
					"one __tables__ where name = \"%s\" { name, primary_key, columns: many __columns__ { name, references } }",
					segments[1],
				), nil
			} else {
				return "", errors.New("unknown command")
			}
		}
	} else {
		return line, nil
	}
}

func readResult(conn net.Conn) error {
	reader := bufio.NewReader(conn)
	message, err := reader.ReadBytes('\n')
	if err != nil {
		return err
	}
	var dstBuffer bytes.Buffer
	jsonErr := json.Indent(&dstBuffer, message, "", "  ")
	if jsonErr == nil {
		dstBuffer.WriteTo(os.Stdout)
	} else {
		fmt.Println(string(message))
	}
	return nil
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
