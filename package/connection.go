package treesql

import (
	"bufio"
	"fmt"
	"net"

	"github.com/davecgh/go-spew/spew"
)

type Connection struct {
	ClientConn net.Conn
	ID         int
	Database   *Database
}

func HandleConnection(conn *Connection) {
	fmt.Printf("connection id %d from %s\n", conn.ID, conn.ClientConn.RemoteAddr())
	for {
		// will listen for message to process ending in newline (\n)
		message, err := bufio.NewReader(conn.ClientConn).ReadString('\n')

		if err != nil {
			fmt.Printf("conn id %d terminated: %v\n", conn.ID, err)
			return
		}

		// parse what was sent to us
		statement, err := Parse(message)
		if err != nil {
			fmt.Println("parse error:", err)
			conn.ClientConn.Write([]byte(fmt.Sprintf("parse error: %s\n", err)))
			continue
		}

		// output message received
		fmt.Print("SQL statement received:", spew.Sdump(statement))

		// validate query
		queryErr := conn.Database.ValidateSelect(statement)
		if queryErr != nil {
			conn.ClientConn.Write([]byte(fmt.Sprintf("query error: %s\n", queryErr)))
			continue
		}

		// execute query
		ExecuteQuery(conn, statement)
	}
}
