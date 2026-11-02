## Context

- **The format gate.** `store.Config.Validate` refuses any `format` other than `dkf.FormatVersion` (`dkf/0.1`). That constant serves three purposes at once: the format the CLI reads, the format it writes into `dkf.yaml` and `index.yaml`, and what `version` reports.
- **Paths.** Every id-to-path derivation goes through `Workspace.Path(id)`, which returns `<type-dir>/<id>.yaml`.
- **Loading.** Every directory walk is in `Workspace.Load`. It reads each type directory and silently skips subdirectories.
- **Writes.** They funnel through four methods: `Workspace.Create` (new objects), `WriteParticular`, `Retract`, and `WriteIndex` (which `UpsertIndex` and `RebuildIndex` call). `export` writes graph, dot or mermaid output outside the workspace; the CLI has no DKF feed writer.
- **Validation.** `validate` calls `CheckIndex`, which rebuilds one document and compares it with `index.yaml`.

The upstream `v0.2` tag defines the read side precisely. The relevant requirements are in `workspace-layout`: *Object files are sharded…*, *A reader reads the layout its declared version names*, and *A validator warns on time it cannot have seen*. The write side (index segments, sealing, guards, migration) is stages 2 and 3 of #11.

## Goals / Non-Goals

**Goals:**
- Every read verb works on a `dkf/0.2` workspace exactly as on a `dkf/0.1` one: `recall`, `lineage`, `conflicts`, `register`, `topics`, `workspace`, `unresolved`, `validate`, and the MCP read tools.
- `validate` enforces the layout: misplaced files, plus the time rules (warning and failure).
- Path derivation and the loader are the final versions, so stages 2 and 3 build on them unchanged.
- Nothing changes for a `dkf/0.1` workspace, apart from the new time checks.

**Non-Goals:**
- Writing `dkf/0.2` in any form: objects, the index, retractions, `init`, or `migrate`. That is stages 2 and 3. #11's "feed paths" item doesn't apply to this CLI, which writes no feed.
- Reading or checking the `dkf/0.2` index (root, segments, `{path, count}`, per-month `retracted`). No read verb uses the index, so leaving it out costs only `validate`'s index comparison and `index --check`.
- Late-merge detection (a file arriving in a sealed month). It needs segment counts, so it belongs to stage 2.
- `dkf/0.3` redaction (#23).

## Decisions

### D1. Month derivation is a pure function in `dkf`

`dkf.MonthOf(id string) (string, bool)` takes the identifier part after the `<prefix>_`. It returns the UTC `YYYY-MM` of the UUIDv7 timestamp, or `false` when no month can be derived.

The identifier part counts as a UUIDv7 when all three hold:
- it is in the 36-character hyphenated form, with hex digits in either case;
- its version nibble is 7;
- its variant is RFC 4122.

The timestamp is the first 48 bits, read as Unix milliseconds. Any other identifier part gets no month and goes to `legacy/`. Examples: draft-era ids such as `clm_07m3zp9s2q1r4t8v`, a UUIDv4, the 32-hex form without hyphens.

- **Why parse the bits directly:** reading 12 hex digits is a few lines that are easy to test. It also avoids depending on how `google/uuid`'s `Time()` represents a version 7 UUID.
- **Why upper case gets a month:** the spec says "a UUID … of version 7" and doesn't require lower case. An upper-case UUIDv7 still gets the existing `legacy_id` warning, since it isn't this CLI's canonical form. Its path is its month.

### D2. The format gate names a set

`dkf` gets:

```go
var ReadFormats = []string{"dkf/0.1", "dkf/0.2"}
const WriteFormat = "dkf/0.1" // becomes dkf/0.2 in stage 3
```

`Config.Validate` refuses any format not in `ReadFormats`, with the message `dkf.yaml: unsupported format "dkf/0.3" (this release reads dkf/0.1, dkf/0.2)`. The error wraps a new `store.ErrUnsupportedFormat`. `validate` maps that to the code `unsupported_format`, the same way it already maps `ErrInvalidBaseURI` to its own code.

`Workspace.Sharded()` reports `Config.Format == "dkf/0.2"`. It is the only place the layout is decided. Nothing else compares format strings.

### D3. `Path` and `Load` branch once, on `Sharded()`

- **`Path(id)`:** in a flat workspace, as today. In a sharded one, `<type-dir>/<MonthOf(id) or "legacy">/<id>.yaml`.
- **`Load`:** in a flat workspace it stays exactly as today, including skipping subdirectories. In a sharded workspace it reads each type directory as follows:
  - A file at the top level is reported as `misplaced_file`, with its derived path when the file name is a valid id.
  - A subdirectory named `YYYY-MM` (a real month) or `legacy` is read. Each file in it goes through the existing parse, id, type and prefix checks, and then one more: its directory must be the one its id derives. If not, it is reported as `misplaced_file`, naming the file and its derived path.
  - Any other subdirectory: each `.yaml` file in it is reported as `misplaced_file`.
  - Hidden files and non-`.yaml` files are ignored at both levels, as today.
  - A misplaced file is never added to the graph: "the reader does not treat it as an object".

`misplaced_file` becomes a `FileProblem`, so `validate` reports it as an error with no extra plumbing, as it already does for `type_mismatch`.

### D4. Writes refuse at four choke points

`Create`, `WriteParticular`, `Retract`, `WriteIndex` and `RebuildIndex` each start with `w.writable()`. In a sharded workspace it returns `ErrReadOnlyFormat`: `this release reads dkf/0.2 but does not write it yet (particulars-cli#11)`. `RebuildIndex` checks before loading, so `particulars index` refuses without reading the tree. `export` is not guarded: it writes derived graph output outside the workspace, so it is a read verb.

Placing the guard in the store means no write path can miss it, including `UpsertParticular`, merges, promotions and every MCP tool. An object write always comes before its index write, so a refusal never leaves a file without its index entry.

### D5. The index is not read in a `dkf/0.2` workspace

- `CheckIndex` returns `ErrReadOnlyFormat` on a sharded workspace, so `index --check` refuses with that message.
- `validate` skips the comparison and adds one warning: `index_unchecked`, "checking a dkf/0.2 index comes in a later release". It warns rather than staying silent so that a passing `validate` doesn't look like a passing index check.
- Stage 2 replaces both behaviours.

### D6. Time checks run for both layouts

`validate` gets a `now` from a package-level clock that tests can override.
- **Warning `future_time`:** an id whose minting instant is after `now`, or an object or retraction `timestamp` after `now`. One finding per object, naming what is ahead.
- **Error `far_future_id`:** an id whose month is two or more months after `now`'s month.

The upstream requirement is about validators, not layouts, so it applies to `dkf/0.1` too.
- **Risk for existing workspaces:** a `dkf/0.1` workspace could only trip the error with an id minted on a clock two months fast. None of the three dogfood workspaces has one; task 4.2 checks.
- **Backdated timestamps** are in the past and never trip either check. The bench backdates `timestamp` to 2023.

## Risks / Trade-offs

- [A read-only `dkf/0.2` release could look like support] → Every refusal names #11 and says writing comes later. `version` reports `writes dkf/0.1`, and the changelog says it plainly.
- [The new time checks could surface warnings in existing workspaces] → A warning never fails validation, and the dogfood check in task 4.2 runs before release.
- [`Load` is the hottest path in the CLI] → The flat branch is unchanged. The sharded branch adds one `ReadDir` per month directory, which is negligible next to parsing the files.
- [Stage 2 has to undo the refusals in D4 and D5] → They are five small guards, gathered behind `writable()` and one check in `CheckIndex`, so they're easy to find and remove.

## Migration Plan

Nothing to migrate. Every existing workspace is `dkf/0.1` and behaves as before, apart from the new time checks. Release as a minor version.

## Open Questions

None.
