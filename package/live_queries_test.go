package treesql

import (
	"testing"
)

func TestLiveQueries(t *testing.T) {
	_, client, err := NewTestServer()
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	defer client.Close()

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

	lqChan := client.LiveQuery(`
		MANY blog_posts {
			id,
			comments: MANY comments {
				id
			}
		}
	`)

	// TODO: assert against actual message contents.

	// Verify table listener is hit.
	go func() {
		initialResult := <-lqChan.Updates // throw away initial result
		if initialResult.Type != InitialResultMessage {
			t.Fatalf("expected %v but got %v", InitialResultMessage, initialResult.Type)
		}

		msg := <-lqChan.Updates
		if msg.Type != TableUpdateMessage {
			t.Fatalf("expected %v but got %v", TableUpdateMessage, msg.Type)
		}
	}()
	client.Exec(`INSERT INTO blog_posts VALUES ("0", "hello world")`)

	// Verify record listener is hit.
	go func() {
		msg := <-lqChan.Updates
		if msg.Type != RecordUpdateMessage {
			t.Fatalf("expected %v but got %v", RecordUpdateMessage, msg.Type)
		}
	}()
	client.Exec(`UPDATE blog_posts SET title = "hello world!" WHERE id = "0"`)

	// Verify nested table listener is hit.
	go func() {
		msg := <-lqChan.Updates
		if msg.Type != TableUpdateMessage {
			t.Fatalf("expected %v but got %v", TableUpdateMessage, msg.Type)
		}
	}()
	client.Exec(`INSERT INTO comments VALUES ("0", "0", "nice post")`)

	// Verify nested record listener is hit.
	go func() {
		msg := <-lqChan.Updates
		if msg.Type != RecordUpdateMessage {
			t.Fatalf("expected %v but got %v", RecordUpdateMessage, msg.Type)
		}
	}()
	client.Exec(`UPDATE comments SET body = "nice post!" WHERE id = "0"`)
}
