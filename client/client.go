package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
)

func main() {
	fmt.Println("TreeSQL client")

	// get cmdline flags
	var host = flag.String("host", "localhost", "host to connect to")
	var port = flag.Int("port", 6000, "port to connect to")
	flag.Parse()

	// connect to server
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		fmt.Printf("failed to connect to %s:%d\n", *host, *port)
		os.Exit(1)
	}

	for {
		fmt.Printf("%s:%d> ", *host, *port)
		input := readFromPrompt()
		conn.Write([]byte(input + "\n"))
		readResult(conn)
	}
}

func readResult(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("connection error:", err)
			os.Exit(0)
		}
		if message == "done\n" {
			return
		}
		fmt.Printf("< %s", message)
	}
}

func printResult(result string) {
	fmt.Print("< ", result)
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
