## 1. Foundations

- [x] 1.1 Add `github.com/modelcontextprotocol/go-sdk` (pin v1.x); confirm `InMemoryTransport`, `StdioTransport`, `AddTool` generics, `ServerOptions.Instructions`, `ToolAnnotations` in the pinned version
- [x] 1.2 Move error classification (`classify`, codes) from `internal/cli` to a shared package so CLI and MCP map identically; CLI behaviour unchanged (tests stay green)
- [x] 1.3 `internal/mcp`: server construction — bind workspace via `store.DiscoverWith`, instructions text (header + `skill.Body()`), `particulars-discipline` prompt, server-wide write mutex, per-session harness default from `clientInfo.name`
- [x] 1.4 Provenance resolution shared with the CLI (call → flags → env → dkf.yaml → clientInfo), reusing `requireProvenance`

## 2. Tools

- [x] 2.1 `particular_define`, `particular_resolve` (null on miss; usage error on ambiguity), `particular_merge`
- [x] 2.2 `claim_assert`, `claim_retract` (claims, syntheses, merges; `superseded_by`), `synthesis_create` (with `particular_id`; retracted-input warnings in result)
- [x] 2.3 `knowledge_recall` (`particular_id` or `query` as topic, scope, topics, include_retracted, limit; class-aware), `lineage_trace`
- [x] 2.4 `conflict_detect`: particular form via `query.Analyse`; claim-set form via a new `query.AnalyseSet(g, ids)`; exactly-one validation
- [x] 2.5 Extension tools `topics_list`, `workspace_status` (counts, validate summary, conflicts, read-only `git status --porcelain` of the workspace path when inside a checkout)
- [x] 2.6 Result shaping: structured content = CLI `--json` value; one-line text summary; error results with CLI codes; annotations (read-only / idempotent)
- [x] 2.7 `internal/cli/cmd_serve.go`: `serve --mcp` with `--workspace/--author/--harness/--model`; exit 5 without a workspace; stdout reserved for the protocol (CLI text emitters pointed at stderr while serving)

## 3. Tests

- [x] 3.1 `internal/mcp` test harness: client over `InMemoryTransport` against a temp workspace, with a `clientInfo` of choice
- [x] 3.2 Scenario tests for every requirement in `specs/mcp-server`: each tool's happy path and errors, null resolve, claim-set conflicts, class-aware recall, handshake harness default and override, instructions/prompt content, twenty parallel asserts → `index --check` clean
- [x] 3.3 A test that starts the real binary with `serve --mcp` over stdio (using the SDK's `CommandTransport`), initialises, lists tools, calls `workspace_status`, and asserts stdout carried only JSON-RPC

## 4. Bundle

- [x] 4.1 `bundle/manifest.json` template (verify `platform_overrides` keys against the MCPB spec; fall back to a darwin universal binary if per-arch overrides are not expressible), `bundle/icon.png`
- [x] 4.2 `make bundle`: assemble `dist/particulars-<ver>.mcpb` from cross-compiled binaries; validate the manifest (JSON + referenced files exist); unit test for the assembly script
- [x] 4.3 GoReleaser: build the bundle after archives and attach it to the release with its checksum
- [x] 4.4 Install the bundle in Claude Desktop on this Mac: pick the knowledge workspace, confirm tools appear, run `workspace_status` and a `knowledge_recall` from a Desktop chat

## 5. Docs, Feedback, Release

- [x] 5.1 `docs/mcp.md`: tool table (spec vs extension), client configs (Desktop bundle + manual, Claude Code `.mcp.json`, Cursor), workspace binding, provenance defaults, review caveat for Desktop users
- [x] 5.2 README: "Use from Claude Desktop or any MCP client" section linking `docs/mcp.md`; verb table gains `serve --mcp`
- [x] 5.3 `SPEC-FEEDBACK.md` item 13: `synthesis_create` signature lacks the subject particular
- [x] 5.4 `CHANGELOG.md` v0.4.0; tag; verify the release has the `.mcpb`, install it from the release asset, and run the stdio smoke against the published binary
- [x] 5.5 Record in the knowledge workspace (branch): the MCP tool contract as the interop boundary between this implementation and a future hosted one, with the spec URL as evidence
