## Why

A workspace's conventions — its topic vocabulary, tagging rules, local naming — belong with the workspace, not in the generic skill every workspace shares. A repo-level document reaches harnesses that read the repo, but an MCP-only client (Claude Desktop with the `.mcpb` bundle) sees nothing except the `initialize` instructions, so today a workspace has no way to teach those clients its conventions.

## What Changes

- A workspace MAY carry a conventions document: `CONVENTIONS.md` at the root by default, or another file named by `workspace.conventions` in `dkf.yaml` (a relative path inside the workspace).
- `serve --mcp` appends the document to the `initialize` instructions (and therefore to the `particulars-discipline` prompt), under a heading naming the file, capped at 16 KiB with a truncation note. A configured file that is missing is a stderr diagnostic, never a failure.
- `particulars workspace` reports the resolved conventions file; `--json` gains `conventions` (and `conventions_missing: true` when configured but absent).

Nothing is **BREAKING**: workspaces without the file behave exactly as today.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `workspace`: the conventions file — default name, `dkf.yaml` key with its validity rule, and the `workspace` verb reporting it.
- `mcp-server`: instructions and prompt carry the workspace conventions after the skill body.

## Impact

- `internal/store` (config key + validation, `Workspace.Conventions()`), `internal/mcp` (instructions), `internal/cli` (`workspace` verb, `serve` diagnostic), `docs/mcp.md`, README, skill (the topics bullet points at the conventions file).
- Dogfood: the news workspace's `TOPICS.md` becomes deliverable to Claude Desktop via `workspace.conventions: TOPICS.md`.
