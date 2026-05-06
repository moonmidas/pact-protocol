package protocol

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewIDUsesPrefix(t *testing.T) {
	id, err := NewID("req")
	if err != nil {
		t.Fatalf("NewID returned error: %v", err)
	}
	if !strings.HasPrefix(id, "req_") {
		t.Fatalf("expected req_ prefix, got %q", id)
	}
}

func TestEnvelopeJSONShape(t *testing.T) {
	env := Envelope{
		ID:        "msg_1",
		Type:      MessageGrantRequest,
		Version:   Version,
		CreatedAt: time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC),
		From: PartyRef{
			Profile:   "esteban",
			HumanID:   "human_1",
			GatewayID: "gateway_1",
		},
		To: PartyRef{
			Profile:   "denis",
			HumanID:   "human_2",
			GatewayID: "gateway_2",
		},
		CorrelationID: "req_1",
		Body:          json.RawMessage(`{"hello":"world"}`),
	}

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	if !strings.Contains(string(data), `"type":"grant.request"`) {
		t.Fatalf("expected grant.request JSON, got %s", string(data))
	}
}
