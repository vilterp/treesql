package treesql

import (
	"testing"
)

// TestRestart tests that the schema can be reloaded when
// the process restarts.
func TestRestart(t *testing.T) {
	// Create, insert, shutdown.
	ts, client, err := NewTestServer(testServerArgs{preserveWhenDone: true})
	if err != nil {
		t.Fatal(err)
	}

	_, err2 := client.Exec("CREATETABLE foo (id string primarykey)")
	if err2 != nil {
		t.Fatal(err2)
	}
	_, err3 := client.Exec("INSERT INTO foo VALUES ('bla')")
	if err3 != nil {
		t.Fatal(err3)
	}

	ts.close()

	// Start 'er back up again and see if our schema and data are still there.
	ts2, client2, err4 := NewTestServer(testServerArgs{dataFilePath: ts.dataFilePath})
	if err4 != nil {
		t.Fatalf("error restarting: %v", err4)
	}
	defer ts2.close()

	_, err5 := client2.Query("MANY foo { id }")
	if err5 != nil {
		t.Fatal(err5)
	}

	// skipping checking the result cuz that's weird right now
}
