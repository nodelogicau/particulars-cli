## Why

v0.13.1 renamed the default conventions document to `dkf.md` and, instead of a fallback, printed a one-release migration notice when a lone `CONVENTIONS.md` was found. The changelog committed to removing that notice in the following release. This is that removal: the notice has done its job for the one release it was promised for, and keeping it means every workspace pays a stat call and carries a field for a file the format no longer knows.

## What Changes

- **The legacy notice is removed.** `particulars workspace` no longer reports `conventions_legacy` or warns about `CONVENTIONS.md`; `serve --mcp` no longer prints the notice at startup. A `CONVENTIONS.md` at the root is now simply a file the tool does not read, like any other.
- The store loses `LegacyConventionsFile`, `LegacyConventionsNotice`, and `LegacyConventions()`.
- The MCP guide drops the sentence describing the notice; the migration itself (rename, or name the file in the key) stays documented.

Nothing is **BREAKING** for objects. A v0.12.0 workspace that never migrated loses the reminder, not any behaviour: its file was already not read.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `workspace`: the conventions file requirement drops the legacy-notice clause and its two scenarios.

## Impact

- `internal/store/workspace.go`, `internal/cli/cmd_workspace.go`, `internal/cli/cmd_serve.go`, their tests, `docs/mcp.md`, `CHANGELOG.md`.
