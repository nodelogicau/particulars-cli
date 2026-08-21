# Using particulars from Claude Desktop and other MCP clients

`particulars serve --mcp` speaks the Model Context Protocol over stdio. One server
is bound to one workspace; a second workspace is a second server entry in your
client. Tools use the DKF specification's names, and every result is exactly what
the corresponding CLI verb prints with `--json`.

## Connecting

### Claude Desktop — extension bundle

Download `particulars-<version>.mcpb` from the
[releases page](https://github.com/nodelogicau/particulars-cli/releases) and open
it; Claude Desktop asks for:

| Setting | What it is |
|---|---|
| Knowledge workspace | the folder containing `dkf.yaml` (create one with `particulars init <folder>`) |
| Your name | recorded as `source.author`; optional — the harness name is recorded regardless |

The bundle contains the server binary (macOS universal, Windows x64); nothing else
needs installing. The server starts when a conversation uses its tools.

### Claude Desktop — manual configuration

If you already have `particulars` installed:

```json
{
  "mcpServers": {
    "particulars": {
      "command": "/opt/homebrew/bin/particulars",
      "args": ["serve", "--mcp", "--workspace", "/Users/you/knowledge", "--author", "you"]
    }
  }
}
```

### Claude Code

`.mcp.json` in the repository. The server is spawned in the project, so it finds
the workspace the same way the CLI does — `dkf.yaml` in an ancestor, or a `.dkf`
pointer at the repo root — and no `--workspace` is needed:

```json
{
  "mcpServers": {
    "particulars": { "command": "particulars", "args": ["serve", "--mcp"] }
  }
}
```

### Cursor and others

Any client that launches stdio servers: command `particulars`, args
`serve --mcp [--workspace <dir>]`.

## What the model is told

At `initialize` the server sends `instructions`: which workspace it is bound to,
that writes land as files for a human to review, and the full particulars
discipline (the same text as the agent skill — recall before you assert, evidence
on every claim, honest `unresolved` on every synthesis). The same text is
available as the prompt `particulars-discipline` for clients that surface prompts.

## Tools

DKF specification tools (names and parameters as in the spec; `particular_id`
parameters accept an id, URI, label, or alias):

| Tool | Parameters | Returns (= CLI `--json`) |
|---|---|---|
| `particular_define` | `label`, `uri?`, `aliases[]?` | `{particular, created}` |
| `particular_resolve` | `query` | `{particular}` — `null` when nothing matches |
| `particular_merge` | `uri_a`, `uri_b`, `reason?`, `source?` | `{merge, sides, path}` |
| `claim_assert` | `particular_id`, `content`, `source?`, `context?`, `confidence?`, `scope?`, `timestamp?` | `{claim, path}` |
| `claim_retract` | `claim_id`, `reason`, `source?`, `superseded_by?` | `{id, type, retracted}` |
| `synthesis_create` | `particular_id`, `content`, `inputs[{id, role, weight?}]`, `unresolved`, `source?`, `method?`, `context?`, `confidence?`, `timestamp?` | `{synthesis, path, warnings}` |
| `knowledge_recall` | `particular_id?` (or `query?`), `topics[]?`, `scope?`, `include_retracted?`, `limit?` | `{entries, count, subject?, class?}` |
| `conflict_detect` | `particular_id` **or** `claim_ids[]` | `{reports, count}` / a report for the set |
| `lineage_trace` | `claim_id`, `depth?` | provenance tree |

particulars extensions (not part of the DKF tool set):

| Tool | Returns |
|---|---|
| `topics_list` | `{topics: [{topic, assertions, particulars}], count}` |
| `workspace_status` | root, id, base-uri, counts, `validate` summary, conflict reports, and — inside a git checkout — `git.uncommitted` file paths (read-only) |

`source` is `{author?, harness?, model?, document?}`; `context` is `{scope?, topics[]?}`.

Errors are tool results with `isError: true`, a text line `<code>: <message>`, and
`{"error": {"code", "message"}}` using the CLI's codes (`not_found`, `usage`,
`invalid`, `conflict`, `runtime`).

## Provenance

`source.harness` defaults to the connected client's name from the MCP handshake
(`claude-ai`, `claude-code`, `cursor`, …) when neither the call, the server flags,
`DKF_HARNESS`, nor `dkf.yaml` supplies one. `source.author` comes from `--author`,
`DKF_AUTHOR`, or `dkf.yaml`. The usual minimum applies: a source needs author or
harness; a synthesis needs harness — so an agent-only client is always valid.

## Reviewing what a Desktop conversation wrote

The server never runs git. Files accumulate in the workspace folder; if that
folder is a git checkout, `workspace_status` lists what is uncommitted so the
assistant can tell you "seven claims await review". Committing, branching, and
opening the pull request remain yours — see
[review-workflow.md](review-workflow.md). A hosted, database-backed server that
reviews without a terminal would be a separate implementation sharing this tool
contract; it is not part of this CLI.
