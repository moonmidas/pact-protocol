package cli

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pact/internal/protocol"
	"pact/internal/relay"
)

func TestRunShowsHelpForNoArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage: pact") {
		t.Fatalf("expected usage in stderr, got %q", stderr.String())
	}
}

func TestRunInitCreatesProfile(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"init", "--root", root, "--profile", "esteban"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "initialized profile esteban") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunInviteAndPairAccept(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"init", "--root", root, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init denis failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", root, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init esteban failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	if code := Run([]string{"invite", "create", "--root", root, "--from", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("invite failed: %s", stderr.String())
	}
	fields := strings.Fields(stdout.String())
	invitePath := fields[len(fields)-1]

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pair", "accept", invitePath, "--root", root, "--as", "esteban", "--handle", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pair accept failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "paired esteban with denis") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunRequestShareInboxApproveAndAudit(t *testing.T) {
	root := t.TempDir()
	invitePath := setupPairedProfilesViaCLI(t, root)
	if invitePath == "" {
		t.Fatal("expected invite path")
	}
	proposalPath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"request", "share", proposalPath, "--root", root, "--from", "esteban", "--to", "denis"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("request share failed: %s", stderr.String())
	}
	fields := strings.Fields(stdout.String())
	requestID := fields[3]

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"inbox", "--root", root, "--as", "denis"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("inbox failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "proposal.md") {
		t.Fatalf("expected proposal in inbox, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"approve", requestID, "--root", root, "--as", "denis"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("approve failed: %s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"audit", "--root", root, "--as", "denis"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("audit failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "artifact.received") {
		t.Fatalf("expected artifact.received audit event, got %s", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"audit", "--root", root, "--as", "esteban"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("audit esteban failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "request.approved") {
		t.Fatalf("expected request.approved in sender audit, got %s", stdout.String())
	}
}

func TestRunCounter(t *testing.T) {
	root := t.TempDir()
	setupPairedProfilesViaCLI(t, root)
	proposalPath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"request", "share", proposalPath, "--root", root, "--from", "esteban", "--to", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("request share failed: %s", stderr.String())
	}
	requestID := strings.Fields(stdout.String())[3]
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"counter", requestID, "--root", root, "--as", "denis", "--message", "Send a summary instead."}, &stdout, &stderr); code != 0 {
		t.Fatalf("counter failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "countered request") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestRunDemoCompletesLocalFlow(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"demo", "--root", root}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("demo failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Pact demo complete") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "received:") {
		t.Fatalf("expected received path, got %s", stdout.String())
	}
}

func TestRunPayloadCreateAndImport(t *testing.T) {
	senderRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"init", "--root", senderRoot, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init sender failed: %s", stderr.String())
	}
	proposalPath := filepath.Join(senderRoot, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	payloadPath := filepath.Join(senderRoot, "pact-payload.json")
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"payload", "create", proposalPath, "--root", senderRoot, "--from", "esteban", "--to", "denis", "--out", payloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("payload create failed: %s", stderr.String())
	}

	receiverRoot := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", receiverRoot, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init receiver failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"payload", "import", payloadPath, "--root", receiverRoot, "--as", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("payload import failed: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Pact Request") {
		t.Fatalf("expected request card, got %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "proposal.md") {
		t.Fatalf("expected proposal in request card, got %s", stdout.String())
	}
}

func TestRunLinkCreateAndOpenHostedPayload(t *testing.T) {
	relayServer := relay.New(t.TempDir(), "")
	ts := httptest.NewServer(relayServer.Handler())
	defer ts.Close()

	senderRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", senderRoot, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init sender failed: %s", stderr.String())
	}
	proposalPath := filepath.Join(senderRoot, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n\nFirst Denis test.\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"link", "create", proposalPath, "--root", senderRoot, "--from", "esteban", "--to", "denis", "--relay", ts.URL}, &stdout, &stderr); code != 0 {
		t.Fatalf("link create failed: %s", stderr.String())
	}
	fields := strings.Fields(stdout.String())
	link := fields[3]
	if !strings.HasPrefix(link, ts.URL+"/i/") {
		t.Fatalf("expected hosted invite URL, got %q", stdout.String())
	}

	receiverRoot := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", receiverRoot, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init receiver failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"link", "open", link, "--root", receiverRoot, "--as", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("link open failed: %s", stderr.String())
	}
	card := stdout.String()
	if !strings.Contains(card, "Pact Request") || !strings.Contains(card, "First Denis test") {
		t.Fatalf("expected clear opened request card with preview, got %s", card)
	}
	if !strings.Contains(card, "Approving will copy the artifact into denis's received folder") {
		t.Fatalf("expected approval consequence in card, got %s", card)
	}
}

func TestRunPayloadImportRejectsTamperedArtifactContent(t *testing.T) {
	senderRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", senderRoot, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init sender failed: %s", stderr.String())
	}
	proposalPath := filepath.Join(senderRoot, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Original\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	payloadPath := filepath.Join(senderRoot, "pact-payload.json")
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"payload", "create", proposalPath, "--root", senderRoot, "--from", "esteban", "--to", "denis", "--out", payloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("payload create failed: %s", stderr.String())
	}

	var payload protocol.SharePayload
	data, err := os.ReadFile(payloadPath)
	if err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	payload.ArtifactContentBase64 = base64.StdEncoding.EncodeToString([]byte("# Tampered\n"))
	tampered, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("encode tampered payload: %v", err)
	}
	if err := os.WriteFile(payloadPath, append(tampered, '\n'), 0o644); err != nil {
		t.Fatalf("write tampered payload: %v", err)
	}

	receiverRoot := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", receiverRoot, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init receiver failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := Run([]string{"payload", "import", payloadPath, "--root", receiverRoot, "--as", "denis"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected import failure, got code %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "artifact hash mismatch") {
		t.Fatalf("expected useful tamper error, got %s", stderr.String())
	}
}

func TestRunPayloadImportJSONReturnsStableRequestData(t *testing.T) {
	senderRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", senderRoot, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init sender failed: %s", stderr.String())
	}
	proposalPath := filepath.Join(senderRoot, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n\nJSON bridge.\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	payloadPath := filepath.Join(senderRoot, "pact-payload.json")
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"payload", "create", proposalPath, "--root", senderRoot, "--from", "esteban", "--to", "denis", "--out", payloadPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("payload create failed: %s", stderr.String())
	}

	receiverRoot := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", receiverRoot, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init receiver failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"payload", "import", payloadPath, "--root", receiverRoot, "--as", "denis", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("payload import failed: %s", stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json response: %v\n%s", err, stdout.String())
	}
	if response["ok"] != true || response["operation"] != "payload.import" {
		t.Fatalf("unexpected response envelope: %#v", response)
	}
	request := response["request"].(map[string]any)
	if request["id"] == "" || request["state"] != "delivered" || request["from_profile"] != "esteban" || request["to_profile"] != "denis" {
		t.Fatalf("unexpected request: %#v", request)
	}
	artifact := request["artifact"].(map[string]any)
	if artifact["name"] != "proposal.md" || artifact["preview"] != "# Proposal JSON bridge." || artifact["pending_path"] == "" {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
	actions := response["actions"].(map[string]any)
	if !strings.Contains(actions["approve"].(string), "pact approve") || !strings.Contains(actions["counter"].(string), "pact counter") || !strings.Contains(actions["reject"].(string), "pact reject") {
		t.Fatalf("unexpected actions: %#v", actions)
	}
}

func TestRunInboxJSONReturnsStableRequestList(t *testing.T) {
	root := t.TempDir()
	requestID := createPendingShareRequestViaCLI(t, root)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"inbox", "--root", root, "--as", "denis", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("inbox failed: %s", stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr for json output, got %s", stderr.String())
	}

	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json response: %v\n%s", err, stdout.String())
	}
	if response["ok"] != true || response["operation"] != "inbox" || response["profile"] != "denis" {
		t.Fatalf("unexpected response envelope: %#v", response)
	}
	if response["count"].(float64) != 1 {
		t.Fatalf("expected count 1, got %#v", response["count"])
	}
	items := response["requests"].([]any)
	request := items[0].(map[string]any)
	if request["id"] != requestID || request["state"] != "delivered" || request["from_profile"] != "esteban" || request["to_profile"] != "denis" {
		t.Fatalf("unexpected request: %#v", request)
	}
	artifact := request["artifact"].(map[string]any)
	if artifact["name"] != "proposal.md" || artifact["preview"] != "# Proposal" {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
	actions := request["actions"].(map[string]any)
	if !strings.Contains(actions["approve"].(string), "pact approve") || !strings.Contains(actions["counter"].(string), "pact counter") || !strings.Contains(actions["reject"].(string), "pact reject") {
		t.Fatalf("unexpected actions: %#v", actions)
	}
}

func TestRunInboxJSONReturnsEmptyList(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", root, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	if code := Run([]string{"inbox", "--root", root, "--as", "denis", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("inbox failed: %s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json response: %v\n%s", err, stdout.String())
	}
	if response["ok"] != true || response["count"].(float64) != 0 {
		t.Fatalf("unexpected response: %#v", response)
	}
	if len(response["requests"].([]any)) != 0 {
		t.Fatalf("expected empty requests, got %#v", response["requests"])
	}
}

func TestRunInboxJSONErrorReturnsStableError(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"inbox", "--root", root, "--as", "", "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json error on stdout only, got stderr=%s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json error: %v\n%s", err, stdout.String())
	}
	if response["ok"] != false || response["operation"] != "inbox" {
		t.Fatalf("unexpected error envelope: %#v", response)
	}
	errObj := response["error"].(map[string]any)
	if errObj["code"] != "missing_required_flag" || errObj["message"] != "--as is required" {
		t.Fatalf("unexpected error object: %#v", errObj)
	}
}

func TestAppInboxFixtureMatchesJSONContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "app-inbox.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var response map[string]any
	if err := json.Unmarshal(data, &response); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	if response["ok"] != true || response["operation"] != "inbox" || response["profile"] != "denis" {
		t.Fatalf("unexpected fixture envelope: %#v", response)
	}
	if response["count"].(float64) != 1 {
		t.Fatalf("unexpected fixture count: %#v", response["count"])
	}
	request := response["requests"].([]any)[0].(map[string]any)
	artifact := request["artifact"].(map[string]any)
	actions := request["actions"].(map[string]any)
	if request["id"] == "" || artifact["preview"] == "" || actions["approve"] == "" || actions["counter"] == "" || actions["reject"] == "" {
		t.Fatalf("fixture is missing app fields: %#v", request)
	}
}

func TestRunLinkOpenJSONReturnsStableRequestData(t *testing.T) {
	relayServer := relay.New(t.TempDir(), "")
	ts := httptest.NewServer(relayServer.Handler())
	defer ts.Close()

	senderRoot := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", senderRoot, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init sender failed: %s", stderr.String())
	}
	proposalPath := filepath.Join(senderRoot, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Hosted\n\nJSON bridge.\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"link", "create", proposalPath, "--root", senderRoot, "--from", "esteban", "--to", "denis", "--relay", ts.URL}, &stdout, &stderr); code != 0 {
		t.Fatalf("link create failed: %s", stderr.String())
	}
	link := strings.Fields(stdout.String())[3]

	receiverRoot := t.TempDir()
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", receiverRoot, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init receiver failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"link", "open", link, "--root", receiverRoot, "--as", "denis", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("link open failed: %s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json response: %v\n%s", err, stdout.String())
	}
	if response["ok"] != true || response["operation"] != "link.open" || response["source_url"] != link {
		t.Fatalf("unexpected response: %#v", response)
	}
	request := response["request"].(map[string]any)
	artifact := request["artifact"].(map[string]any)
	if artifact["preview"] != "# Hosted JSON bridge." {
		t.Fatalf("unexpected artifact: %#v", artifact)
	}
}

func TestRunApproveJSONReturnsStableDecisionData(t *testing.T) {
	root := t.TempDir()
	setupPairedProfilesViaCLI(t, root)
	proposalPath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"request", "share", proposalPath, "--root", root, "--from", "esteban", "--to", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("request share failed: %s", stderr.String())
	}
	requestID := strings.Fields(stdout.String())[3]

	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"approve", requestID, "--root", root, "--as", "denis", "--json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("approve failed: %s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json response: %v\n%s", err, stdout.String())
	}
	if response["ok"] != true || response["operation"] != "approve" {
		t.Fatalf("unexpected response envelope: %#v", response)
	}
	decision := response["decision"].(map[string]any)
	if decision["type"] != "approved" || decision["by_profile"] != "denis" || decision["request_id"] != requestID {
		t.Fatalf("unexpected decision: %#v", decision)
	}
	request := response["request"].(map[string]any)
	if request["state"] != "approved" {
		t.Fatalf("expected approved request, got %#v", request)
	}
}

func TestRunCounterAndRejectJSONReturnStableDecisionData(t *testing.T) {
	t.Run("counter", func(t *testing.T) {
		root := t.TempDir()
		requestID := createPendingShareRequestViaCLI(t, root)
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		if code := Run([]string{"counter", requestID, "--root", root, "--as", "denis", "--message", "Send a summary instead.", "--json"}, &stdout, &stderr); code != 0 {
			t.Fatalf("counter failed: %s", stderr.String())
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("decode json response: %v\n%s", err, stdout.String())
		}
		if response["ok"] != true || response["operation"] != "counter" {
			t.Fatalf("unexpected response envelope: %#v", response)
		}
		decision := response["decision"].(map[string]any)
		if decision["type"] != "countered" || decision["message"] != "Send a summary instead." || decision["by_profile"] != "denis" {
			t.Fatalf("unexpected decision: %#v", decision)
		}
	})

	t.Run("reject", func(t *testing.T) {
		root := t.TempDir()
		requestID := createPendingShareRequestViaCLI(t, root)
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		if code := Run([]string{"reject", requestID, "--root", root, "--as", "denis", "--json"}, &stdout, &stderr); code != 0 {
			t.Fatalf("reject failed: %s", stderr.String())
		}
		var response map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
			t.Fatalf("decode json response: %v\n%s", err, stdout.String())
		}
		if response["ok"] != true || response["operation"] != "reject" {
			t.Fatalf("unexpected response envelope: %#v", response)
		}
		decision := response["decision"].(map[string]any)
		if decision["type"] != "denied" || decision["by_profile"] != "denis" {
			t.Fatalf("unexpected decision: %#v", decision)
		}
	})
}

func TestRunDecisionJSONErrorReturnsStableError(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", root, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	code := Run([]string{"approve", "req_missing", "--root", root, "--as", "denis", "--json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json errors on stdout only, got stderr=%s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json error: %v\n%s", err, stdout.String())
	}
	if response["ok"] != false || response["operation"] != "approve" {
		t.Fatalf("unexpected error envelope: %#v", response)
	}
	errObj := response["error"].(map[string]any)
	if errObj["code"] != "request_not_found" || !strings.Contains(errObj["message"].(string), "req_missing") {
		t.Fatalf("unexpected error object: %#v", errObj)
	}
}

func TestRunDecisionJSONErrorReturnsStableErrorForMissingRequestID(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"approve", "--json", "--as", "denis"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json error on stdout only, got stderr=%s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json error: %v\n%s", err, stdout.String())
	}
	if response["ok"] != false || response["operation"] != "approve" {
		t.Fatalf("unexpected error envelope: %#v", response)
	}
	errObj := response["error"].(map[string]any)
	if errObj["code"] != "missing_required_argument" {
		t.Fatalf("unexpected error object: %#v", errObj)
	}
}

func TestRunPayloadImportJSONErrorReturnsStableErrorForMissingPath(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"payload", "import", "--json", "--as", "denis"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json error on stdout only, got stderr=%s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json error: %v\n%s", err, stdout.String())
	}
	if response["ok"] != false || response["operation"] != "payload.import" {
		t.Fatalf("unexpected error envelope: %#v", response)
	}
	errObj := response["error"].(map[string]any)
	if errObj["code"] != "missing_required_argument" {
		t.Fatalf("unexpected error object: %#v", errObj)
	}
}

func TestRunLinkOpenJSONErrorReturnsStableErrorForInvalidURL(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"init", "--root", root, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()

	code := Run([]string{"link", "open", "not-a-link", "--root", root, "--as", "denis", "--json"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("expected exit 2, got %d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected json errors on stdout only, got stderr=%s", stderr.String())
	}
	var response map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode json error: %v\n%s", err, stdout.String())
	}
	if response["ok"] != false || response["operation"] != "link.open" {
		t.Fatalf("unexpected error envelope: %#v", response)
	}
	errObj := response["error"].(map[string]any)
	if errObj["code"] != "invalid_invite_url" {
		t.Fatalf("unexpected error object: %#v", errObj)
	}
}

func createPendingShareRequestViaCLI(t *testing.T, root string) string {
	t.Helper()
	setupPairedProfilesViaCLI(t, root)
	proposalPath := filepath.Join(root, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatalf("write proposal: %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := Run([]string{"request", "share", proposalPath, "--root", root, "--from", "esteban", "--to", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("request share failed: %s", stderr.String())
	}
	return strings.Fields(stdout.String())[3]
}

func setupPairedProfilesViaCLI(t *testing.T, root string) string {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"init", "--root", root, "--profile", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init denis failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"init", "--root", root, "--profile", "esteban"}, &stdout, &stderr); code != 0 {
		t.Fatalf("init esteban failed: %s", stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"invite", "create", "--root", root, "--from", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("invite failed: %s", stderr.String())
	}
	fields := strings.Fields(stdout.String())
	invitePath := fields[len(fields)-1]
	stdout.Reset()
	stderr.Reset()
	if code := Run([]string{"pair", "accept", invitePath, "--root", root, "--as", "esteban", "--handle", "denis"}, &stdout, &stderr); code != 0 {
		t.Fatalf("pair accept failed: %s", stderr.String())
	}
	return invitePath
}
