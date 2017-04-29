package treesql

import (
	"fmt"
	"io"

	sophia "github.com/pzhin/go-sophia"
)

func ExecuteQuery(resultWriter io.Writer, dbs map[string]*sophia.Database, query *Select) {
	_, err := resultWriter.Write([]byte("hello world\n"))
	if err != nil {
		fmt.Println("error writing query results:", err)
	}
	// writer := bufio.NewWriter(resultWriter)
	// writer.WriteString("hello world\n")
	fmt.Println("wrote response")
}
