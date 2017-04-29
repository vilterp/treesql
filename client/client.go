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

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s:%d> ", *host, *port)
		text, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("bye!")
			os.Exit(0)
		}
		conn.Write([]byte(text + "\n"))
	}
}
