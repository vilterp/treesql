package treesql

import "testing"

func TestLiveQueries(t *testing.T) {
	t.Skip("this is not gonna work until FP is hooked up")

	server, client, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	defer server.close()

	if _, err := client.Exec(`
		CREATETABLE blog_posts (
			id string PRIMARYKEY,
			title string
		)
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Exec(`
		CREATETABLE comments (
			id string PRIMARYKEY,
			blog_post_id string REFERENCESTABLE blog_posts,
			body string
		)
	`); err != nil {
		t.Fatal(err)
	}

	_, _, lqErr := client.LiveQuery(`
		MANY blog_posts {
			id,
			comments: MANY comments {
				id
			}
		} live
	`)
	if lqErr != nil {
		t.Fatal(lqErr)
	}

	// TODO: assert against actual message contents.

	done := make(chan bool)

	// Verify table listener is hit.
	//go func() {
	//	msg2 := <-lqChan.Updates
	//	t.Log("received table listener update")
	//	if msg2.Type != TableUpdateMessage {
	//		t.Fatalf("expected %v but got %v", TableUpdateMessage, msg2.Type)
	//	}
	//
	//	msg3 := <-lqChan.Updates
	//	t.Log("received record listener update")
	//	if msg3.Type != RecordUpdateMessage {
	//		t.Fatalf("expected %v but got %v", RecordUpdateMessage, msg3.Type)
	//	}
	//
	//	msg4 := <-lqChan.Updates
	//	t.Log("received nested table listener update")
	//	if msg4.Type != TableUpdateMessage {
	//		t.Fatalf("expected %v but got %v", TableUpdateMessage, msg4.Type)
	//	}
	//
	//	msg5 := <-lqChan.Updates
	//	t.Log("received nested record listener update")
	//	if msg5.Type != RecordUpdateMessage {
	//		t.Fatalf("expected %v but got %v", RecordUpdateMessage, msg5.Type)
	//	}
	//
	//	done <- true
	//}()

	if _, err := client.Exec(`INSERT INTO blog_posts VALUES ("0", "hello world")`); err != nil {
		t.Fatal(err)
	}

	// Verify record listener is hit.
	if _, err := client.Exec(`UPDATE blog_posts SET title = "hello world!" WHERE id = "0"`); err != nil {
		t.Fatal(err)
	}

	// Verify nested table listener is hit.
	if _, err := client.Exec(`INSERT INTO comments VALUES ("0", "0", "nice post")`); err != nil {
		t.Fatal(err)
	}

	// Verify nested record listener is hit.
	if _, err := client.Exec(`UPDATE comments SET body = "nice post!" WHERE id = "0"`); err != nil {
		t.Fatal(err)
	}

	<-done // Make sure we're done
}
