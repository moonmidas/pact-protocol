package audit

import (
	"path/filepath"
	"testing"

	"pact/internal/protocol"
)

func TestAppendAndReadEvents(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "audit.jsonl")
	event := protocol.AuditEvent{
		ID:        "aud_1",
		Profile:   "denis",
		EventType: "request.approved",
	}

	if err := Append(path, event); err != nil {
		t.Fatalf("Append: %v", err)
	}

	events, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(events) != 1 || events[0].EventType != "request.approved" {
		t.Fatalf("unexpected events: %#v", events)
	}
}
