## MODIFIED Requirements

### Requirement: Index structure
`index.yaml` SHALL contain `format: dkf/0.1` and an `entries` list sorted by ascending id. Each entry SHALL carry `id` and `type`; particulars add `uri`; claims add `subject`, `scope`, `topics` (omitted when empty), `timestamp`; syntheses add the claim fields plus `inputs` as a list of input ids; merges add `uris` (two) and `timestamp`; promotions add `claims` (the covered ids) and `scope`, plus `timestamp`; any retracted object or record adds `retracted: true`. Claim and synthesis entries SHALL additionally carry `author` and `document-author` (omitted when empty), serialised after `subject`, mirroring the object's `source.author` and `source.document.author` **as written** — never resolved. A claim or synthesis entry's `scope` SHALL be its **asserted** scope, so that the index remains a faithful cache of the files; effective scope is derived by combining those entries with the promotion entries, which is why promotions carry `claims` and `scope`. The file SHALL be serialised deterministically.

#### Scenario: Synthesis entry
- **WHEN** synthesis `syn_C` with inputs A and B and topic `architecture` is indexed
- **THEN** its entry is `{id: syn_C, type: synthesis, subject: <par>, author: …, scope: ..., topics: [architecture], timestamp: ..., inputs: [clm_A, clm_B]}`

#### Scenario: Author mirrored as written
- **WHEN** a claim carries `author: ben` unresolved and is indexed
- **THEN** its entry carries `author: ben`, not a URI

#### Scenario: Merge entry
- **WHEN** merge `mrg_M` joining two URIs is indexed
- **THEN** its entry is `{id: mrg_M, type: merge, uris: [<a>, <b>], timestamp: ...}`

#### Scenario: Effective scope is computable from the index alone
- **WHEN** a consumer reads only `index.yaml`
- **THEN** it can compute every object's effective scope without opening an object file

### Requirement: Rebuild
`particulars index` SHALL regenerate every entry for the five known types entirely from the object files, and SHALL **preserve entries whose `type` it does not recognise**, unchanged and merged in ascending-id order, and SHALL **preserve, on entries it does regenerate, fields it does not recognise**, re-emitted after the known fields in their committed order — a rebuild that dropped either would turn a read-compatibility gap into data loss in the cache, stripping a newer writer's rows or columns every time an older tool touched the workspace. Incremental updates SHALL preserve both the same way. It SHALL report the number of entries written.

#### Scenario: Rebuild after manual merge
- **WHEN** `index.yaml` contains git conflict markers and `particulars index` is run
- **THEN** `index.yaml` is replaced with a valid index reflecting every object file

#### Scenario: A future record type survives a rebuild
- **WHEN** `index.yaml` carries an entry of a type this implementation does not know and `particulars index` is run
- **THEN** the entry is present in the rebuilt file, unchanged, in ascending-id order among the rest

#### Scenario: A future entry field survives a rebuild
- **WHEN** a committed claim entry carries a field this implementation does not know and `particulars index` is run
- **THEN** the rebuilt entry still carries that field with its committed value

### Requirement: Check
`particulars index --check` SHALL compare the committed `index.yaml` byte-for-byte with a fresh rebuild — which preserves unknown entry types and unknown entry fields, so their presence is never drift — and exit with code 4 if they differ, 0 if identical, without modifying the file. Before comparing, the check SHALL mask from each rebuilt entry any MAY field mirroring an **immutable** property of the object — `scope`, `topics`, `timestamp`, `author`, `document-author` — that is absent from the corresponding committed entry, because the object cannot have changed and the difference can only mean the committed index's writer predated the field. `retracted` SHALL NOT be masked: it mirrors the one mutable property, and its absence means `false`, so an object retracted after the index was committed SHALL fail the check. A MAY field present on both sides with differing values, and any missing or extra entry, SHALL fail the check. A check MUST NOT fail on evidence of a newer conforming writer. In JSON mode the result SHALL list missing, extra, and differing entry ids.

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
