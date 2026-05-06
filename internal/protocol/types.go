package protocol

import (
	"encoding/json"
	"time"
)

const Version = "0.1"

const (
	MessagePairInvite     = "pair.invite"
	MessagePairAccepted   = "pair.accepted"
	MessageGrantRequest   = "grant.request"
	MessageGrantCounter   = "grant.counter"
	MessageGrantApproved  = "grant.approved"
	MessageGrantDenied    = "grant.denied"
	MessageArtifactShared = "artifact.shared"
	MessageAuditEvent     = "audit.event"
)

const (
	StateCreated   = "created"
	StateDelivered = "delivered"
	StateApproved  = "approved"
	StateDenied    = "denied"
	StateCountered = "countered"
)

type PartyRef struct {
	Profile   string `json:"profile"`
	HumanID   string `json:"human_id"`
	GatewayID string `json:"gateway_id"`
}

type Envelope struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"`
	Version       string          `json:"version"`
	CreatedAt     time.Time       `json:"created_at"`
	From          PartyRef        `json:"from"`
	To            PartyRef        `json:"to"`
	CorrelationID string          `json:"correlation_id"`
	Body          json.RawMessage `json:"body"`
}

type Delivery struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type Profile struct {
	Profile     string    `json:"profile"`
	HumanID     string    `json:"human_id"`
	GatewayID   string    `json:"gateway_id"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

type Contact struct {
	Handle      string    `json:"handle"`
	HumanID     string    `json:"human_id"`
	DisplayName string    `json:"display_name"`
	GatewayID   string    `json:"gateway_id"`
	Delivery    Delivery  `json:"delivery"`
	CreatedAt   time.Time `json:"created_at"`
}

type Invite struct {
	ID              string        `json:"id"`
	Type            string        `json:"type"`
	Version         string        `json:"version"`
	CreatedAt       time.Time     `json:"created_at"`
	ExpiresAt       *time.Time    `json:"expires_at"`
	Human           InviteHuman   `json:"human"`
	Gateway         InviteGateway `json:"gateway"`
	SuggestedHandle string        `json:"suggested_handle"`
}

type InviteHuman struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type InviteGateway struct {
	ID       string   `json:"id"`
	Delivery Delivery `json:"delivery"`
}

type Artifact struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SourcePath  string `json:"source_path"`
	PendingPath string `json:"pending_path"`
	SizeBytes   int64  `json:"size_bytes"`
	SHA256      string `json:"sha256"`
	MIME        string `json:"mime"`
}

type RequestRecord struct {
	ID          string    `json:"id"`
	MessageID   string    `json:"message_id"`
	Type        string    `json:"type"`
	State       string    `json:"state"`
	FromProfile string    `json:"from_profile"`
	ToProfile   string    `json:"to_profile"`
	Artifact    Artifact  `json:"artifact"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Decision    *Decision `json:"decision"`
}

type Decision struct {
	Type      string    `json:"type"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type GrantRecord struct {
	ID           string     `json:"id"`
	RequestID    string     `json:"request_id"`
	State        string     `json:"state"`
	Capabilities []string   `json:"capabilities"`
	CreatedAt    time.Time  `json:"created_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type AuditEvent struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	Profile        string    `json:"profile"`
	ActorGatewayID string    `json:"actor_gateway_id"`
	EventType      string    `json:"event_type"`
	SubjectID      string    `json:"subject_id"`
	MessageType    string    `json:"message_type"`
	Summary        string    `json:"summary"`
}

type SharePayload struct {
	Type                  string        `json:"type"`
	Version               string        `json:"version"`
	CreatedAt             time.Time     `json:"created_at"`
	Envelope              Envelope      `json:"envelope"`
	Request               RequestRecord `json:"request"`
	ArtifactContentBase64 string        `json:"artifact_content_base64"`
}
