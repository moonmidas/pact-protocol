package requests

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pact/internal/audit"
	"pact/internal/pairing"
	"pact/internal/protocol"
	"pact/internal/store"
)

const PayloadShareRequest = "pact.share_request"

func ShareFile(paths store.Paths, fromProfile string, toHandle string, filePath string) (protocol.RequestRecord, error) {
	from, err := pairing.LoadProfile(paths, fromProfile)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	contact, err := findContact(paths, fromProfile, toHandle)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	if isSecretLike(filePath) {
		return protocol.RequestRecord{}, fmt.Errorf("refusing to share secret-like file: %s", filePath)
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	if info.IsDir() {
		return protocol.RequestRecord{}, fmt.Errorf("share path must be a file: %s", filePath)
	}

	requestID, err := protocol.NewID("req")
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	messageID, err := protocol.NewID("msg")
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	artifactID, err := protocol.NewID("art")
	if err != nil {
		return protocol.RequestRecord{}, err
	}

	name := filepath.Base(filePath)
	pendingDir := filepath.Join(paths.PendingArtifactsDir(fromProfile), artifactID)
	pendingPath := filepath.Join(pendingDir, name)
	if err := copyFile(filePath, pendingPath); err != nil {
		return protocol.RequestRecord{}, err
	}
	hash, err := sha256File(filePath)
	if err != nil {
		return protocol.RequestRecord{}, err
	}

	now := time.Now().UTC()
	artifact := protocol.Artifact{
		ID:          artifactID,
		Name:        name,
		SourcePath:  filePath,
		PendingPath: pendingPath,
		SizeBytes:   info.Size(),
		SHA256:      hash,
		MIME:        mimeForPath(filePath),
	}
	senderRecord := protocol.RequestRecord{
		ID:          requestID,
		MessageID:   messageID,
		Type:        "artifact.share",
		State:       protocol.StateCreated,
		FromProfile: fromProfile,
		ToProfile:   contact.Handle,
		Artifact:    artifact,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	recipientRecord := senderRecord
	recipientRecord.State = protocol.StateDelivered

	if err := upsertRequest(paths.RequestsFile(fromProfile), senderRecord); err != nil {
		return protocol.RequestRecord{}, err
	}
	if err := upsertRequest(paths.RequestsFile(contact.Handle), recipientRecord); err != nil {
		return protocol.RequestRecord{}, err
	}

	body, err := json.Marshal(recipientRecord)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	env := protocol.Envelope{
		ID:        messageID,
		Type:      protocol.MessageGrantRequest,
		Version:   protocol.Version,
		CreatedAt: now,
		From: protocol.PartyRef{
			Profile:   from.Profile,
			HumanID:   from.HumanID,
			GatewayID: from.GatewayID,
		},
		To: protocol.PartyRef{
			Profile:   contact.Handle,
			HumanID:   contact.HumanID,
			GatewayID: contact.GatewayID,
		},
		CorrelationID: requestID,
		Body:          body,
	}
	if err := store.WriteJSON(filepath.Join(contact.Delivery.Address, messageID+".json"), env); err != nil {
		return protocol.RequestRecord{}, err
	}

	if err := appendAudit(paths, fromProfile, "request.created", requestID, protocol.MessageGrantRequest, fmt.Sprintf("Created request to share %s with %s.", name, contact.Handle)); err != nil {
		return protocol.RequestRecord{}, err
	}
	if err := appendAudit(paths, contact.Handle, "request.delivered", requestID, protocol.MessageGrantRequest, fmt.Sprintf("Received request to receive %s from %s.", name, fromProfile)); err != nil {
		return protocol.RequestRecord{}, err
	}
	return senderRecord, nil
}

func ExportSharePayload(paths store.Paths, fromProfile string, toProfile string, filePath string) (protocol.SharePayload, error) {
	from, err := pairing.LoadProfile(paths, fromProfile)
	if err != nil {
		return protocol.SharePayload{}, err
	}
	if strings.TrimSpace(toProfile) == "" {
		return protocol.SharePayload{}, fmt.Errorf("to profile is required")
	}
	if isSecretLike(filePath) {
		return protocol.SharePayload{}, fmt.Errorf("refusing to share secret-like file: %s", filePath)
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return protocol.SharePayload{}, err
	}
	if info.IsDir() {
		return protocol.SharePayload{}, fmt.Errorf("share path must be a file: %s", filePath)
	}
	content, err := os.ReadFile(filePath)
	if err != nil {
		return protocol.SharePayload{}, err
	}
	hash, err := sha256File(filePath)
	if err != nil {
		return protocol.SharePayload{}, err
	}
	requestID, err := protocol.NewID("req")
	if err != nil {
		return protocol.SharePayload{}, err
	}
	messageID, err := protocol.NewID("msg")
	if err != nil {
		return protocol.SharePayload{}, err
	}
	artifactID, err := protocol.NewID("art")
	if err != nil {
		return protocol.SharePayload{}, err
	}

	now := time.Now().UTC()
	record := protocol.RequestRecord{
		ID:          requestID,
		MessageID:   messageID,
		Type:        "artifact.share",
		State:       protocol.StateDelivered,
		FromProfile: fromProfile,
		ToProfile:   toProfile,
		Artifact: protocol.Artifact{
			ID:         artifactID,
			Name:       filepath.Base(filePath),
			SourcePath: filePath,
			SizeBytes:  info.Size(),
			SHA256:     hash,
			MIME:       mimeForPath(filePath),
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	body, err := json.Marshal(record)
	if err != nil {
		return protocol.SharePayload{}, err
	}
	return protocol.SharePayload{
		Type:      PayloadShareRequest,
		Version:   protocol.Version,
		CreatedAt: now,
		Envelope: protocol.Envelope{
			ID:        messageID,
			Type:      protocol.MessageGrantRequest,
			Version:   protocol.Version,
			CreatedAt: now,
			From: protocol.PartyRef{
				Profile:   from.Profile,
				HumanID:   from.HumanID,
				GatewayID: from.GatewayID,
			},
			To: protocol.PartyRef{
				Profile: toProfile,
			},
			CorrelationID: requestID,
			Body:          body,
		},
		Request:               record,
		ArtifactContentBase64: base64.StdEncoding.EncodeToString(content),
	}, nil
}

func ImportSharePayload(paths store.Paths, asProfile string, payloadPath string) (protocol.RequestRecord, error) {
	profile, err := pairing.LoadProfile(paths, asProfile)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	var payload protocol.SharePayload
	if err := store.ReadJSON(payloadPath, &payload); err != nil {
		return protocol.RequestRecord{}, fmt.Errorf("read payload: %w", err)
	}
	if payload.Type != PayloadShareRequest {
		return protocol.RequestRecord{}, fmt.Errorf("unsupported payload type: %s", payload.Type)
	}
	content, err := base64.StdEncoding.DecodeString(payload.ArtifactContentBase64)
	if err != nil {
		return protocol.RequestRecord{}, fmt.Errorf("decode artifact: %w", err)
	}
	record := payload.Request
	actualHash := sha256.Sum256(content)
	if record.Artifact.SHA256 == "" {
		return protocol.RequestRecord{}, fmt.Errorf("artifact hash is missing")
	}
	if hex.EncodeToString(actualHash[:]) != record.Artifact.SHA256 {
		return protocol.RequestRecord{}, fmt.Errorf("artifact hash mismatch for %s", record.Artifact.Name)
	}
	record.State = protocol.StateDelivered
	record.ToProfile = asProfile
	record.UpdatedAt = time.Now().UTC()
	pendingPath := filepath.Join(paths.PendingArtifactsDir(asProfile), "imported", record.Artifact.ID, record.Artifact.Name)
	if err := os.MkdirAll(filepath.Dir(pendingPath), 0o755); err != nil {
		return protocol.RequestRecord{}, err
	}
	if err := os.WriteFile(pendingPath, content, 0o644); err != nil {
		return protocol.RequestRecord{}, err
	}
	record.Artifact.PendingPath = pendingPath

	body, err := json.Marshal(record)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	payload.Envelope.To = protocol.PartyRef{
		Profile:   profile.Profile,
		HumanID:   profile.HumanID,
		GatewayID: profile.GatewayID,
	}
	payload.Envelope.Body = body

	if err := upsertRequest(paths.RequestsFile(asProfile), record); err != nil {
		return protocol.RequestRecord{}, err
	}
	if err := store.WriteJSON(filepath.Join(paths.InboxDir(asProfile), payload.Envelope.ID+".json"), payload.Envelope); err != nil {
		return protocol.RequestRecord{}, err
	}
	if err := appendAudit(paths, asProfile, "request.delivered", record.ID, protocol.MessageGrantRequest, fmt.Sprintf("Imported request to receive %s from %s.", record.Artifact.Name, record.FromProfile)); err != nil {
		return protocol.RequestRecord{}, err
	}
	return record, nil
}

func ListInbox(paths store.Paths, profile string) ([]protocol.Envelope, error) {
	entries, err := os.ReadDir(paths.InboxDir(profile))
	if os.IsNotExist(err) {
		return []protocol.Envelope{}, nil
	}
	if err != nil {
		return nil, err
	}
	var messages []protocol.Envelope
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		var env protocol.Envelope
		if err := store.ReadJSON(filepath.Join(paths.InboxDir(profile), entry.Name()), &env); err != nil {
			return nil, err
		}
		messages = append(messages, env)
	}
	return messages, nil
}

func Approve(paths store.Paths, asProfile string, requestID string) (protocol.RequestRecord, error) {
	return decide(paths, asProfile, requestID, protocol.StateApproved, "")
}

func Reject(paths store.Paths, asProfile string, requestID string) (protocol.RequestRecord, error) {
	return decide(paths, asProfile, requestID, protocol.StateDenied, "")
}

func Counter(paths store.Paths, asProfile string, requestID string, message string) (protocol.RequestRecord, error) {
	return decide(paths, asProfile, requestID, protocol.StateCountered, message)
}

func decide(paths store.Paths, asProfile string, requestID string, state string, message string) (protocol.RequestRecord, error) {
	requests, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile(asProfile))
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	idx := -1
	for i := range requests {
		if requests[i].ID == requestID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return protocol.RequestRecord{}, fmt.Errorf("request not found: %s", requestID)
	}
	record := requests[idx]
	if record.State != protocol.StateDelivered {
		return protocol.RequestRecord{}, fmt.Errorf("request %s is not delivered; current state %s", requestID, record.State)
	}

	now := time.Now().UTC()
	record.State = state
	record.UpdatedAt = now
	record.Decision = &protocol.Decision{Type: state, Message: message, CreatedAt: now}
	requests[idx] = record
	if err := store.WriteJSONArray(paths.RequestsFile(asProfile), requests); err != nil {
		return protocol.RequestRecord{}, err
	}

	senderRecord := record
	senderRecord.ToProfile = asProfile
	senderExists := profileExists(paths, record.FromProfile)
	if senderExists {
		if err := updateSenderRequest(paths, record.FromProfile, senderRecord); err != nil {
			return protocol.RequestRecord{}, err
		}
	}

	switch state {
	case protocol.StateApproved:
		if err := createGrant(paths, asProfile, requestID, now); err != nil {
			return protocol.RequestRecord{}, err
		}
		receivedPath := filepath.Join(paths.ReceivedArtifactsDir(asProfile), record.Artifact.ID, record.Artifact.Name)
		if err := copyFile(record.Artifact.PendingPath, receivedPath); err != nil {
			return protocol.RequestRecord{}, err
		}
		if senderExists {
			if err := writeDecisionEnvelope(paths, asProfile, record.FromProfile, protocol.MessageGrantApproved, record); err != nil {
				return protocol.RequestRecord{}, err
			}
			if err := writeDecisionEnvelope(paths, asProfile, record.FromProfile, protocol.MessageArtifactShared, record); err != nil {
				return protocol.RequestRecord{}, err
			}
		}
		if err := appendAudit(paths, asProfile, "request.approved", requestID, protocol.MessageGrantApproved, fmt.Sprintf("Approved receiving %s from %s.", record.Artifact.Name, record.FromProfile)); err != nil {
			return protocol.RequestRecord{}, err
		}
		if err := appendAudit(paths, asProfile, "artifact.received", record.Artifact.ID, protocol.MessageArtifactShared, fmt.Sprintf("Received %s.", record.Artifact.Name)); err != nil {
			return protocol.RequestRecord{}, err
		}
		if senderExists {
			if err := appendAudit(paths, record.FromProfile, "request.approved", requestID, protocol.MessageGrantApproved, fmt.Sprintf("%s approved receiving %s.", asProfile, record.Artifact.Name)); err != nil {
				return protocol.RequestRecord{}, err
			}
		}
	case protocol.StateDenied:
		if senderExists {
			if err := writeDecisionEnvelope(paths, asProfile, record.FromProfile, protocol.MessageGrantDenied, record); err != nil {
				return protocol.RequestRecord{}, err
			}
		}
		if err := appendAudit(paths, asProfile, "request.denied", requestID, protocol.MessageGrantDenied, fmt.Sprintf("Rejected receiving %s from %s.", record.Artifact.Name, record.FromProfile)); err != nil {
			return protocol.RequestRecord{}, err
		}
		if senderExists {
			if err := appendAudit(paths, record.FromProfile, "request.denied", requestID, protocol.MessageGrantDenied, fmt.Sprintf("%s rejected receiving %s.", asProfile, record.Artifact.Name)); err != nil {
				return protocol.RequestRecord{}, err
			}
		}
	case protocol.StateCountered:
		if senderExists {
			if err := writeDecisionEnvelope(paths, asProfile, record.FromProfile, protocol.MessageGrantCounter, record); err != nil {
				return protocol.RequestRecord{}, err
			}
		}
		if err := appendAudit(paths, asProfile, "request.countered", requestID, protocol.MessageGrantCounter, fmt.Sprintf("Countered request for %s.", record.Artifact.Name)); err != nil {
			return protocol.RequestRecord{}, err
		}
		if senderExists {
			if err := appendAudit(paths, record.FromProfile, "request.countered", requestID, protocol.MessageGrantCounter, fmt.Sprintf("%s countered request for %s.", asProfile, record.Artifact.Name)); err != nil {
				return protocol.RequestRecord{}, err
			}
		}
	default:
		return protocol.RequestRecord{}, fmt.Errorf("unsupported decision state: %s", state)
	}
	return record, nil
}

func profileExists(paths store.Paths, profile string) bool {
	_, err := pairing.LoadProfile(paths, profile)
	return err == nil
}

func findContact(paths store.Paths, profile string, handle string) (protocol.Contact, error) {
	contacts, err := store.ReadJSONArray[protocol.Contact](paths.ContactsFile(profile))
	if err != nil {
		return protocol.Contact{}, err
	}
	for _, contact := range contacts {
		if contact.Handle == handle {
			return contact, nil
		}
	}
	return protocol.Contact{}, fmt.Errorf("contact not found: %s", handle)
}

func upsertRequest(path string, record protocol.RequestRecord) error {
	requests, err := store.ReadJSONArray[protocol.RequestRecord](path)
	if err != nil {
		return err
	}
	for i := range requests {
		if requests[i].ID == record.ID {
			requests[i] = record
			return store.WriteJSONArray(path, requests)
		}
	}
	requests = append(requests, record)
	return store.WriteJSONArray(path, requests)
}

func updateSenderRequest(paths store.Paths, senderProfile string, record protocol.RequestRecord) error {
	requests, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile(senderProfile))
	if err != nil {
		return err
	}
	for i := range requests {
		if requests[i].ID == record.ID {
			requests[i].State = record.State
			requests[i].UpdatedAt = record.UpdatedAt
			requests[i].Decision = record.Decision
			return store.WriteJSONArray(paths.RequestsFile(senderProfile), requests)
		}
	}
	return nil
}

func createGrant(paths store.Paths, profile string, requestID string, now time.Time) error {
	grantID, err := protocol.NewID("grt")
	if err != nil {
		return err
	}
	grants, err := store.ReadJSONArray[protocol.GrantRecord](paths.GrantsFile(profile))
	if err != nil {
		return err
	}
	grants = append(grants, protocol.GrantRecord{
		ID:           grantID,
		RequestID:    requestID,
		State:        "active",
		Capabilities: []string{"artifact.receive"},
		CreatedAt:    now,
	})
	return store.WriteJSONArray(paths.GrantsFile(profile), grants)
}

func writeDecisionEnvelope(paths store.Paths, fromProfile string, toProfile string, messageType string, record protocol.RequestRecord) error {
	from, err := pairing.LoadProfile(paths, fromProfile)
	if err != nil {
		return err
	}
	to, err := pairing.LoadProfile(paths, toProfile)
	if err != nil {
		return err
	}
	msgID, err := protocol.NewID("msg")
	if err != nil {
		return err
	}
	body, err := json.Marshal(record)
	if err != nil {
		return err
	}
	env := protocol.Envelope{
		ID:        msgID,
		Type:      messageType,
		Version:   protocol.Version,
		CreatedAt: time.Now().UTC(),
		From: protocol.PartyRef{
			Profile:   from.Profile,
			HumanID:   from.HumanID,
			GatewayID: from.GatewayID,
		},
		To: protocol.PartyRef{
			Profile:   to.Profile,
			HumanID:   to.HumanID,
			GatewayID: to.GatewayID,
		},
		CorrelationID: record.ID,
		Body:          body,
	}
	if err := store.WriteJSON(filepath.Join(paths.OutboxDir(fromProfile), msgID+".json"), env); err != nil {
		return err
	}
	return store.WriteJSON(filepath.Join(paths.InboxDir(toProfile), msgID+".json"), env)
}

func appendAudit(paths store.Paths, profileName string, eventType string, subjectID string, messageType string, summary string) error {
	profile, err := pairing.LoadProfile(paths, profileName)
	if err != nil {
		return err
	}
	id, err := protocol.NewID("aud")
	if err != nil {
		return err
	}
	return audit.Append(paths.AuditFile(profileName), protocol.AuditEvent{
		ID:             id,
		Timestamp:      time.Now().UTC(),
		Profile:        profileName,
		ActorGatewayID: profile.GatewayID,
		EventType:      eventType,
		SubjectID:      subjectID,
		MessageType:    messageType,
		Summary:        summary,
	})
}

func isSecretLike(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	full := strings.ToLower(path)
	return base == ".env" || strings.HasPrefix(base, ".env.") || strings.Contains(full, "secret")
}

func sha256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func copyFile(src string, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func mimeForPath(path string) string {
	if typ := mime.TypeByExtension(filepath.Ext(path)); typ != "" {
		return typ
	}
	return "application/octet-stream"
}
