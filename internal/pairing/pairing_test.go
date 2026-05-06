package pairing

import (
	"os"
	"path/filepath"
	"testing"

	"pact/internal/protocol"
	"pact/internal/store"
)

func TestInitProfileCreatesProfileAndEmptyState(t *testing.T) {
	root := t.TempDir()
	paths := store.NewPaths(root)

	profile, err := InitProfile(paths, "esteban")
	if err != nil {
		t.Fatalf("InitProfile: %v", err)
	}
	if profile.Profile != "esteban" {
		t.Fatalf("expected profile esteban, got %q", profile.Profile)
	}

	var saved protocol.Profile
	if err := store.ReadJSON(paths.ProfileFile("esteban"), &saved); err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if saved.HumanID == "" || saved.GatewayID == "" {
		t.Fatalf("expected generated IDs, got %#v", saved)
	}
	if contacts, err := store.ReadJSONArray[protocol.Contact](paths.ContactsFile("esteban")); err != nil || len(contacts) != 0 {
		t.Fatalf("expected empty contacts, got %#v err=%v", contacts, err)
	}
	if requests, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile("esteban")); err != nil || len(requests) != 0 {
		t.Fatalf("expected empty requests, got %#v err=%v", requests, err)
	}
	if grants, err := store.ReadJSONArray[protocol.GrantRecord](paths.GrantsFile("esteban")); err != nil || len(grants) != 0 {
		t.Fatalf("expected empty grants, got %#v err=%v", grants, err)
	}
	if _, err := os.Stat(filepath.Join(paths.ProfileDir("esteban"), "audit.jsonl")); err != nil {
		t.Fatalf("expected audit log: %v", err)
	}
}

func TestCreateInviteAndAcceptPairing(t *testing.T) {
	root := t.TempDir()
	paths := store.NewPaths(root)
	if _, err := InitProfile(paths, "denis"); err != nil {
		t.Fatalf("init denis: %v", err)
	}
	if _, err := InitProfile(paths, "esteban"); err != nil {
		t.Fatalf("init esteban: %v", err)
	}

	invitePath, invite, err := CreateInvite(paths, "denis")
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	if invite.Type != protocol.MessagePairInvite {
		t.Fatalf("expected pair invite, got %q", invite.Type)
	}
	if invitePath == "" || !InvitePathExists(invitePath) {
		t.Fatalf("expected invite path, got %q", invitePath)
	}

	contact, err := AcceptInvite(paths, "esteban", "denis", invitePath)
	if err != nil {
		t.Fatalf("AcceptInvite: %v", err)
	}
	if contact.Handle != "denis" || contact.GatewayID != invite.Gateway.ID {
		t.Fatalf("unexpected contact: %#v", contact)
	}

	contacts, err := store.ReadJSONArray[protocol.Contact](paths.ContactsFile("esteban"))
	if err != nil {
		t.Fatalf("read contacts: %v", err)
	}
	if len(contacts) != 1 || contacts[0].Handle != "denis" {
		t.Fatalf("unexpected contacts: %#v", contacts)
	}
}
