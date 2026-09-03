## Why

v0.12.0 shipped workspace conventions ahead of the spec; upstream has now blessed the feature (`workspace-conventions`, archived 2026-09-03, accepting nodelogicau/particulars#23) with four differences, and particulars-cli#8 lists them. All are additive, but one is a rename of the default file and one turns a hard error into a warning, so a v0.12.0 workspace and a spec-conformant workspace currently disagree about which file is read and whether the workspace opens at all. The floor-and-boundary truncation rule also exposes a real defect: the server slices bytes at 16 KiB and can split a UTF-8 sequence.

## What Changes

- **The default conventions file is `dkf.md`**, the prose sibling of `dkf.yaml`, replacing `CONVENTIONS.md`. There is deliberately **no fallback** that keeps reading `CONVENTIONS.md`: the spec's reason for a DKF-specific name is that a generic file already present for another tool gets delivered silently, and a fallback preserves exactly that. Instead, a root that holds `CONVENTIONS.md` with neither `dkf.md` nor `workspace.conventions` gets a one-line stderr notice from `serve --mcp` and `particulars workspace`, and `workspace --json` reports `conventions_legacy`. The notice is removed after one release.
- **An invalid `workspace.conventions` is a warning, not a refusal.** The check stays lexical on the cleaned slashed path (absolute, leading slash, or first segment `..`); an invalid value is treated as unset, reported on stderr by `serve` and `workspace`, and carried in `workspace --json` as `conventions_invalid`. The workspace opens under every verb.
- **Truncation honours the floor and the boundary.** At least the first 16 KiB of UTF-8 is delivered and the cut lands only on a character boundary — by advancing to the *next* rune start, never backing off, since backing off would deliver less than the floor. The note naming the file stays.
- **The conventions document is an MCP resource**, read on demand, listed only when a document applies — the spec's MAY, taken because it is the only channel to clients that never surface `instructions`.
- **Docs**: the README line calling `.dkf` an implementation extension is stale (upstream blessed it in #12) and is corrected; the conventions section notes that `AGENTS.md` is a good *value* for the key when the workspace directory is its own agent scope.
- Register in tool descriptions: already shipped in `knowledge-not-catalogue`; no change.

Nothing is **BREAKING** for objects. The rename is a behaviour change for a v0.12.0 workspace that relies on the default filename; the notice exists for that case.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `workspace`: the conventions file requirement — `dkf.md` default, lenient key with `conventions_invalid`, legacy notice with `conventions_legacy`.
- `mcp-server`: the instructions requirement — `dkf.md`, boundary-safe truncation as a floor; plus a new requirement exposing the document as a resource.

## Impact

- `internal/store/workspace.go` (`ConventionsFile`, `Conventions()`, legacy detection), `internal/store/config.go` (`Validate` loses the conventions check; a `ConventionsPath` accessor carries the warning), `internal/mcp/server.go` (truncation, resource registration), `internal/cli/cmd_serve.go` and `cmd_workspace.go` (warnings and notice), `skills/particulars/SKILL.md` + installed copy, `docs/mcp.md`, `README.md`, `CHANGELOG.md`.
- Dogfood: the knowledge workspace has no conventions file and no key, so nothing changes there.
