# Pact Protocol

Pact is a local-first protocol and reference CLI for permissioned agent work.

The core idea:

> My agent can safely ask your agent to do work, and both of us can see, approve, counter, reject, and audit what happened.

Pact is not agent chat. It is a small permission layer for delegated work between people and their agents.

## What Works Today

This repository contains a dependency-light Go CLI that simulates two people, their local gateways, pairing, file-share requests, decisions, and audit logs on one machine.

The current V0 supports:

- creating local profiles
- creating a local pairing invite
- accepting an invite as a contact
- requesting to share a file
- listing a recipient inbox
- approving a request
- rejecting a request
- countering a request
- inspecting audit logs

## Quick Demo

Run:

```bash
go run ./cmd/pact demo
```

That creates local `esteban` and `denis` profiles, pairs Esteban with Denis, shares a demo proposal, approves it as Denis, and prints commands to inspect the inbox and audit logs.

## Manual Flow

```bash
go run ./cmd/pact init --profile denis
go run ./cmd/pact init --profile esteban
go run ./cmd/pact invite create --from denis
go run ./cmd/pact pair accept .pact-local/invites/<invite>.json --as esteban --handle denis
echo "# Proposal" > proposal.md
go run ./cmd/pact request share proposal.md --from esteban --to denis
go run ./cmd/pact inbox --as denis
go run ./cmd/pact approve <request-id> --as denis
go run ./cmd/pact audit --as denis
go run ./cmd/pact audit --as esteban
```

## Copy-Paste Payload Flow

For the first cross-agent experience, Pact can create a portable JSON payload that another person can import into their own local Pact inbox.

Sender:

```bash
go run ./cmd/pact init --profile esteban
echo "# Proposal" > proposal.md
go run ./cmd/pact payload create proposal.md --from esteban --to denis --out pact-payload.json
```

Send `pact-payload.json` to Denis or paste its JSON into Denis's agent.

Receiver:

```bash
go run ./cmd/pact init --profile denis
go run ./cmd/pact payload import pact-payload.json --as denis
go run ./cmd/pact approve <request-id> --as denis
```

Importing the payload renders an approval card with approve, counter, and reject commands. This is the no-relay V0 version of "Esteban's agent asked Denis's agent for permissioned work."

## First Encounter

Pact does not require a centralized directory.

A user should be able to tell their agent:

```text
Pact with Denis.
```

The agent can create a local invite that Denis can paste into his own agent. A future hosted service can make this smoother with shareable links, but the protocol should not require a central directory to start.

## Agent Integration

This repo includes instruction files for agent-first use:

- `AGENTS.md`: general coding-agent instructions
- `agents/codex/SKILL.md`: Codex-style skill instructions
- `agents/claude/CLAUDE.md`: Claude-style instructions
- `agents/mcp/README.md`: future MCP wrapper plan

The current integration is instruction-based: an agent reads the instructions, builds or runs the CLI, and uses Pact commands. A future MCP server or marketplace plugin should be a thin wrapper around the same core behavior.

## Safety Defaults

- Pairing creates a contact; it does not grant access.
- Requests must be approved, rejected, or countered.
- Secret-like files such as `.env` are refused by the CLI.
- Pact writes audit logs for meaningful actions.
- V0 is a local demo, not a hardened security boundary.

## Development

Run tests:

```bash
go test ./...
```

Build:

```bash
go build -o pact ./cmd/pact
```

Run help:

```bash
./pact help
```
