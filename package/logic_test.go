package treesql

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"net/http"

	"github.com/pkg/errors"
)

func TestCreateTable(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// TODO: how to reliably find a port?
	// maybe the answer is not to run a freaking server inside the test process
	port := 12345
	s := NewServer(dir+"/test.data", port)
	go func() {
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Fatal(errors.Wrap(err, "ListenAndServe"))
		}
	}()
	defer s.Close()

	url := fmt.Sprintf("ws://localhost:%d/ws", port)
	conn, err := NewClientConn(url)
	if err != nil {
		t.Fatal(err)
	}

	response, err := conn.Exec(`
		CREATETABLE blog_posts (
			id string PRIMARYKEY,
			title string
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	expectedResp := "CREATE TABLE"
	if response != expectedResp {
		t.Fatalf("expected %s; got %s", expectedResp, response)
	}
}
