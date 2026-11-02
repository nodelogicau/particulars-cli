## MODIFIED Requirements

### Requirement: Index structure
`index.yaml` SHALL contain `format: dkf/0.1` and an `entries` list sorted by ascending id. Each entry SHALL carry `id` and `type`; particulars add `uri`; claims add `subject`, `scope`, `topics` (omitted when empty), `timestamp`, and `basis` as a list of ids when the claim carries one (omitted otherwise); syntheses add the claim fields plus `inputs` as a list of input ids; merges add `uris` (two) and `timestamp`; promotions add `claims` (the covered ids) and `scope`, plus `timestamp`; any retracted object or record adds `retracted: true`. A claim entry SHALL NOT carry `inputs`, so that "has `inputs`" continues to mean "is a synthesis". Claim and synthesis entries SHALL additionally carry `author` and `document-author` (omitted when empty), serialised after `subject`, mirroring the object's `source.author` and `source.document.author` **as written** — never resolved. A claim or synthesis entry's `scope` SHALL be its **asserted** scope, so that the index remains a faithful cache of the files; effective scope is derived by combining those entries with the promotion entries, which is why promotions carry `claims` and `scope`. The file SHALL be serialised deterministically.

In a `dkf/0.2` workspace the index SHALL be the documents `workspace-layout` and `index-manifest` define, each carrying `format: dkf/0.2`: the root `index.yaml` with, in order, `format`, `segments` (each `{path, count}`, `index/legacy.yaml` first when it exists, then one per sealed month in ascending order), `entries` (objects minted in an open month), and `retracted` (sorted ids of objects whose `retracted.timestamp` falls in an open month or later); one `index/<YYYY-MM>.yaml` per sealed month that has mints or retractions, with `format`, `entries` (that month's mints) and `retracted` (that month's retractions, `[]` when none); and `index/legacy.yaml`, with `format` and `entries` for ids from which no month derives, present only when such ids exist. The current month SHALL be the month of the newest UUIDv7 id, the open months it and the month before, and every earlier month sealed; a workspace with no UUIDv7 id has no sealed month and its root carries every retraction. A segment's `count` SHALL equal its number of entries. Entries in a `dkf/0.2` index SHALL NOT carry `retracted`; every other entry field is as in `dkf/0.1`. The documents SHALL be a function of the files alone.

#### Scenario: Synthesis entry
- **WHEN** synthesis `syn_C` with inputs A and B and topic `architecture` is indexed
- **THEN** its entry is `{id: syn_C, type: synthesis, subject: <par>, author: …, scope: ..., topics: [architecture], timestamp: ..., inputs: [clm_A, clm_B]}`

#### Scenario: Inferred claim entry
- **WHEN** claim `clm_rule` with `basis: [clm_a, clm_b]` is indexed
- **THEN** its entry carries `basis: [clm_a, clm_b]` after `timestamp` and carries no `inputs`

#### Scenario: Staleness from the index
- **WHEN** a claim entry carries `basis` and one of the listed entries carries `retracted: true`
- **THEN** a consumer can report the claim as stale without opening its file

#### Scenario: Author mirrored as written
- **WHEN** a claim carries `author: ben` unresolved and is indexed
- **THEN** its entry carries `author: ben`, not a URI

#### Scenario: Merge entry
- **WHEN** merge `mrg_M` joining two URIs is indexed
- **THEN** its entry is `{id: mrg_M, type: merge, uris: [<a>, <b>], timestamp: ...}`

#### Scenario: Effective scope is computable from the index alone
- **WHEN** a consumer reads only `index.yaml`
- **THEN** it can compute every object's effective scope without opening an object file

#### Scenario: Shape of a sharded index
- **WHEN** a `dkf/0.2` workspace holds objects minted in August, September and October 2026, the newest in October, and an August claim retracted in September
- **THEN** `index.yaml` lists `{path: index/2026-08.yaml, count: <August's objects>}` under `segments`, carries September's and October's entries, and its `retracted` names the August claim; `index/2026-08.yaml` carries August's entries and `retracted: []`

#### Scenario: A month of retractions only
- **WHEN** objects are retracted in November 2026, none is minted that month, and the newest id is from January 2027
- **THEN** `index/2026-11.yaml` exists with empty `entries` and November's retractions

#### Scenario: No retracted flag on sharded entries
- **WHEN** a claim is retracted in a `dkf/0.2` workspace
- **THEN** its entry carries no `retracted` field and its id is in the `retracted` list of the document for the retraction's month

### Requirement: Incremental maintenance
Every command that creates or modifies an object or record (`particular define`, `particular merge`, `claim assert`, `synthesis create`, `retract` and its `claim retract` alias) SHALL update the index so that it matches a full rebuild immediately afterwards. In a `dkf/0.2` workspace the update SHALL write only what the operation touches: a mint adds its entry to the root, or to `index/legacy.yaml` for a legacy id, and updates that segment's `count`; a retraction adds its id to the root's `retracted`; an update to a particular minted in a sealed month writes no index document, its entry being unchanged. When a mint's month is later than the root's current month, the update SHALL seal every month that has become sealed, moving the root's entries minted in it and the root's retracted ids dated in it into a new `index/<YYYY-MM>.yaml`, adding the segment to `segments`, and rewriting the root; it SHALL NOT rewrite an existing sealed segment. A missing or unparseable root SHALL fall back to a rebuild.

#### Scenario: Index current after assert
- **WHEN** `claim assert` succeeds and `index --check` is run immediately
- **THEN** `index --check` exits 0

#### Scenario: Index current after merge
- **WHEN** `particular merge` succeeds and `index --check` is run immediately
- **THEN** `index --check` exits 0

#### Scenario: An append writes the root only
- **WHEN** `claim assert` succeeds in a `dkf/0.2` workspace whose current month is the clock's month
- **THEN** the claim's file and `index.yaml` are the only files written, and `index --check` exits 0

#### Scenario: A retraction of an old object writes the root only
- **WHEN** a claim whose entry is in `index/2026-08.yaml` is retracted in November 2026
- **THEN** its id is added to the root's `retracted`, `index/2026-08.yaml` is byte-identical, and `index --check` exits 0

#### Scenario: The first mint of a month seals the month before last
- **WHEN** the root holds September and October 2026 and the first November object is minted
- **THEN** `index/2026-09.yaml` is written with September's entries and September's retractions, `segments` gains it, the root holds October and November, and the result equals a rebuild

### Requirement: Rebuild
`particulars index` SHALL regenerate every entry for the five known types entirely from the object files, and SHALL **preserve entries whose `type` it does not recognise**, unchanged and merged in ascending-id order, and SHALL **preserve, on entries it does regenerate, fields it does not recognise**, re-emitted after the known fields in their committed order — a rebuild that dropped either would turn a read-compatibility gap into data loss in the cache, stripping a newer writer's rows or columns every time an older tool touched the workspace. Incremental updates SHALL preserve both the same way. It SHALL report the number of entries written. In a `dkf/0.2` workspace it SHALL write the root and `index/legacy.yaml` (removing the latter when no legacy id remains) and every sealed segment that does not yet exist, and SHALL NOT rewrite a sealed segment that exists: one that differs from what the files produce SHALL be left unchanged and reported by path, saying it is sealed and pointing at `validate`, and the command SHALL exit with code 4. Unknown entry types and fields SHALL be preserved in whichever document their id places them.

#### Scenario: Rebuild after manual merge
- **WHEN** `index.yaml` contains git conflict markers and `particulars index` is run
- **THEN** `index.yaml` is replaced with a valid index reflecting every object file

#### Scenario: A future record type survives a rebuild
- **WHEN** `index.yaml` carries an entry of a type this implementation does not know and `particulars index` is run
- **THEN** the entry is present in the rebuilt file, unchanged, in ascending-id order among the rest

#### Scenario: A future entry field survives a rebuild
- **WHEN** a committed claim entry carries a field this implementation does not know and `particulars index` is run
- **THEN** the rebuilt entry still carries that field with its committed value

#### Scenario: Rebuilding a sharded index from nothing
- **WHEN** a `dkf/0.2` workspace has no index documents and `particulars index` is run
- **THEN** the root, `index/legacy.yaml` if needed, and one segment per sealed month with mints or retractions are written, and `index --check` exits 0

#### Scenario: A differing sealed segment is never rewritten
- **WHEN** `index/2026-08.yaml` lacks a claim present under `claims/2026-08/` and `particulars index` is run
- **THEN** `index/2026-08.yaml` is byte-identical afterwards, the command names it and exits 4, and the root is rewritten

### Requirement: Check
`particulars index --check` SHALL compare the committed `index.yaml` byte-for-byte with a fresh rebuild — which preserves unknown entry types and unknown entry fields, so their presence is never drift — and exit with code 4 if they differ, 0 if identical, without modifying the file. Before comparing, the check SHALL mask from each rebuilt entry any MAY field mirroring an **immutable** property of the object — `scope`, `topics`, `timestamp`, `author`, `document-author` — that is absent from the corresponding committed entry, because the object cannot have changed and the difference can only mean the committed index's writer predated the field. `retracted` SHALL NOT be masked: it mirrors the one mutable property, and its absence means `false`, so an object retracted after the index was committed SHALL fail the check. A MAY field present on both sides with differing values, and any missing or extra entry, SHALL fail the check. A check MUST NOT fail on evidence of a newer conforming writer. In JSON mode the result SHALL list missing, extra, and differing entry ids. In a `dkf/0.2` workspace the check SHALL first compare, for each sealed month, the committed segment's `count` with the number of files of the known types in that month's directories, and SHALL parse that segment only when they differ; files present in the directories and absent from the parsed segment SHALL be reported as a late merge. It SHALL then compare, by the rules above, the root, `index/legacy.yaml`, and every segment whose count differed against their rebuilt counterparts — `entries`, and `retracted` as sets of ids — and SHALL report a segment listed in the root whose file is absent, an expected segment the root does not list, and a listed segment no rebuild produces. A count that matches SHALL NOT be taken to prove anything about the segment's `retracted`. In JSON mode the result SHALL additionally list late-merge ids, missing and extra segments, and retracted ids missing from or extra in the committed documents.

#### Scenario: Drift detected
- **WHEN** a claim file is added by hand without updating the index and `index --check` is run
- **THEN** the command exits with code 4 and names the missing id

#### Scenario: An index committed before the author field passes
- **WHEN** the committed index predates `author`/`document-author`, the files carry authors, and nothing else differs
- **THEN** `index --check` exits 0 and `validate` reports no `index_stale`

#### Scenario: A retraction after the index was committed fails
- **WHEN** a committed entry carries no `retracted` field and its object has since been retracted
- **THEN** `index --check` exits 4 naming that entry, regardless of how many entries lack the field

#### Scenario: A newer writer's entries are not drift
- **WHEN** `index.yaml` carries entries of an unknown type and is otherwise current
- **THEN** `index --check` exits 0 and `validate` reports no `index_stale`

#### Scenario: A sealed segment is compared by count
- **WHEN** objects are appended and retracted in October 2026 and `index --check` runs
- **THEN** `index/2026-08.yaml` is not parsed, and the check exits 0

#### Scenario: A late merge
- **WHEN** `claims/2026-08/clm_X.yaml` arrives by a merge after August has sealed, and `index/2026-08.yaml` does not list it
- **THEN** `index --check` exits 4 and reports `clm_X` as a late merge

#### Scenario: A retraction the root lacks
- **WHEN** an object is retracted after the root was committed
- **THEN** `index --check` exits 4 naming the id among retracted ids missing from the index

## REMOVED Requirements

### Requirement: A dkf/0.2 index is not yet read or written
**Reason**: The CLI now reads, writes, rebuilds and checks the `dkf/0.2` index.
**Migration**: None; the refusal was never released.
