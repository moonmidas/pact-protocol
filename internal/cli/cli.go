package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pact/internal/audit"
	"pact/internal/pairing"
	"pact/internal/protocol"
	"pact/internal/relay"
	"pact/internal/requests"
	"pact/internal/store"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "init":
		return runInit(args[1:], stdout, stderr)
	case "invite":
		return runInvite(args[1:], stdout, stderr)
	case "pair":
		return runPair(args[1:], stdout, stderr)
	case "request":
		return runRequest(args[1:], stdout, stderr)
	case "inbox":
		return runInbox(args[1:], stdout, stderr)
	case "approve":
		return runDecision("approve", args[1:], stdout, stderr)
	case "reject":
		return runDecision("reject", args[1:], stdout, stderr)
	case "counter":
		return runDecision("counter", args[1:], stdout, stderr)
	case "audit":
		return runAudit(args[1:], stdout, stderr)
	case "demo":
		return runDemo(args[1:], stdout, stderr)
	case "payload":
		return runPayload(args[1:], stdout, stderr)
	case "relay":
		return runRelay(args[1:], stdout, stderr)
	case "link":
		return runLink(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: pact <command> [options]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  init")
	fmt.Fprintln(w, "  invite create")
	fmt.Fprintln(w, "  pair accept")
	fmt.Fprintln(w, "  request share")
	fmt.Fprintln(w, "  inbox")
	fmt.Fprintln(w, "  approve")
	fmt.Fprintln(w, "  reject")
	fmt.Fprintln(w, "  counter")
	fmt.Fprintln(w, "  audit")
	fmt.Fprintln(w, "  demo")
	fmt.Fprintln(w, "  payload create")
	fmt.Fprintln(w, "  payload import")
	fmt.Fprintln(w, "  relay serve")
	fmt.Fprintln(w, "  link create")
	fmt.Fprintln(w, "  link open")
}

func runInit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	profile := fs.String("profile", "", "profile name")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *profile == "" {
		fmt.Fprintln(stderr, "--profile is required")
		return 2
	}
	if _, err := pairing.InitProfile(store.NewPaths(*root), *profile); err != nil {
		fmt.Fprintf(stderr, "init: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "initialized profile %s\n", *profile)
	return 0
}

func runInvite(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "create" {
		fmt.Fprintln(stderr, "Usage: pact invite create --from <profile>")
		return 2
	}
	fs := flag.NewFlagSet("invite create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	from := fs.String("from", "", "profile creating invite")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *from == "" {
		fmt.Fprintln(stderr, "--from is required")
		return 2
	}
	invitePath, _, err := pairing.CreateInvite(store.NewPaths(*root), *from)
	if err != nil {
		fmt.Fprintf(stderr, "invite create: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created invite %s\n", invitePath)
	return 0
}

func runPair(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 || args[0] != "accept" {
		fmt.Fprintln(stderr, "Usage: pact pair accept <invite-path> --as <profile> --handle <handle>")
		return 2
	}
	invitePath := args[1]
	fs := flag.NewFlagSet("pair accept", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile accepting invite")
	handle := fs.String("handle", "", "contact handle")
	if err := fs.Parse(args[2:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	contact, err := pairing.AcceptInvite(store.NewPaths(*root), *as, *handle, invitePath)
	if err != nil {
		fmt.Fprintf(stderr, "pair accept: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "paired %s with %s\n", *as, contact.Handle)
	return 0
}

func runRequest(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 || args[0] != "share" {
		fmt.Fprintln(stderr, "Usage: pact request share <path> --from <profile> --to <handle>")
		return 2
	}
	sharePath := args[1]
	fs := flag.NewFlagSet("request share", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	from := fs.String("from", "", "profile sending request")
	to := fs.String("to", "", "contact handle")
	if err := fs.Parse(args[2:]); err != nil {
		return 2
	}
	if *from == "" || *to == "" {
		fmt.Fprintln(stderr, "--from and --to are required")
		return 2
	}
	record, err := requests.ShareFile(store.NewPaths(*root), *from, *to, sharePath)
	if err != nil {
		fmt.Fprintf(stderr, "request share: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created share request %s for %s\n", record.ID, *to)
	return 0
}

func runInbox(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("inbox", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	messages, err := requests.ListInbox(store.NewPaths(*root), *as)
	if err != nil {
		fmt.Fprintf(stderr, "inbox: %v\n", err)
		return 1
	}
	if len(messages) == 0 {
		fmt.Fprintln(stdout, "inbox empty")
		return 0
	}
	for _, msg := range messages {
		renderInboxCard(stdout, *as, msg)
	}
	return 0
}

func runDecision(command string, args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "Usage: pact %s <request-id> --as <profile>\n", command)
		return 2
	}
	requestID := args[0]
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile")
	message := fs.String("message", "", "counter message")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}

	var (
		record protocol.RequestRecord
		err    error
	)
	switch command {
	case "approve":
		record, err = requests.Approve(store.NewPaths(*root), *as, requestID)
	case "reject":
		record, err = requests.Reject(store.NewPaths(*root), *as, requestID)
	case "counter":
		if strings.TrimSpace(*message) == "" {
			fmt.Fprintln(stderr, "--message is required")
			return 2
		}
		record, err = requests.Counter(store.NewPaths(*root), *as, requestID, *message)
	}
	if err != nil {
		fmt.Fprintf(stderr, "%s: %v\n", command, err)
		return 1
	}
	verb := map[string]string{"approve": "approved", "reject": "rejected", "counter": "countered"}[command]
	fmt.Fprintf(stdout, "%s request %s\n", verb, record.ID)
	return 0
}

func runAudit(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	events, err := audit.Read(store.NewPaths(*root).AuditFile(*as))
	if err != nil {
		fmt.Fprintf(stderr, "audit: %v\n", err)
		return 1
	}
	for _, event := range events {
		fmt.Fprintf(stdout, "%s %s %s\n", event.Timestamp.Format("2006-01-02T15:04:05Z"), event.EventType, event.Summary)
	}
	return 0
}

func artifactName(body json.RawMessage) string {
	var record protocol.RequestRecord
	if err := json.Unmarshal(body, &record); err == nil && record.Artifact.Name != "" {
		return record.Artifact.Name
	}
	return filepath.Base(string(body))
}

func renderInboxCard(stdout io.Writer, profile string, msg protocol.Envelope) {
	var record protocol.RequestRecord
	_ = json.Unmarshal(msg.Body, &record)
	name := record.Artifact.Name
	if name == "" {
		name = "unknown artifact"
	}
	size := humanBytes(record.Artifact.SizeBytes)
	risk := "Low"
	if strings.Contains(strings.ToLower(name), "secret") {
		risk = "Needs review"
	}

	fmt.Fprintln(stdout, "┌─ Pact Request ─────────────────────────────────────")
	fmt.Fprintf(stdout, "│ From:    %s\n", msg.From.Profile)
	fmt.Fprintf(stdout, "│ To:      %s\n", profile)
	fmt.Fprintf(stdout, "│ Wants:   Share %s\n", name)
	fmt.Fprintf(stdout, "│ Size:    %s\n", size)
	fmt.Fprintf(stdout, "│ Risk:    %s\n", risk)
	fmt.Fprintf(stdout, "│ Request: %s\n", msg.CorrelationID)
	fmt.Fprintln(stdout, "│")
	fmt.Fprintf(stdout, "│ Approve: pact approve %s --as %s\n", msg.CorrelationID, profile)
	fmt.Fprintf(stdout, "│ Counter: pact counter %s --as %s --message \"Send a summary instead.\"\n", msg.CorrelationID, profile)
	fmt.Fprintf(stdout, "│ Reject:  pact reject %s --as %s\n", msg.CorrelationID, profile)
	fmt.Fprintln(stdout, "└────────────────────────────────────────────────────")
}

func humanBytes(size int64) string {
	if size < 0 {
		return "unknown"
	}
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
}

func runDemo(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	paths := store.NewPaths(*root)
	if _, err := pairing.InitProfile(paths, "denis"); err != nil {
		fmt.Fprintf(stderr, "demo init denis: %v\n", err)
		return 1
	}
	if _, err := pairing.InitProfile(paths, "esteban"); err != nil {
		fmt.Fprintf(stderr, "demo init esteban: %v\n", err)
		return 1
	}
	invitePath, _, err := pairing.CreateInvite(paths, "denis")
	if err != nil {
		fmt.Fprintf(stderr, "demo invite: %v\n", err)
		return 1
	}
	if _, err := pairing.AcceptInvite(paths, "esteban", "denis", invitePath); err != nil {
		fmt.Fprintf(stderr, "demo pair: %v\n", err)
		return 1
	}

	proposalPath := filepath.Join(*root, "proposal.md")
	if err := os.WriteFile(proposalPath, []byte("# Proposal\n\nThis is a Pact V0 demo artifact.\n"), 0o644); err != nil {
		fmt.Fprintf(stderr, "demo proposal: %v\n", err)
		return 1
	}
	record, err := requests.ShareFile(paths, "esteban", "denis", proposalPath)
	if err != nil {
		fmt.Fprintf(stderr, "demo share: %v\n", err)
		return 1
	}
	if _, err := requests.Approve(paths, "denis", record.ID); err != nil {
		fmt.Fprintf(stderr, "demo approve: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "Pact demo complete")
	fmt.Fprintf(stdout, "root: %s\n", *root)
	fmt.Fprintf(stdout, "invite: %s\n", invitePath)
	fmt.Fprintf(stdout, "request: %s\n", record.ID)
	fmt.Fprintf(stdout, "received: %s\n", filepath.Join(paths.ReceivedArtifactsDir("denis"), record.Artifact.ID, record.Artifact.Name))
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Try:")
	fmt.Fprintf(stdout, "  pact inbox --root %s --as denis\n", *root)
	fmt.Fprintf(stdout, "  pact audit --root %s --as denis\n", *root)
	fmt.Fprintf(stdout, "  pact audit --root %s --as esteban\n", *root)
	return 0
}

func runPayload(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: pact payload <create|import>")
		return 2
	}
	switch args[0] {
	case "create":
		return runPayloadCreate(args[1:], stdout, stderr)
	case "import":
		return runPayloadImport(args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "Usage: pact payload <create|import>")
		return 2
	}
}

func runPayloadCreate(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: pact payload create <path> --from <profile> --to <name> --out <file>")
		return 2
	}
	sharePath := args[0]
	fs := flag.NewFlagSet("payload create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	from := fs.String("from", "", "profile creating payload")
	to := fs.String("to", "", "recipient name")
	out := fs.String("out", "", "payload output file")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *from == "" || *to == "" {
		fmt.Fprintln(stderr, "--from and --to are required")
		return 2
	}
	payload, err := requests.ExportSharePayload(store.NewPaths(*root), *from, *to, sharePath)
	if err != nil {
		fmt.Fprintf(stderr, "payload create: %v\n", err)
		return 1
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "payload create: %v\n", err)
		return 1
	}
	if *out != "" {
		if err := os.WriteFile(*out, append(data, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "payload create: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "created Pact payload %s\n", *out)
		fmt.Fprintln(stdout, "Send this file or paste its JSON into the recipient's agent.")
		return 0
	}
	fmt.Fprintln(stdout, "-----BEGIN PACT PAYLOAD-----")
	fmt.Fprintln(stdout, string(data))
	fmt.Fprintln(stdout, "-----END PACT PAYLOAD-----")
	return 0
}

func runPayloadImport(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: pact payload import <payload-file> --as <profile>")
		return 2
	}
	payloadPath := args[0]
	fs := flag.NewFlagSet("payload import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile importing payload")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	record, err := requests.ImportSharePayload(store.NewPaths(*root), *as, payloadPath)
	if err != nil {
		fmt.Fprintf(stderr, "payload import: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "imported Pact request %s\n\n", record.ID)
	messages, err := requests.ListInbox(store.NewPaths(*root), *as)
	if err != nil {
		fmt.Fprintf(stderr, "payload import inbox: %v\n", err)
		return 1
	}
	for _, msg := range messages {
		if msg.CorrelationID == record.ID {
			renderInboxCard(stdout, *as, msg)
		}
	}
	return 0
}

func runRelay(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "serve" {
		fmt.Fprintln(stderr, "Usage: pact relay serve --addr :4319 --storage <dir> --base-url <url>")
		return 2
	}
	fs := flag.NewFlagSet("relay serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", ":4319", "listen address")
	storage := fs.String("storage", ".pact-relay", "relay storage directory")
	baseURL := fs.String("base-url", "http://localhost:4319", "public base URL")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	fmt.Fprintf(stdout, "Pact relay listening on %s\n", *addr)
	if err := http.ListenAndServe(*addr, relay.New(*storage, *baseURL).Handler()); err != nil {
		fmt.Fprintf(stderr, "relay serve: %v\n", err)
		return 1
	}
	return 0
}

func runLink(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: pact link <create|open>")
		return 2
	}
	switch args[0] {
	case "create":
		return runLinkCreate(args[1:], stdout, stderr)
	case "open":
		return runLinkOpen(args[1:], stdout, stderr)
	default:
		fmt.Fprintln(stderr, "Usage: pact link <create|open>")
		return 2
	}
}

func runLinkCreate(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: pact link create <path> --from <profile> --to <name> --relay <url>")
		return 2
	}
	sharePath := args[0]
	fs := flag.NewFlagSet("link create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	from := fs.String("from", "", "profile creating link")
	to := fs.String("to", "", "recipient name")
	relayURL := fs.String("relay", "https://wepact.online", "relay base URL")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *from == "" || *to == "" {
		fmt.Fprintln(stderr, "--from and --to are required")
		return 2
	}
	payload, err := requests.ExportSharePayload(store.NewPaths(*root), *from, *to, sharePath)
	if err != nil {
		fmt.Fprintf(stderr, "link create: %v\n", err)
		return 1
	}
	data, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintf(stderr, "link create: %v\n", err)
		return 1
	}
	resp, err := http.Post(strings.TrimRight(*relayURL, "/")+"/api/payloads", "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(stderr, "link create: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		fmt.Fprintf(stderr, "link create: relay returned %s\n", resp.Status)
		return 1
	}
	var created relay.CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		fmt.Fprintf(stderr, "link create: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "created Pact link %s\n", created.URL)
	fmt.Fprintln(stdout, "Send this link to the recipient or paste it into their agent.")
	return 0
}

func runLinkOpen(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 1 {
		fmt.Fprintln(stderr, "Usage: pact link open <url> --as <profile>")
		return 2
	}
	link := args[0]
	fs := flag.NewFlagSet("link open", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile opening link")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}
	payloadURL := strings.Replace(link, "/i/", "/api/payloads/", 1)
	resp, err := http.Get(payloadURL)
	if err != nil {
		fmt.Fprintf(stderr, "link open: %v\n", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(stderr, "link open: relay returned %s\n", resp.Status)
		return 1
	}
	tmp, err := os.CreateTemp("", "pact-link-*.json")
	if err != nil {
		fmt.Fprintf(stderr, "link open: %v\n", err)
		return 1
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		fmt.Fprintf(stderr, "link open: %v\n", err)
		return 1
	}
	if err := tmp.Close(); err != nil {
		fmt.Fprintf(stderr, "link open: %v\n", err)
		return 1
	}
	record, err := requests.ImportSharePayload(store.NewPaths(*root), *as, tmp.Name())
	if err != nil {
		fmt.Fprintf(stderr, "link open: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "opened Pact request %s\n\n", record.ID)
	messages, err := requests.ListInbox(store.NewPaths(*root), *as)
	if err != nil {
		fmt.Fprintf(stderr, "link open inbox: %v\n", err)
		return 1
	}
	for _, msg := range messages {
		if msg.CorrelationID == record.ID {
			renderInboxCard(stdout, *as, msg)
		}
	}
	return 0
}
