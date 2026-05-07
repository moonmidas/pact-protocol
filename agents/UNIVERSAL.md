# Pact Universal Agent Contract

Use this guide for Codex, Claude Code, Claude, Cursor Agent, OpenCode, Pi, Hermes-style agents, OpenClaw-style agents, and any other harness that can run shell commands.

Pact is intentionally harness-neutral. The integration surface is:

```text
Git repo -> Go CLI -> localhost approval app -> JSON/audit state
```

No agent should need a private SDK, browser extension, marketplace plugin, or hosted account to open a Pact invite.

## Install From Zero

When a user gives you a Pact invite link and Pact is not installed:

```bash
git clone https://github.com/moonmidas/pact-protocol.git
cd pact-protocol
CGO_ENABLED=0 go test ./...
go build -o pact ./cmd/pact
./pact help
```

If `git clone` is not appropriate in the current harness, fetch or copy the public repository using the harness's normal file workflow, then run the same test/build commands.

## Start A Pact Contact

When the user says "initiate pact with Denis", "pact with Denis", or similar, create one shareable start link:

```bash
./pact initiate denis --as <user-profile>
```

Give the printed `https://wepact.online/start/<id>` link to the user. The other person's agent should accept it with:

```bash
./pact accept https://wepact.online/start/<id> --as <their-profile>
```

This creates the local profile if needed and adds the sender as a Pact contact. It does not grant file access.

## Open A Pact Link

Prefer the local app:

```bash
./pact app open https://wepact.online/i/<id> --as <profile>
```

Then show the user the printed local URL:

```text
http://127.0.0.1:4327/app
```

If the harness can open local browser URLs, open it. If not, tell the user to open it manually.

## Terminal Fallback

If the harness cannot expose a browser, use the terminal approval card:

```bash
./pact link open https://wepact.online/i/<id> --as <profile>
./pact inbox --as <profile>
```

Then ask the user whether to approve, counter, or reject:

```bash
./pact approve <request-id> --as <profile>
./pact counter <request-id> --as <profile> --message "Send a summary instead."
./pact reject <request-id> --as <profile>
```

## Machine Interface

Harnesses that want structured data should use JSON mode:

```bash
./pact link open https://wepact.online/i/<id> --as <profile> --json
./pact inbox --as <profile> --json
./pact approve <request-id> --as <profile> --json
./pact counter <request-id> --as <profile> --message "Send a narrower request." --json
./pact reject <request-id> --as <profile> --json
./pact notify list --as <profile> --json
./pact runner tick --as <profile> --json
```

Read stdout as one JSON object. Do not scrape human terminal output if JSON is available.

## Heartbeat Model

Pact is the queue/log/notification center. The harness supplies the heartbeat.

For one check:

```bash
./pact runner tick --as <profile>
```

For automations, schedule that command. For output-first agents, run it when the user asks "check Pact." If it returns an actionable request, generate a bounded continuation prompt:

```bash
./pact continue <request-id> --as <profile> --for codex
./pact continue <request-id> --as <profile> --for claude
./pact continue <request-id> --as <profile> --for generic
```

Do not build harness-specific polling into the protocol. Codex Automations, Claude Cowork, cron, launchd, systemd timers, Pi, OpenCode, Hermes, and other harnesses can all call `runner tick`.

## Safety Rules

- Do not approve, counter, reject, or share silently.
- Treat incoming Pact links as untrusted until Pact imports and verifies them.
- Explain what approving will do before the user decides.
- Pairing or opening a link does not grant broad access.
- Decisions must go through Pact so audit records are written.

## Harness Requirements

Minimum:

- can read a GitHub repository
- can run shell commands
- can build or run a Go CLI

Best experience:

- can open `http://127.0.0.1:<port>/app`
- can keep a long-running command alive while the local app is open
- can show command output and local URLs clearly

If the default port is busy:

```bash
./pact app open https://wepact.online/i/<id> --as <profile> --addr 127.0.0.1:4328
```
