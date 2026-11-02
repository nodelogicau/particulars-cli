## Why

Stage 1 of #11 (`read-dkf-02`) made the CLI read a `dkf/0.2` workspace. It refuses every write there, because the `dkf/0.2` index differs from the single `index.yaml` the CLI maintains. In `dkf/0.2` the index is:
- a root with the two open months' entries and retractions;
- one sealed segment per earlier month, listed in the root with a `count`;
- a `legacy` segment for ids that are not UUIDv7.

Stage 2 implements that index, so a `dkf/0.2` workspace can be written, rebuilt and checked. Stage 3 (`migrate`, and `init` writing `dkf/0.2`) depends on it.

The reference is the upstream `v0.2` tag (`1366d3b`):
- `index-manifest`: *The index is a hot tail, sealed segments, and a tombstone list*, *Implementations provide rebuild and drift-check operations*, *Validation compares sealed segments against the files*;
- `workspace-layout`: *A sealed month gains no files*.

## What Changes

- **The `dkf/0.2` index shape** (`format: dkf/0.2` in every document):
  - **The root `index.yaml`** carries, in this order:
    - `format`;
    - `segments`: `{path, count}` for `index/legacy.yaml` first, then each sealed month in ascending order;
    - `entries`: objects minted in the current month or the one before it;
    - `retracted`: sorted ids of objects whose `retracted.timestamp` falls in an open month or later.
  - **A sealed segment** `index/<YYYY-MM>.yaml` carries that month's mints as `entries` and its retractions as `retracted`.
  - **`index/legacy.yaml`** carries entries only.
  - **Months.** The current month is the month of the newest id. A month is sealed once an id exists two or more months after it.
  - **No `retracted` field on entries.**
- **Rebuild (`particulars index`)** is a pure function of the files. It writes the root, `index/legacy.yaml`, and any sealed segment that doesn't exist yet. It never rewrites an existing sealed segment: one that differs from the files is reported, and the command exits 4. Unknown entry types and fields are kept, as in `dkf/0.1`.
- **Incremental maintenance.** Every write verb, from the CLI or MCP, updates only what the operation touches:
  - a mint adds its entry to the root, or to `index/legacy.yaml`;
  - a retraction adds its id to the root's `retracted`;
  - a mint in a month after the current one also seals the months that just closed, moving their entries and retractions out of the root into new segments.

  After every operation the result equals a full rebuild.
- **Sealing guards.** The CLI refuses to mint into a sealed month, and refuses to date a retraction into one. It mints and dates at its own clock, so only a clock more than a month slow trips either guard. The message names the sealed month and says to check the clock.
- **`index --check`** works on `dkf/0.2`:
  - for each sealed month it compares the segment's `count` with the files in that month's directories, and parses the segment only when they differ;
  - known files missing from the segment are reported as a **late merge**;
  - the root, `index/legacy.yaml` and any segment whose count differed are compared in full, both `entries` and `retracted`;
  - missing and extra segments are reported.

  The `dkf/0.1` check is unchanged.
- **`validate`** on `dkf/0.2`:
  - `index_stale` from the check above;
  - `late_merge`, naming the files;
  - `sealed_retraction`: it rebuilds each sealed month's `retracted` list from the files and compares it with the committed segment, naming the object and the segment.

  `index_unchecked` is removed.
- **The read-only refusal from stage 1 is removed**, along with `ErrReadOnlyFormat`. **BREAKING** for anyone relying on it, but it was never released.
- **`init` still writes `dkf/0.1`.** Writing a new `dkf/0.2` workspace, and migrating to one, is stage 3.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `index`:
  - *Index structure* gains the `dkf/0.2` shape;
  - *Incremental maintenance* covers sealing and keeps only the bounded writes;
  - *Rebuild* never rewrites a sealed segment;
  - *Check* gains the count-first comparison and late merges;
  - *A dkf/0.2 index is not yet read or written* is removed.
- `workspace`: *A dkf/0.2 workspace is read-only in this release* is removed, and *Sealed months gain nothing new* is added for the two guards.
- `validation`: *Index consistency check* covers `dkf/0.2`, and *Sealed segment checks* is added for `late_merge` and `sealed_retraction`.

## Impact

- `internal/store/index.go`: the bulk of the change.
  - `Index` gains `Segments` and `Retracted`.
  - Months are derived from ids and retraction timestamps.
  - New functions: `BuildSharded`, the sharded rebuild, upsert-with-sealing, and the count-first `CheckIndex`.
  - `IndexDiff` gains late-merge, segment and retracted differences.
- `internal/store/workspace.go`: `writable()` and `ErrReadOnlyFormat` go; the sealing guards come in `Create` and `Retract`; there's a cheap current-month lookup from the month directories.
- `internal/query/validate.go`: `late_merge` and `sealed_retraction` are added; `index_unchecked` goes.
- `internal/cli` and `internal/apperr`: error mapping for the guard (`sealed_month`) and the `index` report.
- Tests, including a property test that incremental maintenance equals a rebuild across month rollovers.
- `docs/` and `CHANGELOG.md`.
- No change to `dkf/0.1` behaviour.
