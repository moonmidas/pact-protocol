# Pact MCP Plan

Pact does not have an MCP server yet. The current integration path is:

1. Agent reads `AGENTS.md` or the Codex/Claude integration files.
2. Agent builds or runs the Go CLI.
3. Agent calls Pact commands directly.

Future MCP tools should wrap the same CLI/service operations:

- `pact_init_profile`
- `pact_create_invite`
- `pact_accept_invite`
- `pact_request_share`
- `pact_list_inbox`
- `pact_approve_request`
- `pact_reject_request`
- `pact_counter_request`
- `pact_read_audit`

The MCP server should stay thin. Pact protocol state and audit logs should continue to be produced by the Pact core implementation.

