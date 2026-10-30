## MODIFIED Requirements

### Requirement: Index structure
`index.yaml` SHALL contain `format: dkf/0.1` and an `entries` list sorted by ascending id. Each entry SHALL carry `id` and `type`; particulars add `uri`; claims add `subject`, `scope`, `topics` (omitted when empty), `timestamp`, and `basis` as a list of ids when the claim carries one (omitted otherwise); syntheses add the claim fields plus `inputs` as a list of input ids; merges add `uris` (two) and `timestamp`; promotions add `claims` (the covered ids) and `scope`, plus `timestamp`; any retracted object or record adds `retracted: true`. A claim entry SHALL NOT carry `inputs`, so that "has `inputs`" continues to mean "is a synthesis". Claim and synthesis entries SHALL additionally carry `author` and `document-author` (omitted when empty), serialised after `subject`, mirroring the object's `source.author` and `source.document.author` **as written** — never resolved. A claim or synthesis entry's `scope` SHALL be its **asserted** scope, so that the index remains a faithful cache of the files; effective scope is derived by combining those entries with the promotion entries, which is why promotions carry `claims` and `scope`. The file SHALL be serialised deterministically.

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
