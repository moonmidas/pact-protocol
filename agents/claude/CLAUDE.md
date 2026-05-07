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
- Start contact link: `./pact initiate <name> --as <profile>`
- Accept start link: `./pact accept <start-link> --as <profile>`
- Create invite: `./pact invite create --from <profile>`
- Accept invite: `./pact pair accept <invite-file> --as <profile> --handle <handle>`
- Request file share: `./pact request share <path> --from <profile> --to <handle>`
- Create portable payload: `./pact payload create <path> --from <profile> --to <name> --out pact-payload.json`
- Import portable payload: `./pact payload import <payload-file> --as <profile>`
- Open hosted link in local app: `./pact app open <pact-link> --as <profile>`
- List inbox: `./pact inbox --as <profile>`
- Approve: `./pact approve <request-id> --as <profile>`
- Reject: `./pact reject <request-id> --as <profile>`
- Counter: `./pact counter <request-id> --as <profile> --message "<message>"`
- Audit: `./pact audit --as <profile>`

## Safety

Before using Pact to send or approve anything, explain the action and the scope to the user. Do not bypass Pact's CLI for Pact workflows. Do not share secret-like files.

## First Encounter

When the user says "initiate pact with Denis" or "pact with Denis", create one shareable start link:

```bash
./pact initiate denis --as esteban
```

The other person's agent accepts the link:

```bash
./pact accept https://wepact.online/start/<id> --as denis
```

Pairing adds a local contact. It does not grant broad access.

## Cross-Agent Payload Flow

When the sender and recipient are on different machines and there is no relay yet, use a portable payload:

```bash
./pact payload create proposal.md --from esteban --to denis --out pact-payload.json
```

The recipient's agent can import it:

```bash
./pact payload import pact-payload.json --as denis
```

After import, show the request card and ask whether to approve, counter, or reject.

## Hosted Link App Flow

When the user pastes a hosted Pact link, prefer the local approval app:

```bash
./pact app open https://wepact.online/i/<id> --as denis
```

Then show or open the printed local URL. The app lets the human approve, counter, or reject in the browser and writes Pact audit records through the CLI engine.
