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
