package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"

	"pact/internal/protocol"
)

func Append(path string, event protocol.AuditEvent) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := file.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func Read(path string) ([]protocol.AuditEvent, error) {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return []protocol.AuditEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var events []protocol.AuditEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event protocol.AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
