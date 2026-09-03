## Why

Every synthesis is required to say what it could not settle, and nothing reads that field back. `unresolved` is the format's honest list of open questions, but the only way to see it is `recall <particular> --json`, one particular at a time, picking out the entry marked `current`. There is no workspace-wide `recall`, and `conflicts` reports structure only — it says which claims are unreconciled, not what the current belief admits it does not know. In the dogfood workspace 29 of 37 syntheses are superseded, so their `unresolved` text is history ("v0.2.0 is not yet tagged"); the seven current ones are the live backlog, and today nobody can list them without a script.

## What Changes

- **A new `particulars unresolved [<particular>] [--include-none] [--scope <s>]` verb** that lists, for every merge equivalence class with a non-retracted synthesis, the current synthesis's `unresolved` text — oldest current synthesis first, so the longest-neglected question surfaces at the top. `current` is exactly what `conflicts` and `recall` already compute; the verb reuses that code rather than defining a second notion.
- **The conventional empty value is filtered.** Entries whose `unresolved` is exactly `None identified` are omitted unless `--include-none` is given, so the default list contains only real open questions.
- **Each entry carries the structural context** a reader needs to judge whether the question may already have an answer: the `unsynthesised` count for the class (same definition as `conflicts`), alongside particular id, label, uri, `members` when the class has more than one, synthesis id and timestamp.
- **An empty result is success**, exit 0 with an empty list — a workspace with nothing open is the goal, not a lookup failure.
- **MCP parity:** an `unresolved_list` tool, read-only, labelled a particulars extension like `topics_list` and `workspace_status`, whose structured result equals the verb's `--json` output.
- **The skill names the verb** in its review flow beside `conflicts` ("what needs work" is two lists: structural debt and admitted open questions), so the installed skill copy is regenerated.

Nothing is **BREAKING**; no object, config, or existing verb changes.

## Capabilities

### New Capabilities
- `unresolved-listing`: the `unresolved` verb — selection of current syntheses per class, the `None identified` filter, ordering, entry shape, and exit behaviour.

### Modified Capabilities
- `mcp-server`: the extension tool set gains `unresolved_list`, described and annotated like `topics_list`.

The skill's verb table is not pinned by any spec; its update is a task.

## Impact

- `internal/query/` (a new `Unresolved` function beside `conflicts.go`, reusing `CurrentForClass`, `Closure`, and the class enumeration in `Conflicts`), `internal/cli/cmd_query.go` (the verb and its text output), `internal/mcp/tools.go` (registration and handler), `skills/particulars/SKILL.md` and its installed copy, `README.md` verb table, `docs/mcp.md` tool table, `docs/review-workflow.md` (the CI summary step can print it next to `conflicts`).
- Dogfood: read-only; the workspace gains nothing but a view. Its seven current syntheses become the first real output, and the DKF class — one open question plus one unsynthesised claim — is the case the `unsynthesised` count exists for.
