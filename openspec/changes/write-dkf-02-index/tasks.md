## 1. The sharded index model

- [x] 1.1 `Index` gains `Segments []SegmentRef` and `Retracted []string`. Decode them, and encode them in the order root `format, segments, entries, retracted` / segment `format, entries, retracted` / legacy `format, entries`. A `dkf/0.1` document must encode byte-identically to today. Use the workspace's own format for `format` (design D1)
- [x] 1.2 Month helpers: current month of a graph, open/sealed tests, retraction month. Write `BuildSharded(g) ShardedIndex`: entries and retractions placed by D2, unknown entries placed by id (D6), counts, sorted lists, no `retracted` on entries
- [x] 1.3 Unit tests for `BuildSharded`:
  - the upstream shape scenario (Aug/Sep/Oct with a September retraction of an August claim);
  - a retraction-only month;
  - a retraction dated ahead of the newest id;
  - legacy ids;
  - a workspace with no UUIDv7 id;
  - unknown entry types and fields placed and counted

## 2. Rebuild and check

- [x] 2.1 `RebuildIndex` (sharded): write the root and legacy (removing legacy when there are no legacy ids) and the absent sealed segments; never rewrite an existing one; return the differing segment paths. `particulars index` reports them and exits 4 (design D3)
- [x] 2.2 `CheckIndex` (sharded):
  - count-first per sealed month, using the graph's file paths;
  - parse a segment only when its count differs, and report `LateMerge`;
  - full comparison of the root, legacy and differing segments, both entries (with the existing MAY-field masking) and `retracted`;
  - segments missing or extra;
  - extend `IndexDiff` and `Clean()`, and the `index --check --json` output (design D7)
- [x] 2.3 Tests:
  - a rebuild from nothing passes `--check`;
  - a hand-deleted segment is reported missing;
  - a late-merged file is reported, and `index` leaves the sealed segment byte-identical and exits 4;
  - an unparsed segment stays unparsed when its count matches (an instrumented reader, or a corrupt-but-correctly-counted segment that the check never notices);
  - a retraction after commit is reported among retracted ids missing

## 3. Incremental maintenance and guards

- [x] 3.1 `UpsertIndex` (sharded): place the entry by month (root, legacy with count, or no write for a sealed month), add the retraction id to the root, seal on rollover by moving the root's entries and retracted ids (reading those objects' files for their retraction month) into new segments, and fall back to a rebuild when the root is missing or unparseable or a segment being sealed already exists (design D4)
- [x] 3.2 The guards: `currentMonth()` from the month directories; `ErrSealedMonth` in `Create` (UUIDv7 ids) and `Retract`; `apperr` maps it to `sealed_month`, exit 1 (design D5). No store clock was needed: the tests choose the minting instant through the id, and place a future-dated file to make the real clock slow
- [x] 3.3 Sequence test: from a sharded workspace, run scripted mints, retractions (of objects in open and sealed months), and particular updates across gaps of 1, 2 and 5 months, with ids minted at chosen instants. After every step, assert that the documents on disk equal `BuildSharded` of the files, that no existing sealed segment's bytes changed, and that the set of files written is the bounded one: the root, legacy, and new segments
- [x] 3.4 Guard tests: a clock three months slow is refused for mint and retract with `sealed_month` and writes nothing; a clock two days slow across a month boundary is accepted

## 4. Remove the read-only stage and wire validate

- [x] 4.1 Delete `writable()`, `ErrReadOnlyFormat`, the `read_only_format` mapping and `index_unchecked`. Turn `TestShardedIsReadOnly` and `TestShardedWorkspaceIsReadOnlyOverMCP` into tests that every write verb and MCP write tool succeeds in a sharded workspace and leaves `index --check` clean
- [x] 4.2 `validate` (sharded): `index_stale` from the diff, `late_merge` per file naming the segment, and `sealed_retraction` from comparing every sealed segment's `retracted` with the list rebuilt from the files (design D7). Test both codes, including the scenario where a sealed-month retraction passes `--check` and fails `validate`
- [x] 4.3 Every existing `dkf/0.1` index and validate test passes unchanged

## 5. Docs and checks

- [x] 5.1 README: replace the read-only note with the `dkf/0.2` index shape, the sealing rule and the late-merge remedy (re-mint on the branch). Add a `CHANGELOG.md` Unreleased entry that references #11 and says `init` still writes `dkf/0.1`
- [x] 5.2 Run `go test ./...` and `golangci-lint run`
- [ ] 5.3 File the upstream question (nodelogicau/particulars): is updating a particular whose file is in a sealed month allowed? (design Open Questions)
