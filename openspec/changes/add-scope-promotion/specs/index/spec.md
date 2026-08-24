## MODIFIED Requirements

### Requirement: Index structure
`index.yaml` SHALL contain `format: dkf/0.1` and an `entries` list sorted by ascending id. Each entry SHALL carry `id` and `type`; particulars add `uri`; claims add `subject`, `scope`, `topics` (omitted when empty), `timestamp`; syntheses add the claim fields plus `inputs` as a list of input ids; merges add `uris` (two) and `timestamp`; promotions add `claims` (the covered ids) and `scope`, plus `timestamp`; any retracted object or record adds `retracted: true`. A claim or synthesis entry's `scope` SHALL be its **asserted** scope, so that the index remains a faithful cache of the files; effective scope is derived by combining those entries with the promotion entries, which is why promotions carry `claims` and `scope`. The file SHALL be serialised deterministically.

#### Scenario: Synthesis entry
- **WHEN** synthesis `syn_C` with inputs A and B and topic `architecture` is indexed
- **THEN** its entry is `{id: syn_C, type: synthesis, subject: <par>, scope: ..., topics: [architecture], timestamp: ..., inputs: [clm_A, clm_B]}`

#### Scenario: Merge entry
- **WHEN** merge `mrg_M` joining two URIs is indexed
- **THEN** its entry is `{id: mrg_M, type: merge, uris: [<a>, <b>], timestamp: ...}`

#### Scenario: Promotion entry
- **WHEN** promotion `pub_P` covering `clm_A` and `syn_B` at `public` is indexed
- **THEN** its entry is `{id: pub_P, type: publish, claims: [clm_A, syn_B], scope: public, timestamp: ...}`

#### Scenario: Effective scope is computable from the index alone
- **WHEN** a consumer reads only `index.yaml`
- **THEN** it can compute every object's effective scope without opening an object file
