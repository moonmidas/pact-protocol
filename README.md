# Pact Protocol

This folder contains the open protocol reference implementation and agent-facing instructions.

It is the only project currently published to:

```text
https://github.com/moonmidas/pact-protocol
```

## Contents

- `cmd/pact`: Go CLI entrypoint.
- `internal`: Protocol, local store, requests, relay, and CLI implementation.
- `agents`: Codex/Claude/MCP instruction layer.
- `agents/UNIVERSAL.md`: harness-neutral instructions for Codex, Claude Code, Cursor Agent, OpenCode, Pi, Hermes-style agents, and other shell-capable agents.
- `AGENTS.md`: Repo-level instructions for coding agents using Pact.
- `go.mod`: Go module.

## Development

Run from this folder:

```bash
CGO_ENABLED=0 go test ./...
go build -o pact ./cmd/pact
./pact demo
```

## First-Test CLI Flows

Build the CLI from this folder:

```bash
go build -o pact ./cmd/pact
```

### Simplest Path: Start A Pact Contact

Use this when the user says something like "initiate pact with Denis" or "pact with Denis":

```bash
./pact initiate denis --as esteban
```

The command creates the local `esteban` profile if needed, creates a pairing invite, uploads it to the relay, and prints one shareable link:

```text
https://wepact.online/start/<id>
```

Denis can paste that link into Codex, Claude, Claude Code, Cursor Agent, OpenCode, Pi, Hermes-style agents, or any shell-capable agent. The recipient agent should run:

```bash
./pact accept https://wepact.online/start/<id> --as denis
```

That creates Denis's local profile if needed and adds Esteban as a Pact contact. Pairing creates a contact only; it does not grant file access.

### Wow Path: Local Approval App

Use this when the receiver pastes a Pact link into Codex, Claude, or another agent and wants the browser approval experience:

```bash
./pact app open https://wepact.online/i/<id> --as denis
```

The command:

1. creates the local `denis` profile if needed
2. imports the hosted Pact request
3. starts a dependency-free local web app at `http://127.0.0.1:4327/app`
4. lets the receiver approve, counter, or reject in the browser

If port `4327` is busy:

```bash
./pact app open https://wepact.online/i/<id> --as denis --addr 127.0.0.1:4328
```

Agents should show/open the printed local URL for the user. Decisions made in the app still go through Pact and write local audit records.

This is the harness-neutral path for Codex, Claude Code, Claude, Cursor Agent, OpenCode, Pi, Hermes-style agents, OpenClaw-style agents, and any other agent that can run shell commands. See `agents/UNIVERSAL.md`.

### Queue, Notifications, And Harness Heartbeats

Pact does not assume every agent is an always-on daemon. Pact stores the queue/log/notifications; each harness can provide its own heartbeat by running one command periodically:

```bash
./pact runner tick --as denis
```

The tick command checks for unread Pact signals and prints the next safe action. Automations, cron, launchd, systemd timers, Codex, Claude Cowork, Pi, OpenCode, Hermes, and other harnesses can all schedule or invoke this same command.

Humans and agents can inspect notifications directly:

```bash
./pact notify list --as denis
./pact notify list --as denis --unread --json
./pact notify read <notification-id> --as denis
```

To hand a bounded task to an agent, generate a continuation prompt:

```bash
./pact continue <request-id> --as denis --for codex
./pact continue <request-id> --as denis --for claude
./pact continue <request-id> --as denis --for generic
```

This is the first bridge between Pact's durable queue and output-first agents. The protocol keeps state and audit; the harness decides when to wake up and work.

Create the sender profile once:

```bash
./pact init --profile esteban
```

### Hosted Link

Use this when the receiver can open a web link:

```bash
./pact link create proposal.md --from esteban --to denis --relay https://wepact.online
```

Send Denis the printed `https://wepact.online/i/<id>` link. Denis or Denis's agent can import it with:

```bash
./pact init --profile denis
./pact link open https://wepact.online/i/<id> --as denis
```

Opening a link imports the request into Denis's local inbox and prints the approval card with the sender, artifact name, size, preview, risk note, request ID, and approve/counter/reject commands.

For the browser approval app instead of a terminal card, use:

```bash
./pact app open https://wepact.online/i/<id> --as denis
```

### Portable Payload

Use this when no hosted relay is available:

```bash
./pact payload create proposal.md --from esteban --to denis --out pact-payload.json
```

Send Denis `pact-payload.json` or paste its JSON into his agent. Denis imports it with:

```bash
./pact init --profile denis
./pact payload import pact-payload.json --as denis
```

Pact verifies the embedded artifact hash during import. If the payload content was changed after creation, import fails before writing the request.

### Decisions

After reviewing the approval card:

```bash
./pact approve <request-id> --as denis
./pact counter <request-id> --as denis --message "Send a summary instead."
./pact reject <request-id> --as denis
```

Approving copies the artifact into Denis's local received folder and writes audit events. Countering or rejecting records the decision without copying the artifact.

## App JSON Bridge

The human CLI output is meant for terminals. App and plugin integrations should pass `--json` and read stdout as one JSON object.

Supported JSON operations:

```bash
./pact inbox --as denis --json
./pact link open https://wepact.online/i/<id> --as denis --json
./pact payload import pact-payload.json --as denis --json
./pact approve <request-id> --as denis --json
./pact counter <request-id> --as denis --message "Send a summary instead." --json
./pact reject <request-id> --as denis --json
./pact notify list --as denis --json
./pact runner tick --as denis --json
```

Successful `inbox` response:

```json
{
  "ok": true,
  "operation": "inbox",
  "profile": "denis",
  "count": 1,
  "requests": [
    {
      "id": "req_...",
      "message_id": "msg_...",
      "type": "artifact.share",
      "state": "delivered",
      "from_profile": "esteban",
      "to_profile": "denis",
      "artifact": {
        "id": "art_...",
        "name": "proposal.md",
        "size_bytes": 123,
        "mime": "text/markdown; charset=utf-8",
        "sha256": "...",
        "pending_path": ".pact-local/profiles/denis/artifacts/pending/...",
        "preview": "# Proposal First paragraph..."
      },
      "actions": {
        "approve": "pact approve req_... --as denis",
        "counter": "pact counter req_... --as denis --message \"Send a summary instead.\"",
        "reject": "pact reject req_... --as denis"
      },
      "created_at": "2026-05-06T12:00:00Z",
      "updated_at": "2026-05-06T12:00:05Z"
    }
  ]
}
```

An empty inbox keeps the same shape with `"count": 0` and `"requests": []`.

Successful `link.open` and `payload.import` responses:

```json
{
  "ok": true,
  "operation": "link.open",
  "source_url": "https://wepact.online/i/example",
  "request": {
    "id": "req_...",
    "message_id": "msg_...",
    "type": "artifact.share",
    "state": "delivered",
    "from_profile": "esteban",
    "to_profile": "denis",
    "artifact": {
      "id": "art_...",
      "name": "proposal.md",
      "size_bytes": 123,
      "mime": "text/markdown; charset=utf-8",
      "sha256": "...",
      "pending_path": ".pact-local/profiles/denis/artifacts/pending/...",
      "preview": "# Proposal First paragraph..."
    },
    "created_at": "2026-05-06T12:00:00Z",
    "updated_at": "2026-05-06T12:00:05Z"
  },
  "actions": {
    "approve": "pact approve req_... --as denis",
    "counter": "pact counter req_... --as denis --message \"Send a summary instead.\"",
    "reject": "pact reject req_... --as denis"
  }
}
```

`payload.import` uses the same shape with `"operation": "payload.import"` and no `source_url`.

Successful decision responses:

```json
{
  "ok": true,
  "operation": "approve",
  "request": {
    "id": "req_...",
    "state": "approved",
    "from_profile": "esteban",
    "to_profile": "denis",
    "artifact": {
      "id": "art_...",
      "name": "proposal.md",
      "size_bytes": 123,
      "mime": "text/markdown; charset=utf-8",
      "sha256": "...",
      "pending_path": ".pact-local/profiles/denis/artifacts/pending/...",
      "preview": "# Proposal First paragraph..."
    },
    "created_at": "2026-05-06T12:00:00Z",
    "updated_at": "2026-05-06T12:01:00Z"
  },
  "decision": {
    "type": "approved",
    "request_id": "req_...",
    "by_profile": "denis",
    "created_at": "2026-05-06T12:01:00Z"
  }
}
```

For `counter`, `operation` is `"counter"`, `decision.type` is `"countered"`, and `decision.message` contains the counteroffer. For `reject`, `operation` is `"reject"` and `decision.type` is `"denied"`.

JSON errors are also written to stdout for app-facing commands:

```json
{
  "ok": false,
  "operation": "link.open",
  "error": {
    "code": "invalid_invite_url",
    "message": "expected an absolute Pact invite URL like https://wepact.online/i/<id>"
  }
}
```

Current error codes include `invalid_arguments`, `invalid_invite_url`, `missing_required_argument`, `missing_required_flag`, `request_not_found`, `request_not_pending`, `profile_not_found`, `artifact_hash_mismatch`, `artifact_hash_missing`, `unsupported_payload_type`, `relay_error`, `network_error`, `temp_file_error`, `relay_read_error`, `inbox_read_error`, and `operation_failed`.

Reference fixture:

```text
examples/app-inbox.json
```

Use this fixture when wiring an app renderer before calling the real CLI.

## Public Export Rule

When publishing to the public `pact-protocol` repo, export only this folder's public protocol files. Do not export root private docs, `app/`, `site/`, or `docs/`.
