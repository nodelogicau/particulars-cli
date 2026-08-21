## MODIFIED Requirements

### Requirement: Index structure
`index.yaml` SHALL contain `format: dkf/0.1` and an `entries` list sorted by ascending id. Each entry SHALL carry `id` and `type`; particulars add `uri`; claims add `subject`, `scope`, `topics` (omitted when empty), `timestamp`; syntheses add the claim fields plus `inputs` as a list of input ids; merges add `uris` (two) and `timestamp`; any retracted object or record adds `retracted: true`. The file SHALL be serialised deterministically.

#### Scenario: Synthesis entry
- **WHEN** synthesis `syn_C` with inputs A and B and topic `architecture` is indexed
- **THEN** its entry is `{id: syn_C, type: synthesis, subject: <par>, scope: ..., topics: [architecture], timestamp: ..., inputs: [clm_A, clm_B]}`

#### Scenario: Merge entry
- **WHEN** merge `mrg_M` joining two URIs is indexed
- **THEN** its entry is `{id: mrg_M, type: merge, uris: [<a>, <b>], timestamp: ...}`

### Requirement: Incremental maintenance
Every command that creates or modifies an object or record (`particular define`, `particular merge`, `claim assert`, `synthesis create`, `retract` and its `claim retract` alias) SHALL update `index.yaml` so that it matches a full rebuild immediately afterwards.

#### Scenario: Index current after assert
- **WHEN** `claim assert` succeeds and `index --check` is run immediately
- **THEN** `index --check` exits 0

#### Scenario: Index current after merge
- **WHEN** `particular merge` succeeds and `index --check` is run immediately
- **THEN** `index --check` exits 0
