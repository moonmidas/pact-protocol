package relay

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	StorageDir string
	BaseURL    string
}

type CreateResponse struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func New(storageDir string, baseURL string) Server {
	if storageDir == "" {
		storageDir = ".pact-relay"
	}
	return Server{
		StorageDir: storageDir,
		BaseURL:    strings.TrimRight(baseURL, "/"),
	}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/payloads", s.handlePayloads)
	mux.HandleFunc("/api/payloads/", s.handlePayloadByID)
	mux.HandleFunc("/api/start", s.handleStart)
	mux.HandleFunc("/api/start/", s.handleStartByID)
	mux.HandleFunc("/i/", s.handleInvitePage)
	mux.HandleFunc("/start/", s.handleStartPage)
	return mux
}

func (s Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, landingHTML())
}

func (s Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) handlePayloads(w http.ResponseWriter, r *http.Request) {
	s.handleCreate(w, r, "payloads", s.publicPayloadURL)
}

func (s Server) handleStart(w http.ResponseWriter, r *http.Request) {
	s.handleCreate(w, r, "start", s.publicStartURL)
}

func (s Server) handleCreate(w http.ResponseWriter, r *http.Request, folder string, publicURL func(string) string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var payload json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}
	id, err := newRelayID()
	if err != nil {
		http.Error(w, "could not create id", http.StatusInternalServerError)
		return
	}
	if err := os.MkdirAll(filepath.Join(s.StorageDir, folder), 0o755); err != nil {
		http.Error(w, "could not create storage", http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(s.itemPath(folder, id), append(payload, '\n'), 0o600); err != nil {
		http.Error(w, "could not store payload", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, CreateResponse{ID: id, URL: publicURL(id)})
}

func (s Server) handlePayloadByID(w http.ResponseWriter, r *http.Request) {
	s.handleGet(w, r, "payloads", strings.TrimPrefix(r.URL.Path, "/api/payloads/"))
}

func (s Server) handleStartByID(w http.ResponseWriter, r *http.Request) {
	s.handleGet(w, r, "start", strings.TrimPrefix(r.URL.Path, "/api/start/"))
}

func (s Server) handleGet(w http.ResponseWriter, r *http.Request, folder string, id string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !validRelayID(id) {
		http.NotFound(w, r)
		return
	}
	data, err := os.ReadFile(s.itemPath(folder, id))
	if os.IsNotExist(err) && folder == "payloads" {
		data, err = os.ReadFile(s.legacyPayloadPath(id))
	}
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not read payload", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(data)
}

func (s Server) handleInvitePage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/i/")
	if !validRelayID(id) {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(s.itemPath("payloads", id)); os.IsNotExist(err) {
		if _, legacyErr := os.Stat(s.legacyPayloadPath(id)); os.IsNotExist(legacyErr) {
			http.NotFound(w, r)
			return
		}
	} else if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, inviteHTML(), id, s.publicPayloadURL(id), id)
}

func (s Server) handleStartPage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/start/")
	if !validRelayID(id) {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(s.itemPath("start", id)); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, startHTML(), id, s.publicStartURL(id), id)
}

func (s Server) itemPath(folder string, id string) string {
	return filepath.Join(s.StorageDir, folder, id+".json")
}

func (s Server) legacyPayloadPath(id string) string {
	return filepath.Join(s.StorageDir, id+".json")
}

func (s Server) publicPayloadURL(id string) string {
	if s.BaseURL == "" {
		return "/i/" + id
	}
	return s.BaseURL + "/i/" + id
}

func (s Server) publicStartURL(id string) string {
	if s.BaseURL == "" {
		return "/start/" + id
	}
	return s.BaseURL + "/start/" + id
}

func newRelayID() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func validRelayID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func landingHTML() string {
	return `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Pact</title><style>
body{font-family:ui-sans-serif,system-ui;margin:48px;max-width:760px;line-height:1.5;color:#101418;background:#f7f4ee}
.card{background:white;border:1px solid #e3ddd2;border-radius:18px;padding:28px;box-shadow:0 20px 60px rgba(32,24,12,.08)}
code{background:#f0ece4;padding:2px 6px;border-radius:6px}
a{color:#8b3f16}
</style></head>
<body><div class="card">
<h1>Pact</h1>
<p>Pact is a permission layer for agent collaboration.</p>
<p>If someone sent you a Pact link, paste it into Codex or Claude and ask it to open the request.</p>
<p><a href="https://github.com/moonmidas/pact-protocol">Install from GitHub</a></p>
</div></body></html>`
}

func inviteHTML() string {
	return `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Pact Request</title><style>
body{font-family:ui-sans-serif,system-ui;margin:48px;max-width:820px;line-height:1.5;color:#101418;background:#f7f4ee}
.card{background:white;border:1px solid #e3ddd2;border-radius:18px;padding:28px;box-shadow:0 20px 60px rgba(32,24,12,.08)}
code,pre{background:#f0ece4;padding:2px 6px;border-radius:6px}
pre{padding:16px;overflow:auto}
a{color:#8b3f16}
</style></head>
<body><div class="card">
<h1>Pact Request</h1>
<p>You received a Pact request.</p>
<p>Open this link with your agent:</p>
<pre>Open this Pact request: %s</pre>
<p>If Pact is installed, the agent should run:</p>
<pre>pact app open %s --as &lt;your-name&gt;</pre>
<p>Payload API:</p>
<pre>/api/payloads/%s</pre>
</div></body></html>`
}

func startHTML() string {
	return `<!doctype html>
<html>
<head><meta charset="utf-8"><title>Start a Pact</title><style>
body{font-family:ui-sans-serif,system-ui;margin:48px;max-width:820px;line-height:1.5;color:#101418;background:#f7f4ee}
.card{background:white;border:1px solid #e3ddd2;border-radius:18px;padding:28px;box-shadow:0 20px 60px rgba(32,24,12,.08)}
code,pre{background:#f0ece4;padding:2px 6px;border-radius:6px}
pre{padding:16px;overflow:auto}
a{color:#8b3f16}
</style></head>
<body><div class="card">
<h1>Start a Pact</h1>
<p>Someone wants to start a Pact contact with you.</p>
<p>Paste this into your agent:</p>
<pre>Open this Pact start link: %s</pre>
<p>If Pact is installed, the agent should run:</p>
<pre>pact accept %s --as &lt;your-name&gt;</pre>
<p>Start API:</p>
<pre>/api/start/%s</pre>
</div></body></html>`
}
