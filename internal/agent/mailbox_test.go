package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMailboxSendReceive(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)

	// Send a message
	msg := mb.Send(TeamMessage{
		From: "worker-1",
		To:   "lead",
		Type: MsgText,
		Text: "Task completed",
	})

	if msg.ID == "" {
		t.Error("Message ID should not be empty")
	}

	// Receive
	messages := mb.Receive("lead")
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
	if messages[0].Text != "Task completed" {
		t.Errorf("Expected 'Task completed', got %q", messages[0].Text)
	}

	// Second receive should be empty (already read)
	messages = mb.Receive("lead")
	if len(messages) != 0 {
		t.Errorf("Expected 0 unread messages, got %d", len(messages))
	}
}

func TestMailboxBroadcast(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)

	// Create inboxes
	mb.EnsureInbox("worker-1")
	mb.EnsureInbox("worker-2")
	mb.EnsureInbox("lead")

	// Broadcast from lead
	mb.Send(TeamMessage{
		From: "lead",
		To:   "*",
		Type: MsgText,
		Text: "All workers stop",
	})

	// Worker 1 should receive
	w1 := mb.Receive("worker-1")
	if len(w1) != 1 {
		t.Errorf("Worker-1 expected 1 message, got %d", len(w1))
	}

	// Worker 2 should receive
	w2 := mb.Receive("worker-2")
	if len(w2) != 1 {
		t.Errorf("Worker-2 expected 1 message, got %d", len(w2))
	}

	// Lead should NOT receive their own broadcast
	lead := mb.Receive("lead")
	if len(lead) != 0 {
		t.Errorf("Lead should not receive own broadcast, got %d", len(lead))
	}
}

func TestMailboxIdleCollapsing(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)

	// Send 3 idle notifications from same worker
	for i := 0; i < 3; i++ {
		mb.Send(TeamMessage{
			From: "worker-1",
			To:   "lead",
			Type: MsgIdleNotify,
			Text: "idle",
		})
	}

	// Should only have 1 (latest collapsed)
	messages := mb.Receive("lead")
	if len(messages) != 1 {
		t.Errorf("Expected 1 collapsed idle notification, got %d", len(messages))
	}
}

func TestMailboxPeek(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)

	mb.Send(TeamMessage{From: "a", To: "b", Type: MsgText, Text: "hello"})

	// Peek should return without marking read
	peeked := mb.Peek("b")
	if len(peeked) != 1 {
		t.Fatalf("Peek expected 1, got %d", len(peeked))
	}

	// Receive should still work (not marked read by peek)
	received := mb.Receive("b")
	if len(received) != 1 {
		t.Errorf("Receive expected 1 after peek, got %d", len(received))
	}
}

func TestMailboxAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	mb := NewMailbox(dir)

	mb.Send(TeamMessage{From: "a", To: "b", Type: MsgText, Text: "test"})

	// Check no .tmp files left behind
	entries, _ := os.ReadDir(filepath.Join(dir, "mailbox"))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("Temporary file left behind: %s", e.Name())
		}
	}
}
