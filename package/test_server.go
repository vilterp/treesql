package treesql

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func NewTestServer() (*Server, *ClientConn, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, nil, err
	}
	defer os.RemoveAll(dir)

	// TODO: how to reliably find a port?
	// maybe the answer is not to run a freaking server inside the test process
	port := 12345
	server := NewServer(dir+"/test.data", port)
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()

	url := fmt.Sprintf("ws://localhost:%d/ws", port)
	client, err := NewClientConn(url)
	if err != nil {
		return nil, nil, err
	}

	return server, client, nil
}

// simpleTestCase is a test case for a statement that
// is not a live query -- i.e. it just has one response.
type simpleTestCase struct {
	stmt  string
	query string

	ack           string
	error         string
	initialResult string
}

func runSimpleTestCases(t *testing.T, cases []simpleTestCase) {
	server, client, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	defer server.Close()

	for idx, testCase := range cases {
		if testCase.stmt != "" {
			result, err := client.Exec(testCase.stmt)
			if err != nil {
				if testCase.error == "" {
					t.Fatalf(`case %d: expected success; got error "%s"`, idx, err.Error())
				}
				if err.Error() != testCase.error {
					t.Fatalf(`case %d: expected error "%s"; got "%s"`, idx, testCase.error, err.Error())
				}
				continue
			}
			if err == nil && testCase.error != "" {
				t.Fatalf(`case %d: expected error "%s"; got success`, idx, testCase.error)
			}
			// TODO: maybe move this to a validation phase
			if testCase.ack == "" {
				t.Fatal("no ack specified for statement")
			}
			if result != testCase.ack {
				t.Fatalf(`case %d: expected ack "%s"; got "%s"`, idx, testCase.ack, result)
			}
		}
	}
}
