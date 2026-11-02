## Context

**What stage 1 left.** `read-dkf-02` gave the CLI:
- path derivation (`dkf.MonthOf`, `Workspace.Path`);
- a sharded loader that reports `misplaced_file`;
- a read-only guard (`writable()`) in `Create`, `WriteParticular`, `Retract`, `WriteIndex` and `RebuildIndex`;
- a `CheckIndex` that refuses on `dkf/0.2`.

**The `dkf/0.1` index code** (`internal/store/index.go`) assumes one document holding one sorted `entries` list:
- `Index` keeps entry types it doesn't recognise as raw YAML nodes (`Unknown`), and `Entry` keeps fields it doesn't recognise, so a rebuild preserves both.
- `BuildIndex` derives the document from the graph.
- `UpsertIndex` reads, upserts and writes it, falling back to a rebuild.
- `CheckIndex` rebuilds and compares, after masking MAY fields the committed index lacks.

**The upstream `v0.2` rules** (`index-manifest`, `workspace-layout`) are summarised in the proposal. Points that bear on the design:
- A rebuild is a pure function of the files, and the open months follow the newest id, not the clock.
- A sealed segment never changes once written.
- The drift check compares counts first.
- `validate`, not the drift check, is where a retraction dated into a sealed month is caught.

## Goals / Non-Goals

**Goals:**
- **Writes work.** Every write verb works in a `dkf/0.2` workspace. Afterwards the index equals a rebuild, and the only documents written are those the operation touches: the root, `index/legacy.yaml`, or new segments when months seal.
- **No sealed segment is rewritten**, by any path, rebuild included.
- **Checks.** `index --check` and `validate` implement the upstream count-first comparison, the late-merge report and the sealed-retraction comparison.
- **No `dkf/0.1` change.** Every existing index test passes unchanged.

**Non-Goals:**
- `migrate`, `init` writing `dkf/0.2`, and the dogfood migrations. That's stage 3.
- Answering `recall` from the index. The CLI reads files, as today.
- `dkf/0.3` redaction (#23).
- Making `Load` cheaper. Every verb still parses every file. The working-set principle bounds what is *written*; the CLI's read cost is a separate matter, measured in `docs/benchmarks.md`.

## Decisions

### D1. One document type, three roles

`Index` gains `Segments []SegmentRef` (`{Path, Count}`) and `Retracted []string`. All three documents use it:

| Document | Keys |
|---|---|
| root | `format`, `segments`, `entries`, `retracted` |
| sealed segment | `format`, `entries`, `retracted` (`[]` when empty) |
| `index/legacy.yaml` | `format`, `entries` |

- The keys are emitted in the order of the upstream README examples. `Segments` and `Retracted` are left out of a `dkf/0.1` document, so its bytes don't change.
- `format` is the workspace's own format (`w.Config.Format`), not `dkf.WriteFormat`. A `dkf/0.1` workspace keeps writing a `dkf/0.1` index. `WriteFormat` keeps meaning only "what `init` writes".
- A `dkf/0.2` entry never carries `retracted`. `EntryFor` still sets it, and the sharded builder clears it.

`ShardedIndex{Root *Index; Docs map[string]*Index}` holds a full set, keyed by path relative to the root (`index/legacy.yaml`, `index/2026-08.yaml`). Building, writing and comparing all work on that one shape.

### D2. Months are computed once, from the graph or from the root

- `month(id)` is `dkf.MonthOf`. `retractionMonth(obj)` is the UTC month of `retracted.timestamp`.
- **Current month:** the newest id's month across the workspace. The open months are the current month and the one before it; any earlier month is sealed. Months are compared as integers (`year*12 + month`), as in the stage 1 time checks.
- **No UUIDv7 id at all** (empty, or legacy-only): there is no current month and nothing is sealed. The root holds every retraction, `index/legacy.yaml` every entry, and the root's `entries` is empty.

`BuildSharded(g)` assigns each object as follows:
- its **entry** goes to legacy, to the root if its month is open, or to its sealed month's document;
- its **retraction**, if any, goes to the root if the month is open or later, otherwise to that sealed month's document.

A sealed month with only retractions gets a segment with empty `entries`; a month with neither gets none. `count` is the number of entries in the document, including preserved unknown entries.

### D3. Rebuild never overwrites a sealed segment

`RebuildIndex` in a sharded workspace:
1. Builds the full set (D2), preserving unknown entry types and fields from whichever committed document held them (D6).
2. Writes the root and `index/legacy.yaml`, removing `index/legacy.yaml` when there are no legacy ids.
3. Writes each sealed segment that doesn't exist yet.
4. For each existing sealed segment, compares it byte for byte with the rebuilt one. If they differ, leaves the file alone and records the path.
5. Returns the list of differing segments. `particulars index` reports them and exits 4: `index/2026-08.yaml differs from the files and is sealed; it is never rewritten — run particulars validate`.

**Why not rewrite:** a sealed segment is something a remote consumer may cache forever. The upstream fix for a late merge is to re-mint on the branch, not to rewrite history. Rewriting would hide exactly what `validate` exists to catch. An unparseable root (git conflict markers, say) is replaced, as in `dkf/0.1`, because conflicts can only arise in open documents.

*Alternative:* `--force` to rewrite sealed segments. Not now: no workflow needs it, and it would be the one way to break the caching guarantee.

### D4. Incremental maintenance updates the root in place, and seals on rollover

`UpsertIndex(obj)` in a sharded workspace:
1. Reads the root. If it's missing or unparseable, falls back to a full rebuild (D3), as in `dkf/0.1`.
2. Finds where the entry belongs:
   - a legacy id goes to `index/legacy.yaml`, and the root's count for it is updated;
   - an id in an open month goes to the root;
   - an id in a sealed month gets no write, because the entry fields are immutable. This happens when a particular is updated or an old object is retracted; `uri` is fixed by define being idempotent on URI.
3. If the object carries a retraction, adds its id to the root's `retracted`, sorted and without duplicates.
4. **Rollover.** If the new id's month is after the root's current month, the months that become sealed are cut out of the root:
   - root entries minted in a now-sealed month move to that month's new segment;
   - root `retracted` ids whose retraction month is now sealed move to that month's segment. Finding that month means reading those objects' files, which costs at most the root's own `retracted` list.

   New segments are written, the root's `segments` gains their `{path, count}`, and then the root is written. If a segment for a month being sealed already exists, the rollover stops and falls back to D3. That can only happen if the index was edited by hand, and D3 reports it.

The root's current month is the newest month among its entries. If it has no entries, the workspace has no UUIDv7 id (D2).

**Correctness argument:** each step moves exactly what D2 would put in a different document, and nothing else. A test drives sequences of mints, retractions and particular updates across gaps of one, two and five months, including months with only retractions and retractions dated ahead of the newest id. After every step it checks that the written documents equal `BuildSharded` of the files (task 3.3).

### D5. The guards use the month directories, not the index

`Create` (for a UUIDv7 id) and `Retract` compute the current month as the newest `YYYY-MM` directory that holds a file, across the type directories. That costs one `ReadDir` per type directory and doesn't depend on the index being present or current.
- **Mint into a sealed month:** `ErrSealedMonth`, with the message `this machine's clock puts the new id in 2026-08, which is sealed (the workspace's newest id is from 2026-11); check the clock`.
- **Retraction timestamp in a sealed month:** the same error, saying to date the retraction now and give the decision date in `reason`.
- `apperr` maps `ErrSealedMonth` to `sealed_month`, exit 1.

The CLI and MCP always mint and date at `time.Now()`, so these guards fire only on a clock more than a month slow. The tests reach them by choosing ids' minting instants, and by placing a future-dated file so the real clock is behind.

### D6. Unknown entries live where their id puts them

A preserved unknown entry, of a type this release doesn't know, is placed by its `id` like any other: legacy, root, or its sealed month. On rollover it moves with its month. It counts toward its document's `count`.

**Counting files:** the count check (D7) counts files only in the five known type directories. A newer writer's type directory is invisible to it, so a segment holding such entries has a higher `count` than the known files. The check then parses the segment and finds no known file missing (D7), so it reports no late merge. The cost is parsing that segment, and only for workspaces written by a newer implementation.

### D7. The check: counts first, then documents

`CheckIndex` in a sharded workspace returns an `IndexDiff` with these additions:

| Field | Meaning |
|---|---|
| `LateMerge []string` | ids of known files present in a sealed month's directories and absent from its committed segment |
| `SegmentsMissing` | paths listed in the root whose file is absent, or expected segments the root doesn't list |
| `SegmentsExtra` | paths listed in the root that a rebuild wouldn't produce |
| `RetractedMissing`, `RetractedExtra` | per-document `retracted` differences, as ids |

It works in four steps:
1. Load the graph and build the expected set (D2).
2. For each sealed month, compare the committed `count` with the number of known files in that month's directories, taken from the graph's file paths with no extra I/O. If they match, don't parse the segment. If they differ, parse it, record late-merge ids, and compare it in full.
3. Compare the root and `index/legacy.yaml` in full: entries with the existing `dkf/0.1` MAY-field masking, unknown entries preserved, and `retracted`.
4. Report a segment path mismatch in either direction.

`Clean()` covers every new field. `index --check --json` gains the new lists.

`validate` maps the result to codes:
- the diff → `index_stale`;
- `LateMerge` → one `late_merge` error per file;
- a sealed-retraction pass → `sealed_retraction`. It parses every sealed segment and compares its `retracted` with the list rebuilt from the files, which `validate` has already parsed; the drift check is not where this is caught.

### D8. What stage 1 code goes

`writable()`, `ErrReadOnlyFormat`, the `read_only_format` error code and the `index_unchecked` finding are deleted. The stage 1 tests that asserted refusals become tests that the same verbs succeed and leave a clean index (task 4.1).

## Risks / Trade-offs

- **Incremental and rebuilt indexes drift apart in an edge case.** The sequence test (D4) and the existing "index current after every verb" tests run in both layouts. A single regression shows as `index --check` failing straight after a write.
- **Rollover reads files for the root's retractions.** That's bounded by one or two months of retractions, roughly a handful in the dogfood workspaces.
- **A sealed segment can never be repaired by `particulars index`.** That's deliberate (D3). The message points to `validate`, and the upstream remedy for a late merge is to re-mint.
- **Particulars in sealed months are mutable** (`particular define` adds aliases). Upstream `workspace-layout` says a sealed month's file may change only by gaining a `retracted` block. The index is unaffected, because the entry carries only `id`, `type` and `uri`. But the spec and the format's mutable-particular rule disagree; see Open Questions.

## Migration Plan

Nothing to migrate. No `dkf/0.2` workspace exists outside tests, and `init` still writes `dkf/0.1`. No release until stage 3; see the discussion on #11.

## Open Questions

- **Upstream:** should updating a particular whose file is in a sealed month be allowed? DKF says particulars are "create or update", but `workspace-layout` says a sealed-month file changes only by retraction. This change allows it, since the count and the index are unaffected, and flags the conflict for nodelogicau/particulars. It doesn't block this change.
