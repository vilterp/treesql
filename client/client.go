package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

// TODO: read from flags
const Host = "localhost"
const Port = "6000"

func main() {
	fmt.Println("TreeSQL client")
	reader := bufio.NewReader(os.Stdin)
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", Host, Port))

	if err != nil {
		fmt.Printf("failed to connect to %s:%s\n", Host, Port)
		os.Exit(1)
	}

	for {
		fmt.Printf("%s:%s> ", Host, Port)
		text, _ := reader.ReadString('\n')
		conn.Write([]byte(text + "\n"))
	}
}
