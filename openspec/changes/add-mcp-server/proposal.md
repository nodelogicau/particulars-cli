## Why

Claude Desktop — and every other MCP client — has no way to drive `particulars`: its integration model is MCP, not a shell. The DKF spec's own reference interface is an MCP server with eleven named tools, and this CLI was built with a transport-agnostic core precisely so that server could be a second front-end rather than a rewrite. Shipping it closes the Desktop gap, makes the spec's tool contract concrete (so a future database-backed hosted implementation can be a drop-in for clients), and delivers the agent discipline through MCP `instructions` instead of a file on disk.

## What Changes

- New verb `particulars serve --mcp` — a stdio MCP server over one workspace (the MCP convention: one server per workspace; a second workspace is a second server entry in the client). Workspace from `--workspace`, `DKF_WORKSPACE`, or discovery (`dkf.yaml` / `.dkf`) from the server's cwd.
- Nine spec tools with spec-exact names and parameters: `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect` (both the `particular_id` and `claim_ids[]` forms), `lineage_trace`. Results are the CLI's `--json` shapes as structured content. `particular_resolve` returns `null` on no match, per spec.
- Two read-only tools of ours, clearly labelled as extensions: `topics_list` and `workspace_status` (root, id, base-uri, counts, `validate` summary, open conflicts, files not yet committed if the workspace is a git checkout — read-only, no git writes).
- `instructions` in the `initialize` response = the embedded skill body (plus a note that the server is bound to one workspace); a `prompts/list` entry carrying the same text for clients that prefer prompts.
- `source.harness` defaulted from the client's `clientInfo.name` at the handshake; `author` from `dkf.yaml` defaults or `--author`; callers may override per call.
- Domain failures (not found, usage, validation) as `isError` tool results carrying the CLI's error codes; JSON-RPC errors only for protocol faults. Tool annotations: read-only for queries, idempotent for `particular_define`, non-destructive everywhere (nothing deletes).
- A `.mcpb` bundle for Claude Desktop containing the darwin (arm64, amd64) and windows (amd64, arm64) binaries with `platform_overrides`, and a `user_config` form: workspace folder (required), author (optional). Built and attached to each release by GoReleaser.
- One in-process write lock (index upserts were never concurrent in the CLI).
- Tests over the SDK's `InMemoryTransport`, one per spec scenario, as the CLI does.
- Release as **v0.4.0**.

## Capabilities

### New Capabilities
- `mcp-server`: the stdio server, its workspace binding, the tool set and contract, instructions/prompt delivery, provenance defaults from the handshake, error mapping, and concurrency.
- `mcp-bundle`: the `.mcpb` manifest, contents, `user_config`, platform handling, and release attachment.

### Modified Capabilities
<!-- none: `serve` is an additional verb; no existing requirement changes -->

## Impact

- New dependency: `github.com/modelcontextprotocol/go-sdk` (v1, stable). Binary grows by a few MB.
- New package `internal/mcp` (tool handlers, instructions, bundle manifest template); `internal/cli/cmd_serve.go`.
- `.goreleaser.yaml`: a post-build step assembling `particulars-<ver>.mcpb` (a zip) and attaching it to the release; `Makefile` target `bundle`.
- README: "Use from Claude Desktop / any MCP client" section; `docs/mcp.md` with the tool table, an example `claude_desktop_config.json` / `.mcp.json`, and how Claude Code picks the workspace.
- Knowledge workspace gains the MCP tool contract as claims once shipped.
- Not in scope: a hosted/remote server (likely a separate, database-backed implementation sharing only the tool contract), streamable HTTP transport, `feed_index` / `knowledge_publish` (federation), multi-workspace addressing within one server (designed in exploration; additive later as an optional `workspace` parameter), git writes of any kind, MCP resources (`dkf://…`) — deferred until a client asks for them.
