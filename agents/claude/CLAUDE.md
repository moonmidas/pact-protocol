# Pact Claude Integration

Use Pact when the user asks to install Pact, accept a Pact invite, "pact with" someone, send a file through Pact, or approve/reject/counter a Pact request.

Pact is implemented as a local Go CLI. The current integration is instruction-based; an MCP server can be added later.

## Setup

From the Pact repository root:

```bash
go test ./...
go build -o pact ./cmd/pact
./pact help
```

For a quick local verification:

```bash
go run ./cmd/pact demo
```

## Command Map

- Create profile: `./pact init --profile <profile>`
- Create invite: `./pact invite create --from <profile>`
- Accept invite: `./pact pair accept <invite-file> --as <profile> --handle <handle>`
- Request file share: `./pact request share <path> --from <profile> --to <handle>`
- List inbox: `./pact inbox --as <profile>`
- Approve: `./pact approve <request-id> --as <profile>`
- Reject: `./pact reject <request-id> --as <profile>`
- Counter: `./pact counter <request-id> --as <profile> --message "<message>"`
- Audit: `./pact audit --as <profile>`

## Safety

Before using Pact to send or approve anything, explain the action and the scope to the user. Do not bypass Pact's CLI for Pact workflows. Do not share secret-like files.

