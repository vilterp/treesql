package treesql

import (
	"fmt"
	"io"

	sophia "github.com/pzhin/go-sophia"
)

func ExecuteQuery(resultWriter io.Writer, dbs map[string]*sophia.Database, query *Select) {
	db, ok := dbs[query.From.Table]
	if !ok {
		errorMsg := fmt.Sprintf("nonexistent table: %s", query.From.Table)
		resultWriter.Write([]byte(errorMsg + "\n"))
		resultWriter.Write([]byte("done"))
		fmt.Println(errorMsg)
		return
	}
	doc := db.Document()
	cursor, _ := db.Cursor(doc)
	rowsRead := 0
	for {
		nextDoc := cursor.Next()
		if nextDoc == nil {
			resultWriter.Write([]byte("done\n"))
			break
		}
		rowsRead++
		bpId := nextDoc.GetInt("id")
		bpTitleSize := 0
		bpTitle := nextDoc.GetString("title", &bpTitleSize)
		bpBodySize := 0
		bpBody := nextDoc.GetString("body", &bpBodySize)
		resultWriter.Write([]byte(fmt.Sprintf("{id:%d,title:%s,body:%s}\n", bpId, bpTitle, bpBody)))
	}
	fmt.Println("wrote response")
}
