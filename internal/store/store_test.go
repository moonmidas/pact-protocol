package store

import (
	"os"
	"path/filepath"
	"testing"

	"pact/internal/protocol"
)

func TestWriteAndReadJSON(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "profile.json")
	profile := protocol.Profile{Profile: "esteban", HumanID: "human_1"}

	if err := WriteJSON(path, profile); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	var got protocol.Profile
	if err := ReadJSON(path, &got); err != nil {
		t.Fatalf("ReadJSON: %v", err)
	}
	if got.Profile != "esteban" {
		t.Fatalf("expected profile esteban, got %q", got.Profile)
	}
}

func TestEnsureProfileDirs(t *testing.T) {
	root := t.TempDir()
	paths := NewPaths(root)

	if err := paths.EnsureProfileDirs("denis"); err != nil {
		t.Fatalf("EnsureProfileDirs: %v", err)
	}

	for _, rel := range []string{"inbox", "outbox", "artifacts/pending", "artifacts/received"} {
		path := filepath.Join(root, "profiles", "denis", rel)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %s to be directory", path)
		}
	}
}
