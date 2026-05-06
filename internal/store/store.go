package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Paths struct {
	Root string
}

func NewPaths(root string) Paths {
	if root == "" {
		root = ".pact-local"
	}
	return Paths{Root: root}
}

func (p Paths) InvitesDir() string {
	return filepath.Join(p.Root, "invites")
}

func (p Paths) ProfileDir(profile string) string {
	return filepath.Join(p.Root, "profiles", profile)
}

func (p Paths) ProfileFile(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "profile.json")
}

func (p Paths) ContactsFile(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "contacts.json")
}

func (p Paths) RequestsFile(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "requests.json")
}

func (p Paths) GrantsFile(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "grants.json")
}

func (p Paths) AuditFile(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "audit.jsonl")
}

func (p Paths) InboxDir(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "inbox")
}

func (p Paths) OutboxDir(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "outbox")
}

func (p Paths) PendingArtifactsDir(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "artifacts", "pending")
}

func (p Paths) ReceivedArtifactsDir(profile string) string {
	return filepath.Join(p.ProfileDir(profile), "artifacts", "received")
}

func (p Paths) EnsureProfileDirs(profile string) error {
	for _, dir := range []string{
		p.ProfileDir(profile),
		p.InboxDir(profile),
		p.OutboxDir(profile),
		p.PendingArtifactsDir(profile),
		p.ReceivedArtifactsDir(profile),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.MkdirAll(p.InvitesDir(), 0o755)
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func ReadJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}

func ReadJSONArray[T any](path string) ([]T, error) {
	var values []T
	err := ReadJSON(path, &values)
	if errors.Is(err, os.ErrNotExist) {
		return []T{}, nil
	}
	return values, err
}

func WriteJSONArray[T any](path string, values []T) error {
	if values == nil {
		values = []T{}
	}
	return WriteJSON(path, values)
}
