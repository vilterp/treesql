package treesql

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vilterp/treesql/pkg/util"
)

type testServer struct {
	db         *Database
	testServer *httptest.Server
}

func (ts *testServer) close() error {
	if err := ts.db.Close(); err != nil {
		return err
	}
	ts.testServer.Close()
	return nil
}

func NewTestServer() (*testServer, *Client, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)

	db, handler := newServerInternal(dir + "/test.data")
	httpServer := httptest.NewServer(handler)

	url := fmt.Sprintf("ws://%s/ws", httpServer.Listener.Addr().String())
	client, err := NewClient(url)
	if err != nil {
		return nil, nil, err
	}

	tsServer := &testServer{
		testServer: httpServer,
		db:         db,
	}

	return tsServer, client, nil
}

// define stmt => define error or ack
// define query => define error or initialResponse
type simpleTestStmt struct {
	stmt  string
	query string

	ack           string
	error         string
	initialResult string
}

type testServerRef struct {
	server *testServer
	client *Client
}

func (tsr *testServerRef) Close() {
	tsr.server.close()
	tsr.client.Close()
}

// runSimpleTestScript spins up a test server and runs statements on it,
// checking each result. It doesn't support live queries; only initial results
// are checked.
func runSimpleTestScript(t *testing.T, cases []simpleTestStmt) *testServerRef {
	server, client, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}

	for idx, testCase := range cases {
		// Run a statement.
		if testCase.stmt != "" {
			result, err := client.Exec(testCase.stmt)
			if util.AssertError(t, idx, testCase.error, err) {
				continue
			}
			if result != testCase.ack {
				t.Fatalf(`case %d: expected ack "%s"; got "%s"`, idx, testCase.ack, result)
			}
			continue
		}
		// Run a query.
		if testCase.query != "" {
			res, err := client.Query(testCase.query)
			if util.AssertError(t, idx, testCase.error, err) {
				continue
			}
			indented, _ := json.MarshalIndent(res.Value, "", "  ")
			if string(indented) != testCase.initialResult {
				t.Fatalf("expected:\n\n%s\n\ngot:\n\n%s", testCase.initialResult, indented)
			}
		}
	}

	return &testServerRef{
		server: server,
		client: client,
	}
}
