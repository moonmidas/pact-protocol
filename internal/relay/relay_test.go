package relay

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateAndFetchPayload(t *testing.T) {
	server := New(t.TempDir(), "https://wepact.online")
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/payloads", "application/json", bytes.NewBufferString(`{"type":"pact.share_request"}`))
	if err != nil {
		t.Fatalf("post payload: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", resp.StatusCode)
	}

	var created CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.ID == "" || !strings.Contains(created.URL, "/i/") {
		t.Fatalf("unexpected create response: %#v", created)
	}

	fetch, err := http.Get(ts.URL + "/api/payloads/" + created.ID)
	if err != nil {
		t.Fatalf("fetch payload: %v", err)
	}
	defer fetch.Body.Close()
	if fetch.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", fetch.StatusCode)
	}
}
