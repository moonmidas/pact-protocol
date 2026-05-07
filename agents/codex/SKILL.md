---
name: pact
description: Use Pact to create or accept pairing invites, request permissioned file shares, approve/reject/counter requests, and inspect audit logs through the local Pact CLI.
---

# Pact Skill

Use this skill when the user says "Pact", "pact with Denis", "send this through Pact", "accept this Pact invite", or asks for permissioned agent-to-agent collaboration.

## Behavior

Pact is CLI-backed. Do not invent protocol outcomes in text. Use the local `pact` binary if present, or `go run ./cmd/pact` from the Pact repository.

Prefer these commands:

```bash
go test ./...
go build -o pact ./cmd/pact
./pact help
./pact demo
./pact init --profile <profile>
./pact initiate <name> --as <profile>
./pact accept <start-link> --as <profile>
./pact invite create --from <profile>
./pact pair accept <invite-file> --as <profile> --handle <handle>
./pact request share <path> --from <profile> --to <handle>
./pact payload create <path> --from <profile> --to <name> --out pact-payload.json
./pact payload import <payload-file> --as <profile>
./pact app open <pact-link> --as <profile>
./pact inbox --as <profile>
./pact approve <request-id> --as <profile>
./pact reject <request-id> --as <profile>
./pact counter <request-id> --as <profile> --message "<message>"
./pact audit --as <profile>
```

## Safety

- Pairing creates a local contact, not permission.
- Do not approve or share files silently.
- Do not share secret-like files.
- Use Pact commands so JSON messages and audit logs are written.

## First Encounter

If a requested contact is not paired, create one shareable start link and ask the user to send it to the other person:

```bash
./pact initiate <name> --as <profile>
```

The other person can paste the `https://wepact.online/start/<id>` link into their own agent, which should install Pact from the repository if needed and run `pact accept <start-link> --as <profile>`.

## Cross-Agent Payload

If the other person is not paired in the same local store, prefer a portable payload:

```bash
./pact payload create <path> --from <profile> --to <name> --out pact-payload.json
```

The recipient's agent imports it:

```bash
./pact payload import pact-payload.json --as <profile>
```

Import renders an approval card with exact approve, counter, and reject commands.

## Hosted Link App Flow

When the user pastes a hosted Pact link such as `https://wepact.online/i/<id>`, prefer the app flow:

```bash
./pact app open https://wepact.online/i/<id> --as <profile>
```

Show or open the printed local app URL. The local app imports the request and gives the user approve, counter, and reject controls in the browser while preserving Pact audit logs.
