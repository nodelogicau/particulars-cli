## Context

The skill carries workspace-agnostic discipline; conventions are per workspace. MCP instructions are assembled once, at server startup, from a header plus the embedded skill body (`internal/mcp/server.go: instructions()`), and the server is bound to one workspace for its lifetime — so startup is also the right moment to read the conventions document.

## Goals / Non-Goals

**Goals:** a zero-config path (`CONVENTIONS.md` present → delivered), an explicit path (`workspace.conventions`), and visibility (`particulars workspace` says which file applies).

**Non-Goals:** validating or structuring the document (it is prose for a model); re-reading it mid-session (the workspace binding is per-session already); delivering it to non-MCP harnesses (they read the repo).

## Decisions

- **Default-on with `CONVENTIONS.md`, override by config.** A fixed default makes the common case zero-config; the key covers workspaces whose document already exists under another name (`TOPICS.md`). Absence of both is silence, not a warning.
- **Config validity is writer-strict:** `workspace.conventions` must be a relative path that stays inside the workspace; an absolute path or `..` escape fails config validation — the document travels with the workspace or it is not the workspace's.
- **Missing-but-configured is lenient at read time:** `serve` emits a stderr diagnostic and omits the section; `workspace` reports `conventions_missing`. A typo in dkf.yaml must not take the server down.
- **16 KiB cap with a truncation note** naming the file, so one oversized document cannot swamp every session's context. The note tells the model where the rest lives.
- **Read once at startup**, matching the workspace binding; a changed file applies to the next session.

## Risks / Trade-offs

- [Instructions bloat] → the cap, and docs advising brevity.
- [Stale conventions in long sessions] → same staleness class as every other startup-bound property; acceptable.

## Migration Plan

Additive; no file or config changes required anywhere.

## Open Questions

- None.
