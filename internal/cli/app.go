package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"

	"pact/internal/pairing"
	"pact/internal/protocol"
	"pact/internal/requests"
	"pact/internal/store"
)

func runApp(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "open" {
		fmt.Fprintln(stderr, "Usage: pact app open <invite-url> --as <profile> [--addr 127.0.0.1:4327]")
		return 2
	}
	if len(args) < 2 || strings.HasPrefix(args[1], "-") {
		fmt.Fprintln(stderr, "Usage: pact app open <invite-url> --as <profile> [--addr 127.0.0.1:4327]")
		return 2
	}

	link := args[1]
	fs := flag.NewFlagSet("app open", flag.ContinueOnError)
	fs.SetOutput(stderr)
	root := fs.String("root", ".pact-local", "storage root")
	as := fs.String("as", "", "profile opening the local app")
	addr := fs.String("addr", "127.0.0.1:4327", "local app address")
	if err := fs.Parse(args[2:]); err != nil {
		return 2
	}
	if *as == "" {
		fmt.Fprintln(stderr, "--as is required")
		return 2
	}

	paths := store.NewPaths(*root)
	if err := ensureProfile(paths, *as); err != nil {
		fmt.Fprintf(stderr, "app open: %v\n", err)
		return 1
	}
	record, err := importLink(paths, *as, link)
	if err != nil {
		fmt.Fprintf(stderr, "app open: %v\n", err)
		return 1
	}

	appURL := "http://" + *addr + "/app"
	fmt.Fprintln(stdout, "Pact app is ready")
	fmt.Fprintf(stdout, "Imported request: %s\n", record.ID)
	fmt.Fprintf(stdout, "Open: %s\n", appURL)
	fmt.Fprintln(stdout, "Approve, counter, or reject from the browser.")

	server := &http.Server{
		Addr:    *addr,
		Handler: newAppHandler(paths, *as, link),
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "app open: %v\n", err)
		return 1
	}
	return 0
}

func ensureProfile(paths store.Paths, profile string) error {
	if _, err := pairing.LoadProfile(paths, profile); err == nil {
		return nil
	}
	_, err := pairing.InitProfile(paths, profile)
	return err
}

func importLink(paths store.Paths, profile string, link string) (protocol.RequestRecord, error) {
	payloadURL, err := payloadAPIURL(link)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	resp, err := http.Get(payloadURL)
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return protocol.RequestRecord{}, fmt.Errorf("relay returned %s", resp.Status)
	}
	tmp, err := os.CreateTemp("", "pact-app-link-*.json")
	if err != nil {
		return protocol.RequestRecord{}, err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		_ = tmp.Close()
		return protocol.RequestRecord{}, err
	}
	if err := tmp.Close(); err != nil {
		return protocol.RequestRecord{}, err
	}
	return requests.ImportSharePayload(paths, profile, tmp.Name())
}

func newAppHandler(paths store.Paths, profile string, sourceURL string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/app", http.StatusFound)
	})
	mux.HandleFunc("/app", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		renderApp(w, paths, profile, sourceURL, r.URL.Query().Get("flash"))
	})
	mux.HandleFunc("/decision", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/app?flash="+urlQuery("Could not read the decision."), http.StatusSeeOther)
			return
		}
		requestID := r.FormValue("request_id")
		action := r.FormValue("action")
		var err error
		switch action {
		case "approve":
			_, err = requests.Approve(paths, profile, requestID)
		case "reject":
			_, err = requests.Reject(paths, profile, requestID)
		case "counter":
			message := strings.TrimSpace(r.FormValue("message"))
			if message == "" {
				message = "Send a narrower version of this request."
			}
			_, err = requests.Counter(paths, profile, requestID, message)
		default:
			err = fmt.Errorf("unknown action: %s", action)
		}
		if err != nil {
			http.Redirect(w, r, "/app?flash="+urlQuery(err.Error()), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/app?flash="+urlQuery(decisionFlash(action)), http.StatusSeeOther)
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		records, err := appRecords(paths, profile)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"profile":    profile,
			"source_url": sourceURL,
			"requests":   records,
		})
	})
	return mux
}

func renderApp(w http.ResponseWriter, paths store.Paths, profile string, sourceURL string, flash string) {
	records, err := appRecords(paths, profile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, appHTMLStart(profile, sourceURL, flash, len(records)))
	if len(records) == 0 {
		fmt.Fprint(w, `<section class="empty"><h2>No Pact requests yet.</h2><p>Open a Pact invite and this inbox will become the approval layer for your agent.</p></section>`)
	}
	for _, record := range records {
		fmt.Fprint(w, requestCardHTML(profile, record))
	}
	fmt.Fprint(w, appHTMLEnd())
}

func appRecords(paths store.Paths, profile string) ([]protocol.RequestRecord, error) {
	records, err := store.ReadJSONArray[protocol.RequestRecord](paths.RequestsFile(profile))
	if err != nil {
		return nil, err
	}
	for i := range records {
		if records[i].Artifact.PendingPath != "" {
			records[i].Artifact.SourcePath = artifactPreview(records[i].Artifact.PendingPath)
		}
	}
	return records, nil
}

func appHTMLStart(profile string, sourceURL string, flash string, count int) string {
	flashHTML := ""
	if flash != "" {
		flashHTML = `<div class="flash">` + html.EscapeString(flash) + `</div>`
	}
	return `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Pact local approval app</title>
<style>` + appCSS() + `</style>
</head>
<body>
<main class="shell">
  <section class="hero">
    <div>
      <p class="eyebrow">Pact local app</p>
      <h1>Your agent caught a request.</h1>
      <p class="lede">Nothing happens until you approve it. Review the sender, artifact, preview, and consequence, then approve, counter, or reject.</p>
    </div>
    <aside class="status">
      <span>` + html.EscapeString(profile) + `</span>
      <strong>` + fmt.Sprintf("%d", count) + ` request(s)</strong>
      <small>` + html.EscapeString(sourceURL) + `</small>
    </aside>
  </section>
  ` + flashHTML + `
  <section class="grid">`
}

func requestCardHTML(profile string, record protocol.RequestRecord) string {
	preview := record.Artifact.SourcePath
	if preview == "" {
		preview = "No preview available."
	}
	stateClass := strings.ToLower(record.State)
	disabled := ""
	if record.State != protocol.StateDelivered {
		disabled = " disabled"
	}
	counterMessage := "Send a summary instead."
	return `<article class="card ` + html.EscapeString(stateClass) + `">
  <div class="card-top">
    <div>
      <p class="eyebrow">From ` + html.EscapeString(record.FromProfile) + `</p>
      <h2>` + html.EscapeString(record.Artifact.Name) + `</h2>
    </div>
    <span class="pill">` + html.EscapeString(statusLabel(record.State)) + `</span>
  </div>
  <p class="summary">` + html.EscapeString(record.FromProfile) + ` wants ` + html.EscapeString(profile) + `'s agent to receive this artifact. Approving copies it into your local received folder and writes an audit event.</p>
  <div class="meta">
    <span>Size <strong>` + html.EscapeString(humanBytes(record.Artifact.SizeBytes)) + `</strong></span>
    <span>Risk <strong>Low</strong></span>
    <span>Request <strong>` + html.EscapeString(record.ID) + `</strong></span>
  </div>
  <pre>` + html.EscapeString(preview) + `</pre>
  <form class="actions" method="post" action="/decision">
    <input type="hidden" name="request_id" value="` + html.EscapeString(record.ID) + `">
    <button class="approve" name="action" value="approve"` + disabled + `>Approve</button>
    <button class="reject" name="action" value="reject"` + disabled + `>Reject</button>
    <label>
      <span>Counteroffer</span>
      <input name="message" value="` + html.EscapeString(counterMessage) + `"` + disabled + `>
    </label>
    <button class="counter" name="action" value="counter"` + disabled + `>Counter</button>
  </form>
</article>`
}

func appHTMLEnd() string {
	return `</section>
  <section class="footer-note">
    <strong>Why this matters:</strong> Pact separates the message from permission. Your agent can inspect the request, but only your decision changes local state.
  </section>
</main>
</body>
</html>`
}

func appCSS() string {
	return `
:root {
  color-scheme: light;
  --ink: #17211b;
  --muted: #627061;
  --paper: #f6f0e3;
  --panel: #fffaf0;
  --line: #d8cbb6;
  --green: #235f3b;
  --red: #9d3f32;
  --gold: #c9822b;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  min-height: 100vh;
  color: var(--ink);
  background:
    radial-gradient(circle at 8% 12%, rgba(201,130,43,.22), transparent 28rem),
    linear-gradient(135deg, #efe3c8 0%, #f8f3e8 48%, #e4eedf 100%);
  font-family: ui-serif, Georgia, Cambria, "Times New Roman", serif;
}
.shell { width: min(1120px, calc(100% - 32px)); margin: 0 auto; padding: 48px 0; }
.hero { display: grid; grid-template-columns: 1fr minmax(240px, 340px); gap: 24px; align-items: end; margin-bottom: 24px; }
.eyebrow { margin: 0 0 10px; color: var(--green); font: 700 12px/1.1 ui-sans-serif, system-ui; letter-spacing: .16em; text-transform: uppercase; }
h1 { margin: 0; font-size: clamp(44px, 8vw, 92px); line-height: .86; letter-spacing: -.06em; max-width: 760px; }
.lede { max-width: 680px; color: var(--muted); font-size: 20px; line-height: 1.45; }
.status, .card, .footer-note, .empty, .flash {
  border: 1px solid var(--line);
  background: color-mix(in srgb, var(--panel) 88%, white);
  box-shadow: 0 24px 80px rgba(42, 36, 25, .12);
}
.status { padding: 20px; border-radius: 28px; display: grid; gap: 8px; }
.status span, .status small { color: var(--muted); overflow-wrap: anywhere; }
.status strong { font-size: 28px; }
.flash { margin: 0 0 18px; padding: 14px 18px; border-radius: 18px; color: var(--green); font-weight: 700; }
.grid { display: grid; gap: 20px; }
.card { border-radius: 34px; padding: clamp(20px, 4vw, 34px); }
.card-top { display: flex; justify-content: space-between; gap: 18px; align-items: start; }
h2 { margin: 0; font-size: clamp(30px, 5vw, 56px); line-height: .92; letter-spacing: -.045em; }
.pill { border: 1px solid var(--line); border-radius: 999px; padding: 8px 12px; color: var(--green); background: #eef5e9; font: 700 12px/1 ui-sans-serif, system-ui; text-transform: uppercase; letter-spacing: .08em; }
.approved .pill { color: var(--green); }
.denied .pill { color: var(--red); background: #f8e7df; }
.countered .pill { color: #7b4a13; background: #fff1d8; }
.summary { font-size: 18px; line-height: 1.5; color: var(--muted); max-width: 780px; }
.meta { display: flex; flex-wrap: wrap; gap: 10px; margin: 20px 0; }
.meta span { border: 1px solid var(--line); border-radius: 999px; padding: 8px 12px; background: #fbf5e8; color: var(--muted); font: 13px/1.1 ui-sans-serif, system-ui; }
.meta strong { color: var(--ink); }
pre { white-space: pre-wrap; overflow-wrap: anywhere; border-left: 5px solid var(--gold); background: #2b2419; color: #fff5df; padding: 18px; border-radius: 18px; font: 15px/1.45 ui-monospace, SFMono-Regular, Menlo, monospace; }
.actions { display: grid; grid-template-columns: auto auto 1fr auto; gap: 12px; align-items: end; }
button, input { font: 700 15px/1 ui-sans-serif, system-ui; border-radius: 16px; border: 1px solid var(--line); }
button { padding: 14px 18px; cursor: pointer; color: var(--ink); background: #fff8e9; }
button:disabled, input:disabled { opacity: .45; cursor: not-allowed; }
.approve { color: white; background: var(--green); border-color: var(--green); }
.reject { color: var(--red); }
.counter { color: #6d400f; }
label { display: grid; gap: 6px; color: var(--muted); font: 12px/1 ui-sans-serif, system-ui; text-transform: uppercase; letter-spacing: .08em; }
input { width: 100%; padding: 14px 16px; background: #fffdf6; color: var(--ink); }
.footer-note, .empty { margin-top: 22px; padding: 20px; border-radius: 24px; color: var(--muted); }
@media (max-width: 760px) {
  .hero { grid-template-columns: 1fr; }
  .actions { grid-template-columns: 1fr; }
  .card-top { display: grid; }
}`
}

func statusLabel(state string) string {
	switch state {
	case protocol.StateDelivered:
		return "waiting for you"
	case protocol.StateApproved:
		return "approved"
	case protocol.StateDenied:
		return "rejected"
	case protocol.StateCountered:
		return "countered"
	default:
		return state
	}
}

func decisionFlash(action string) string {
	switch action {
	case "approve":
		return "Approved. Pact copied the artifact locally and wrote the audit trail."
	case "reject":
		return "Rejected. Nothing was copied into your received folder."
	case "counter":
		return "Counter sent. The request is paused until the other side accepts the narrower terms."
	default:
		return "Decision saved."
	}
}

func urlQuery(value string) string {
	replacer := strings.NewReplacer(
		" ", "+",
		"!", "%21",
		"\"", "%22",
		"#", "%23",
		"%", "%25",
		"&", "%26",
		"'", "%27",
		"?", "%3F",
	)
	return replacer.Replace(value)
}
