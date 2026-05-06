package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
