## Why

DKF v0.2 was declared on 2026-11-03. Its one breaking change is the sharded layout:
- object files live at `<type-dir>/<YYYY-MM>/<id>.yaml`, or `<type-dir>/legacy/<id>.yaml` for an id from which no month can be derived;
- the index is split into a root and sealed segments.

The CLI implements none of this (#11). It refuses any `format` other than `dkf/0.1`, so it can't read a v0.2 workspace written by anything else, and it can't be the implementation the dogfood workspaces migrate onto.

#11 splits the work into three releasable stages. This change is stage 1: **read both layouts**. Reading comes first because the other two stages depend on it. The index writer (stage 2) and `migrate` (stage 3) both need path derivation, the layout-aware loader, and validation that rejects a misplaced file. Reading also stands alone: a v0.2 workspace becomes readable and checkable by `recall`, `validate`, `lineage` and the MCP server before the CLI writes one.

The upstream reference is the `v0.2` tag of nodelogicau/particulars, `1366d3b`: `workspace-layout`, `workspace-config` and `object-identifiers`. Upstream `main` also carries `dkf/0.3` redaction, which is out of scope here and tracked in #23.

## What Changes

- **Format gate.** `dkf.yaml` may declare `dkf/0.1` or `dkf/0.2`. Any other value is refused, and the message names the supported formats.
  - `validate` reports the refusal under a new code, `unsupported_format`, instead of `parse_error`.
  - `dkf.FormatVersion` splits into the set of formats this release reads and the one format it writes. The written format stays `dkf/0.1` until stage 3.
- **Path derivation from the id.** In a `dkf/0.2` workspace, `Workspace.Path(id)` returns `<type-dir>/<YYYY-MM>/<id>.yaml`. The month is the UTC calendar month of the UUIDv7 timestamp in the id. An id that isn't a UUIDv7 maps to `<type-dir>/legacy/<id>.yaml`. In a `dkf/0.1` workspace, paths stay flat.
- **Layout-aware loading.** In a `dkf/0.2` workspace, `Graph.Load` reads each type directory's month directories and its `legacy/` directory. It never infers the layout from what it finds. A file is reported as misplaced and is not loaded if it is:
  - a stray file at the top of a type directory;
  - in a month directory its id doesn't derive;
  - a legacy id inside a month directory;
  - inside a directory that is neither a month nor `legacy`.

  Each report names the file and its derived path.
- **Time checks in `validate`, for both layouts.** `validate` warns (`future_time`) on any id whose minting instant is later than the validator's clock, and on any object or retraction `timestamp` later than that clock. It fails (`far_future_id`) on an id whose month is two or more months after the validator's current month.
- **A `dkf/0.2` workspace is read-only in this release.** Every write refuses with a message saying that writing `dkf/0.2` comes in a later release: object files, retractions, particular updates, and `index`. `export` writes graph, dot or mermaid output outside the workspace, never a DKF feed, so it reads a `dkf/0.2` workspace like any other read verb. `validate` skips the index comparison for a `dkf/0.2` workspace and warns `index_unchecked`. `index --check` on a `dkf/0.2` workspace refuses the same way. **This is temporary, and stage 2 removes it.**
- **`version`** reports the formats this release reads and the format it writes.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `workspace`:
  - *dkf.yaml structure* accepts `dkf/0.1` or `dkf/0.2`;
  - *Directory layout* gains the sharded paths;
  - a new requirement makes a `dkf/0.2` workspace read-only for now.
- `validation`:
  - *Structural checks* gains `unsupported_format`;
  - *Index consistency check* skips a `dkf/0.2` index;
  - new requirements cover layout checks and time checks.
- `index`: *Check* refuses on a `dkf/0.2` workspace for now.
- `cli-interface`: *Version verb* reports the formats read and the format written.

## Impact

- `internal/dkf`:
  - `types.go`: `FormatVersion` becomes `ReadFormats` and `WriteFormat`.
  - `id.go`: a new `MonthOf(id) (month string, ok bool)`.
- `internal/store`:
  - `config.go`: the format gate.
  - `workspace.go`: `Path` and a write guard in `Create`, `WriteParticular` and `Retract`.
  - `graph.go`: the layout-aware `Load` and misplaced-file problems.
  - `index.go`: a write guard in `WriteIndex`, and `CheckIndex` refuses on `dkf/0.2`.
- `internal/query/validate.go`: the new codes, time checks, and the index skip.
- `internal/cli`: `version`; `validate`'s handling of the config error; the `index` refusal.
- Tests, `docs/`, and `CHANGELOG.md`.
- The skill and the dogfood workspaces are unchanged: nothing writes `dkf/0.2` yet.
- No change to `dkf/0.1` behaviour except the new time checks, which apply to both layouts.
