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

### Crush

In a `crushrc` (Crush's Bash-based config). Crush spawns the server in the
project directory with its own environment inherited, so — as with Claude Code —
discovery finds the workspace and nothing more is needed:

```bash
mcp add particulars --type stdio --command particulars --args serve --args --mcp
```

Do not add `--env DKF_WORKSPACE "$PWD"`: `$PWD` is the project, and through
v0.10.0 an explicit path had to be the workspace root itself, so a project that
reaches its workspace through a `.dkf` pointer made the server exit before the
handshake — Crush reported it as `calling "initialize": … EOF`
([#6](https://github.com/nodelogicau/particulars-cli/issues/6)). Explicit paths
now follow a pointer, but the line remains redundant.

### Cursor and others

Any client that launches stdio servers: command `particulars`, args
`serve --mcp [--workspace <dir>]`.

`--workspace` and `$DKF_WORKSPACE` accept either the workspace root or a
directory holding a `.dkf` pointer to it, so a host may pass its project
directory. A server that exits before answering `initialize` (clients report
`EOF` or "connection closed") almost always could not resolve a workspace; run
the same command by hand from the same directory to see the message.

## What the model is told

### Workspace conventions

The generic discipline is the skill's; a workspace's own conventions — its
topic vocabulary, local naming, what goes in which scope — are the
workspace's. Put them in `dkf.md` at the workspace root — the prose sibling of
`dkf.yaml`, as the [DKF specification](https://github.com/nodelogicau/particulars#dkfmd)
names it — or name another file with `workspace.conventions: TOPICS.md` in
`dkf.yaml`, and the server appends them to the `initialize` instructions under
a heading naming the file. At least the first 16 KiB is always delivered, cut
only on a character boundary, with a note naming the file when longer. This
is how conventions reach clients that never read the repository — Claude
Desktop above all. For clients that never surface `instructions` either, the
same document is listed as an MCP resource (`file://…/dkf.md`, `text/markdown`),
read from disk on each request, so an edit after startup is visible there while
the instructions stay a startup snapshot.

`particulars workspace` shows which file applies. A configured file that is
missing is a stderr warning, never a failure; so is a `workspace.conventions`
that is absolute or escapes the workspace — the key is then treated as unset
and reported as `conventions_invalid`, because a workspace must open under
every tool. The name is DKF-specific on purpose: a generic file such as
`CONVENTIONS.md` or `AGENTS.md` can already exist at a repository root for
another tool and would be delivered without anyone having asked. `AGENTS.md`
is nevertheless a good *value* for the key when the workspace directory is its
own agent scope (`knowledge/AGENTS.md`): harnesses that read the repository
then pick it up with no DKF support at all. A `CONVENTIONS.md` left from
v0.12.0 is not read; rename it or name it in the key. Keep it short: it rides
in every session's context.

A workspace that ingests feeds or documents should state its ingestion
register here — *extract the facts; the feed itself is never a subject* —
because that is exactly the discipline that varies most between models, and
this file is the one place it reaches every client.


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
| `claim_assert` | `particular_id`, `content`, `evidential`, `source?`, `context?`, `confidence?`, `scope?`, `timestamp?` | `{claim, path}` |
| `claim_retract` | `claim_id`, `reason`, `source?`, `superseded_by?` | `{id, type, retracted}` |
| `synthesis_create` | `particular_id`, `content`, `inputs[{id, role, weight?}]`, `unresolved`, `source?`, `method?`, `context?`, `confidence?`, `timestamp?` | `{synthesis, path, warnings}` |
| `knowledge_recall` | `particular_id?` (or `query?`), `author?`, `topics[]?`, `scope?`, `include_retracted?`, `limit?` | `{entries, count, subject?, class?, author?}`; author-filtered entries carry `relations` (`asserted`/`reported`) |
| `conflict_detect` | `particular_id` **or** `claim_ids[]` | `{reports, count}` / a report for the set |
| `lineage_trace` | `claim_id`, `depth?` | provenance tree |

particulars extensions (not part of the DKF tool set):

| Tool | Returns |
|---|---|
| `topics_list` | `{topics: [{topic, assertions, particulars}], count}` |
| `unresolved_list` | `{entries: [{particular, label, uri, members?, synthesis, timestamp, unresolved, unsynthesised}], count}` — each current synthesis's `unresolved`, oldest first; `particular_id?`, `scope?`, `include_none?` |
| `workspace_status` | root, id, base-uri, counts, `validate` summary, conflict reports, and — inside a git checkout — `git.uncommitted` file paths (read-only) |

`source` is `{author?, harness?, model?, document?}`; `context` is `{scope?, topics[]?}`.
`author` is a particular reference — id, URI, or name; a defined particular is
written as its `uri`. `document` is a string, or a mapping of `ref` (`uri` is
the legacy alias), `author` (who produced what was read — for testimony, who
told you), `hash`, and `quote`.

Errors are tool results with `isError: true`, a text line `<code>: <message>`, and
`{"error": {"code", "message"}}` using the CLI's codes (`not_found`, `usage`,
`invalid`, `conflict`, `runtime`).

## Provenance

`source.harness` defaults to the connected client's name from the MCP handshake
(`claude-ai`, `claude-code`, `cursor`, …) when neither the call, the server flags,
`DKF_HARNESS`, nor `dkf.yaml` supplies one. `source.author` comes from `--author`,
`DKF_AUTHOR`, or `dkf.yaml`. The usual minimum applies: a source needs author or
harness; a synthesis needs harness — so an agent-only client is always valid.

**Leaving "Your name" blank in the Desktop extension is safe.** Claude Desktop
substitutes `${user_config.author}` only when the field has a value; left empty,
the literal text reaches the server. Since v0.5.2 an unsubstituted `${…}` in any
provenance value — flag, `DKF_*`, or config — is treated as absent, so it falls
through to `dkf.yaml` defaults rather than being recorded as the author. Older
servers write the placeholder into `source.author` verbatim; fill the field in,
or upgrade the extension. The same applies to the workspace path, where an
unsubstituted value is reported as such instead of "no dkf.yaml in ${…}".

## Reviewing what a Desktop conversation wrote

The server never runs git. Files accumulate in the workspace folder; if that
folder is a git checkout, `workspace_status` lists what is uncommitted so the
assistant can tell you "seven claims await review". Committing, branching, and
opening the pull request remain yours — see
[review-workflow.md](review-workflow.md). A hosted, database-backed server that
reviews without a terminal would be a separate implementation sharing this tool
contract; it is not part of this CLI.
