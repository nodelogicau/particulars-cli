## Why

LLM agents accumulate knowledge while working, but every harness stores it in a proprietary silo and treats contradiction as an error to overwrite. The [Dialectical Knowledge Format (DKF)](https://github.com/nodelogicau/particulars) defines an open, file-based format (particulars, claims, syntheses as YAML in a git repo) that preserves contradiction and provenance. There is no tool that lets an agent write and read DKF workspaces today; `particulars` is that tool, and a reference implementation that will stress-test the v0.1 draft before it is finalised.

## What Changes

- New `particulars` CLI: a single, statically linked native binary (Go) with no runtime dependencies.
- Workspace bootstrap (`init`) creating the DKF directory layout plus a `dkf.yaml` marker/config file; workspace discovery by walking up from the current directory.
- Object model conforming to DKF v0.1: `DPARTICULAR`, `DCLAIM`, `DSYNTHESIS`, one YAML file per object, deterministic serialisation in spec field order.
- UUIDv7-based identifiers with spec prefixes (`par_`, `clm_`, `syn_`) so files sort chronologically on disk.
- Verbs mirroring the spec's non-federation tools: `particular define|resolve`, `claim assert|retract`, `synthesis create`, `recall`, `conflicts`, `lineage`.
- Retraction recorded as an append-only `retracted:` block on the object file — the only mutation ever applied to an existing file.
- Structural (non-semantic) conflict detection: unsynthesised claims and stale syntheses per particular. The agent reasons; the tool stores and reports.
- Derived `index.yaml` with `index` (rebuild) and `index --check` (verify) so merge conflicts on the index are resolved by regeneration.
- `validate` to lint a workspace against the format — useful to other DKF implementers and as a PR check.
- Agent-first interface: every verb supports `--json`, never prompts, uses documented exit codes. Human review happens through git pull requests, so the CLI performs no git operations.

## Capabilities

### New Capabilities
- `cli-interface`: Global CLI contract — non-interactive operation, `--json` output on every verb, exit codes, workspace selection.
- `workspace`: `init`, the `dkf.yaml` config/marker file, directory layout, and upward workspace discovery.
- `object-format`: Identifier scheme (UUIDv7 + prefixes), file naming, deterministic YAML serialisation, create-only write discipline.
- `particulars`: `particular define` (idempotent on URI, URI minting from slug or base URI) and `particular resolve`.
- `claims`: `claim assert` and `claim retract` including the `retracted` block semantics.
- `syntheses`: `synthesis create` with input validation and the mandatory `unresolved` field.
- `knowledge-query`: `recall` (lineage-ordered retrieval per particular, retracted filtering) and `lineage` (provenance tree traversal).
- `conflict-detection`: `conflicts` reporting unsynthesised claims and stale syntheses per particular.
- `index`: Derived `index.yaml` generation and verification.
- `validation`: `validate` checks for structural and referential integrity of a workspace.

### Modified Capabilities
<!-- none — greenfield -->

## Impact

- New Go module in this repository; new dependencies: a CLI framework (e.g. `cobra`), `gopkg.in/yaml.v3`, a UUIDv7 generator (e.g. `github.com/google/uuid`).
- Build/release pipeline producing cross-compiled binaries for macOS, Linux, and Windows (amd64/arm64).
- Produces feedback to the upstream DKF spec (ID format, retraction representation, URI scheme for unpublished particulars, `dkf.yaml` convention, index as derived cache) — tracked in `design.md`.
- Out of scope for this change: federation tools (`feed_index`, `particular_merge`, `knowledge_publish`), MCP server transport, git operations, any LLM/semantic reasoning inside the binary. The core package is structured so a later `serve --mcp` can reuse it.
