package requests

import (
	"os"
	"path/filepath"
	"testing"

	"pact/internal/pairing"
	"pact/internal/protocol"
	"pact/internal/store"
)

func setupPairedProfiles(t *testing.T) (store.Paths, string) {
	t.Helper()
	root := t.TempDir()
	paths := store.NewPaths(root)
	if _, err := pairing.InitProfile(paths, "denis"); err != nil {
		t.Fatalf("init denis: %v", err)
	}
	if _, err := pairing.InitProfile(paths, "esteban"); err != nil {
		t.Fatalf("init esteban: %v", err)
	}
	invitePath, _, err := pairing.CreateInvite(paths, "denis")
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if _, err := pairing.AcceptInvite(paths, "esteban", "denis", invitePath); err != nil {
		t.Fatalf("accept invite: %v", err)
	}
	return paths, root
}

func TestShareFileCreatesDeliveredRequest(t *testing.T) {
	paths, root := setupPairedProfiles(t)
	filePath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(filePath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	record, err := ShareFile(paths, "esteban", "denis", filePath)
	if err != nil {
		t.Fatalf("ShareFile: %v", err)
	}
	if record.State != protocol.StateCreated {
		t.Fatalf("sender record should be created, got %q", record.State)
	}

	recipientRequests, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile("denis"))
	if err != nil {
		t.Fatalf("read recipient requests: %v", err)
	}
	if len(recipientRequests) != 1 || recipientRequests[0].State != protocol.StateDelivered {
		t.Fatalf("unexpected recipient requests: %#v", recipientRequests)
	}

	messages, err := ListInbox(paths, "denis")
	if err != nil {
		t.Fatalf("ListInbox: %v", err)
	}
	if len(messages) != 1 || messages[0].Type != protocol.MessageGrantRequest {
		t.Fatalf("unexpected inbox messages: %#v", messages)
	}
}

func TestShareFileRejectsSecretLikePath(t *testing.T) {
	paths, root := setupPairedProfiles(t)
	filePath := filepath.Join(root, ".env")
	if err := os.WriteFile(filePath, []byte("TOKEN=abc\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := ShareFile(paths, "esteban", "denis", filePath); err == nil {
		t.Fatal("expected secret-like file to be rejected")
	}
}

func TestApproveCopiesArtifactAndCreatesGrant(t *testing.T) {
	paths, root := setupPairedProfiles(t)
	filePath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(filePath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	record, err := ShareFile(paths, "esteban", "denis", filePath)
	if err != nil {
		t.Fatalf("ShareFile: %v", err)
	}

	updated, err := Approve(paths, "denis", record.ID)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if updated.State != protocol.StateApproved {
		t.Fatalf("expected approved, got %q", updated.State)
	}

	received := filepath.Join(paths.ReceivedArtifactsDir("denis"), updated.Artifact.ID, "proposal.md")
	if _, err := os.Stat(received); err != nil {
		t.Fatalf("expected received artifact: %v", err)
	}

	grants, err := store.ReadJSONArray[protocol.GrantRecord](paths.GrantsFile("denis"))
	if err != nil {
		t.Fatalf("read grants: %v", err)
	}
	if len(grants) != 1 || grants[0].RequestID != record.ID {
		t.Fatalf("unexpected grants: %#v", grants)
	}
	senderRequests, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile("esteban"))
	if err != nil {
		t.Fatalf("read sender requests: %v", err)
	}
	if len(senderRequests) != 1 || senderRequests[0].State != protocol.StateApproved {
		t.Fatalf("expected sender request approved, got %#v", senderRequests)
	}
}

func TestRejectDoesNotCopyArtifact(t *testing.T) {
	paths, root := setupPairedProfiles(t)
	filePath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(filePath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	record, err := ShareFile(paths, "esteban", "denis", filePath)
	if err != nil {
		t.Fatalf("ShareFile: %v", err)
	}

	updated, err := Reject(paths, "denis", record.ID)
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if updated.State != protocol.StateDenied {
		t.Fatalf("expected denied, got %q", updated.State)
	}

	received := filepath.Join(paths.ReceivedArtifactsDir("denis"), updated.Artifact.ID, "proposal.md")
	if _, err := os.Stat(received); !os.IsNotExist(err) {
		t.Fatalf("expected no received artifact, stat err=%v", err)
	}
}

func TestCounterRecordsMessage(t *testing.T) {
	paths, root := setupPairedProfiles(t)
	filePath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(filePath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	record, err := ShareFile(paths, "esteban", "denis", filePath)
	if err != nil {
		t.Fatalf("ShareFile: %v", err)
	}

	updated, err := Counter(paths, "denis", record.ID, "Send a summary instead.")
	if err != nil {
		t.Fatalf("Counter: %v", err)
	}
	if updated.State != protocol.StateCountered {
		t.Fatalf("expected countered, got %q", updated.State)
	}
	if updated.Decision == nil || updated.Decision.Message != "Send a summary instead." {
		t.Fatalf("unexpected decision: %#v", updated.Decision)
	}
}

func TestExportAndImportSharePayload(t *testing.T) {
	senderRoot := t.TempDir()
	senderPaths := store.NewPaths(senderRoot)
	if _, err := pairing.InitProfile(senderPaths, "esteban"); err != nil {
		t.Fatalf("init sender: %v", err)
	}
	filePath := filepath.Join(senderRoot, "proposal.md")
	if err := os.WriteFile(filePath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	payload, err := ExportSharePayload(senderPaths, "esteban", "denis", filePath)
	if err != nil {
		t.Fatalf("ExportSharePayload: %v", err)
	}

	receiverRoot := t.TempDir()
	receiverPaths := store.NewPaths(receiverRoot)
	if _, err := pairing.InitProfile(receiverPaths, "denis"); err != nil {
		t.Fatalf("init receiver: %v", err)
	}
	payloadPath := filepath.Join(receiverRoot, "payload.json")
	if err := store.WriteJSON(payloadPath, payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	record, err := ImportSharePayload(receiverPaths, "denis", payloadPath)
	if err != nil {
		t.Fatalf("ImportSharePayload: %v", err)
	}
	if record.State != protocol.StateDelivered {
		t.Fatalf("expected delivered, got %s", record.State)
	}
	if _, err := Approve(receiverPaths, "denis", record.ID); err != nil {
		t.Fatalf("approve imported request: %v", err)
	}
	received := filepath.Join(receiverPaths.ReceivedArtifactsDir("denis"), record.Artifact.ID, "proposal.md")
	if _, err := os.Stat(received); err != nil {
		t.Fatalf("expected received artifact: %v", err)
	}
}

func TestApproveImportedPayloadWhenSenderProfileExistsWithoutSenderRequest(t *testing.T) {
	root := t.TempDir()
	paths := store.NewPaths(root)
	if _, err := pairing.InitProfile(paths, "esteban"); err != nil {
		t.Fatalf("init sender: %v", err)
	}
	if _, err := pairing.InitProfile(paths, "denis"); err != nil {
		t.Fatalf("init receiver: %v", err)
	}

	filePath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(filePath, []byte("# Proposal\n\nRelay import.\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	payload, err := ExportSharePayload(paths, "esteban", "denis", filePath)
	if err != nil {
		t.Fatalf("ExportSharePayload: %v", err)
	}
	payloadPath := filepath.Join(root, "payload.json")
	if err := store.WriteJSON(payloadPath, payload); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	record, err := ImportSharePayload(paths, "denis", payloadPath)
	if err != nil {
		t.Fatalf("ImportSharePayload: %v", err)
	}

	updated, err := Approve(paths, "denis", record.ID)
	if err != nil {
		t.Fatalf("approve imported request with local sender profile: %v", err)
	}
	if updated.State != protocol.StateApproved {
		t.Fatalf("expected approved, got %q", updated.State)
	}
	received := filepath.Join(paths.ReceivedArtifactsDir("denis"), record.Artifact.ID, "proposal.md")
	if _, err := os.Stat(received); err != nil {
		t.Fatalf("expected received artifact: %v", err)
	}
}
