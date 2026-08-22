## Context

The core (`internal/store`, `internal/query`, `internal/dkf`, the embedded skill) has no transport knowledge; `internal/cli` is its only front-end. The DKF spec defines an MCP tool surface (README "MCP Server Tools", resolved 2026-08-21: `synthesis_create(content, inputs[], unresolved, source)`, `knowledge_recall` "operates across merged particulars", `lineage_trace` "including superseded-by successors"). The official Go SDK (`github.com/modelcontextprotocol/go-sdk/mcp`) is v1.7.0 and stable: `NewServer(impl, &ServerOptions{Instructions: …})`, generic `AddTool[In, Out]` inferring JSON schemas from structs, `StdioTransport`, `InMemoryTransport` for tests, `req.Session.InitializeParams().ClientInfo`, `ToolAnnotations`. MCPB manifests (`manifest_version "0.3"`) support `server.type: "binary"`, `platform_overrides`, `user_config` with a `directory` type, and `${user_config.key}` / `${__dirname}` substitution.

Scope decided in exploration: local stdio server, one server per workspace, hosted connector out of scope (probably a different implementation behind the same tool contract).

## Goals / Non-Goals

**Goals:**
- Same core, same results: an MCP tool returns exactly what the CLI's `--json` returns.
- Spec-exact tool names and parameters, so clients and the spec's examples transfer, and a hosted implementation can be swapped in.
- Discipline without a file: the skill reaches the model through `instructions`.
- Double-click install on Claude Desktop; zero-config in Claude Code and Cursor.

**Non-Goals:**
- HTTP transport, auth, multi-tenancy; multi-workspace addressing; federation tools; MCP resources; git writes; sampling/elicitation.

## Decisions

### D1. `serve --mcp` is a verb on the same binary

One install path (installer, cask, bundle) and one version for CLI and server. `serve` without `--mcp` is a usage error reserved for future transports. The server opens its workspace once at startup using `store.DiscoverWith` (flag → env → discovery from cwd) and refuses to start without one (exit 5), printing the usual message to stderr — stdout is the protocol channel and is never written to outside JSON-RPC.

### D2. One server, one workspace

The MCP convention, and the user's call. A client that wants two workspaces configures two server entries (`particulars serve --mcp --workspace <a>` / `<b>`); clients already render that. Adding an optional `workspace` parameter later is additive and was designed in exploration (registry from flags, names from `dkf.yaml`, read-only fan-out for resolve/recall); it is not built now.

### D3. Tool surface and names

Spec tools, spec names, spec parameter names:

| Tool | Maps to | Annotations |
|---|---|---|
| `particular_define(uri?, label, aliases[])` | `store.UpsertParticular` + minting | idempotent |
| `particular_resolve(query)` → particular or `null` | `query.Resolve` (single match; ambiguity is an error listing ids) | read-only |
| `particular_merge(uri_a, uri_b, source?, reason?)` | `store.CreateMerge` | — |
| `claim_assert(particular_id, content, source?, context?, confidence?, scope?)` | `store.Create` | — |
| `claim_retract(claim_id, reason, source?, superseded_by?)` | `store.Retract` (claims, syntheses, merges) | — |
| `synthesis_create(particular_id, content, inputs[], unresolved, source?, method?, context?, confidence?, timestamp?)` | `store.Create` | — |
| `knowledge_recall(particular_id? \| query?, scope?, topics[]?, include_retracted?, limit?)` | `query.Recall` over the merge class | read-only |
| `conflict_detect(particular_id? \| claim_ids[]?)` | `query.Analyse`; claim-set form below | read-only |
| `lineage_trace(claim_id, depth?)` | `query.Lineage` | read-only |
| `topics_list()` *(ours)* | `query.Topics` | read-only |
| `workspace_status()` *(ours)* | `store`/`query.Validate`/`Conflicts` + `git status --porcelain` when `.git` exists | read-only |

`particular_id` parameters accept what the CLI accepts (id, uri, label, alias) and resolve the same way; the name stays `particular_id` because that is the spec's. `synthesis_create` gains `particular_id` (the spec's signature omits the subject, which is an upstream omission — raised as SPEC-FEEDBACK item 13). `source` objects are `{author?, harness?, model?, document?}`; `context` is `{scope?, topics[]?}`.

**`conflict_detect(claim_ids[])`:** treat the given set as the universe: `current` = most recent non-retracted synthesis *in the set*, `unsynthesised` = members not in its closure, `stale` = member syntheses citing (transitively) a retracted object. Same code path as `Analyse` with the candidate list substituted. It answers "have these particular claims been reconciled?" without depending on a particular.

### D4. Results are the CLI's JSON shapes

Each handler returns the same Go value the CLI serialises for `--json` (e.g. `{claim, path}`, `{entries, count, class}`, the `Report`), so `StructuredContent` is byte-compatible with the CLI and the documentation is shared. The text content block is a compact one-line summary (id and path, or counts) for clients that show only text. Output schemas are inferred from the same structs.

### D5. Errors

Domain conditions → `CallToolResult{IsError: true}` with text `"<code>: <message>"` and structured `{error: {code, message}}`, using the CLI's codes (`not_found`, `usage`, `invalid`, `conflict`, `no_workspace`, `runtime`). `particular_resolve` with no match is **not** an error: it returns `{"particular": null}` per spec. Protocol faults (bad JSON, unknown tool) are left to the SDK. The `classify` function in `internal/cli` moves to a shared place so both front-ends map identically.

### D6. Provenance from the handshake

At `initialize`, `clientInfo.name` (e.g. `claude-ai`, `claude-code`, `cursor`) becomes the default `source.harness` for the session; `clientInfo.version` is recorded nowhere (it is the client's version, not the model). Precedence per call: the call's `source` fields → `--author`/`--harness`/`--model` server flags → `DKF_*` env → `dkf.yaml` defaults → `clientInfo.name` for harness. The same `requireProvenance` rule applies (author or harness; syntheses need harness) — with the handshake default, an agent-only client always satisfies it.

### D7. Instructions and prompt

`ServerOptions.Instructions` = `skill.Body()` with the SKILL.md frontmatter stripped, prefixed by two lines: which workspace this server is bound to (root, id) and that writes land as files for a human to review. The same text is exposed as prompt `particulars-discipline` (no arguments) for clients that surface prompts but not instructions. Tool descriptions carry the per-tool rules in one sentence each (recall before assert; `unresolved` required, `None identified` when nothing; retract never deletes; merges are undone by retracting).

### D8. Concurrency

Handlers run concurrently in the SDK. Every mutating tool takes a server-wide mutex around load → write → index upsert; read tools load without the lock (files are immutable; worst case a read sees a half-applied index and falls back to scanning, which `store` already tolerates).

### D9. The bundle

```
particulars-<ver>.mcpb  (zip)
├── manifest.json
├── icon.png
└── server/
    ├── darwin-arm64/particulars
    ├── darwin-amd64/particulars
    ├── win32-x64/particulars.exe
    └── win32-arm64/particulars.exe
```

```json
{
  "manifest_version": "0.3",
  "name": "particulars",
  "display_name": "particulars — dialectical knowledge",
  "version": "<ver>",
  "description": "Capture and recall knowledge as claims and syntheses in a DKF workspace",
  "author": {"name": "NodeLogic"},
  "server": {
    "type": "binary",
    "entry_point": "server/darwin-arm64/particulars",
    "mcp_config": {
      "command": "${__dirname}/server/darwin-arm64/particulars",
      "args": ["serve", "--mcp", "--workspace", "${user_config.workspace}", "--author", "${user_config.author}"],
      "platform_overrides": {
        "darwin": {"command": "${__dirname}/server/${arch}/particulars"},
        "win32":  {"command": "${__dirname}/server/win32-x64/particulars.exe"}
      }
    }
  },
  "user_config": {
    "workspace": {"type": "directory", "title": "Knowledge workspace", "description": "Folder containing dkf.yaml (create one with `particulars init`)", "required": true},
    "author":    {"type": "string", "title": "Your name (source.author)", "required": false}
  },
  "compatibility": {"platforms": ["darwin", "win32"]}
}
```

Exact override keys (per-arch on darwin, whether `${arch}` exists) are verified against the MCPB spec during implementation; if per-arch overrides are not expressible, ship a darwin universal binary (`lipo`) instead — GoReleaser can produce one. `--author` with an empty value is treated as unset. The bundle is assembled by a GoReleaser `publishers`/post-hook step from the already-built binaries and attached to the release; `make bundle` builds it locally; CI runs `mcpb validate` (or a JSON-schema check of the manifest) when the tool is available.

### D10. Testing

`internal/mcp` tests connect a client over `InMemoryTransport` to a server bound to a temp workspace and call tools exactly as a harness would: the nine spec tools (happy paths and each error), `particular_resolve` null, claim-set `conflict_detect`, `instructions` content, harness default from `clientInfo`, concurrency (parallel asserts leave a clean index), and the bundle manifest (valid JSON, version matches, every referenced binary present after `make bundle`).

## Risks / Trade-offs

- [SDK API churn] → it is v1; pin the minor version; the handler layer is thin.
- [Schema inference produces awkward schemas for `particular_id | query` unions] → model as two optional fields with a handler-side "exactly one" check; document in the description.
- [Stdout discipline] → any stray print breaks the protocol; the server redirects the CLI's text emitters to stderr and a test asserts stdout carries only JSON-RPC.
- [Binary size × 4 in the bundle] → ~16 MB; acceptable for a desktop extension. Universal darwin binary halves it if overrides are awkward.
- [Desktop users cannot review] → `workspace_status` makes the unreviewed state visible; the README says Desktop capture is for people who also commit from a terminal. The hosted implementation is the real answer and is out of scope here by decision.
- [Windows path/quarantine quirks in the bundle] → covered by the existing Windows CI matrix for the binary; bundle install is hand-verified on a Mac before tagging; Windows bundle install is verified opportunistically.

## Migration Plan

Additive; the CLI is unchanged. Release v0.4.0; the bundle is a release asset from then on.

## Open Questions

- `synthesis_create` lacks a subject in the spec's signature — SPEC-FEEDBACK item 13; we add `particular_id`.
- Whether `workspace_status` should also expose `validate` *findings* (potentially many) or only counts — counts plus the first N, probably.
- MCP resources (`dkf://claims/<id>`) — deferred until a client wants to attach a claim to context directly.
