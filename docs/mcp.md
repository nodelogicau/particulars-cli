# Using particulars from Claude Desktop and other MCP clients

`particulars serve --mcp` speaks the Model Context Protocol over stdio. One server
is bound to one workspace; a second workspace is a second server entry in your
client. Tools use the DKF specification's names, and every result is exactly what
the corresponding CLI verb prints with `--json`.

## Which should I use — the MCP server or the skill and CLI?

Both reach the same core and write the same files, so this is not a lock-in
decision. It comes down to whether your harness already has a shell.

| | Always in context | When knowledge work happens |
|---|---|---|
| MCP server | ~4,900 tokens (11 tool schemas ≈ 2,600, instructions ≈ 2,300) | the same |
| Skill + CLI | ~86 tokens (the skill's frontmatter description) | ~2,300 tokens (the body loads when triggered) |

**Use the skill and the CLI in harnesses that have a shell** — Claude Code,
Cursor's agent, Codex:

- The tool definitions and instructions are loaded in *every* session, whether or
  not it touches knowledge; the skill stays collapsed to its description until
  something triggers it.
- The shell composes: `particulars recall X --json | jq …`, a heredoc into
  `--content-file -`, a loop over ids, `conflicts --json` piped into a check.
  MCP tools are atomic calls — no piping, no chaining.
- The CLI resolves the workspace on **every invocation**, so it follows you
  across worktrees, branch switches, and repositories within one session. The
  server binds its workspace root and `dkf.yaml` config once at startup (object
  files are still read fresh per call).
- Review is shell work anyway: branch, assert, commit, open the pull request.

**Microsoft 365 Copilot cannot use either of these.** It runs in Microsoft's cloud
with no access to your filesystem, so a local stdio server is unreachable; its own
MCP route (federated connectors) needs a remote HTTPS server and is read-only
regardless. Publish to it instead with `particulars export --format graph` — see
[graph.md](graph.md).

**Use the MCP server where there is no shell**, or where you would rather not
grant one: Claude Desktop chat, mobile, and similar clients. There, typed
arguments earn their cost — the model cannot mistype a flag, and a required
field like `unresolved` is enforced by the schema rather than by the model
remembering. It is also the right choice if you want knowledge writes confined
to an explicit allowlist of operations, since the CLI is reachable through any
shell command.

**Avoid running both in the same harness.** Two surfaces for the same operations
duplicate the context cost and make it a coin flip which one the model reaches
for. Mixing across harnesses is expected, though: Claude Desktop through the
bundle and Claude Code through the CLI, against one git repository — every claim
records the harness that made it.

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

**Enable the extension after installing it.** Claude Desktop installs extensions
switched off; until you toggle `particulars` on in Settings → Extensions, no
particulars tools appear and no server is launched — so there is nothing in
`~/Library/Logs/Claude/` to explain the absence either. If tools are still
missing once it is enabled, `~/Library/Logs/Claude/mcp-server-particulars.log`
will exist and say why.

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
