package treesql

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/vilterp/treesql/package/lang"
	"github.com/vilterp/treesql/package/util"
)

func TestLangExec(t *testing.T) {
	tsr := runSimpleTestScript(t, []simpleTestStmt{
		// TODO: maybe dedup this with SelectTest?
		{
			stmt: `
				CREATETABLE blog_posts (
					id String PRIMARYKEY,
					title String
				)
			`,
			ack: "CREATE TABLE",
		},
		{
			stmt: `
				CREATETABLE comments (
					id String PRIMARYKEY,
					blog_post_id String REFERENCESTABLE blog_posts,
					body String
				)
			`,
			ack: "CREATE TABLE",
		},
		// Insert data.
		{
			stmt: `INSERT INTO blog_posts VALUES ("0", "hello world")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO blog_posts VALUES ("1", "hello again world")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO comments VALUES ("0", "0", "hello yourself!")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO comments VALUES ("1", "1", "sup")`,
			ack:  "INSERT 1",
		},
		{
			stmt: `INSERT INTO comments VALUES ("2", "1", "so creative")`,
			ack:  "INSERT 1",
		},
	})
	defer tsr.Close()

	db := tsr.server.db

	testCases := []struct {
		in      lang.Expr
		typ     string
		outJSON string
	}{
		{
			lang.NewMemberAccess(lang.NewMemberAccess(lang.NewVar("blog_posts"), "id"), "scan"),
			`Iterator<{
  id: String,
  title: String
}>`,
			`[
					{"id": "0", "title": "hello world"},
					{"id": "1", "title": "hello again world"}
			]`,
		},
	}

	for idx, testCase := range testCases {
		// Construct transaction.
		boltTxn, err := db.BoltDB.Begin(false)
		if err != nil {
			t.Fatal(err)
		}
		txn := &Txn{
			boltTxn: boltTxn,
			db:      db,
		}

		// Construct scope.
		userRootScope := db.Schema.toScope(txn)

		// Get type; compare.
		typ, err := testCase.in.GetType(userRootScope)
		if err != nil {
			t.Errorf("case %d: %v", idx, err)
			continue
		}
		if typ.Format().Render() != testCase.typ {
			t.Errorf("case %d: expected %s; got %s", idx, testCase.typ, typ.Format().Render())
		}

		// Interpret the test expression.
		val, err := lang.Interpret(testCase.in, userRootScope)
		if err != nil {
			// TODO: test for error
			t.Errorf("case %d: %v", idx, err)
			continue
		}

		// Get the output as a string of JSON.
		buf := bytes.NewBufferString("")
		bufWriter := bufio.NewWriter(buf)
		if err := val.WriteAsJSON(bufWriter); err != nil {
			t.Errorf("case %d: %v", idx, err)
			continue
		}
		bufWriter.Flush()
		json := buf.String()

		// Compare expected and actual JSON.
		eq, err := util.AreEqualJSON(json, testCase.outJSON)
		if err != nil {
			t.Errorf(`case %d: %v`, idx, err)
			continue
		}
		if !eq {
			t.Errorf("case %d: EXPECTED\n\n%s\n\nGOT:\n\n%s\n", idx, testCase.outJSON, json)
		}
	}
}
