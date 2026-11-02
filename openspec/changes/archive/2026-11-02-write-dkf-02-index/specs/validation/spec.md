## MODIFIED Requirements

### Requirement: Index consistency check
`validate` SHALL report an error with code `index_stale` when the committed index differs from a rebuild by the rules of `index --check`, and a warning with code `index_missing` when `index.yaml` is absent. In a `dkf/0.2` workspace this SHALL cover the root, `index/legacy.yaml`, and the segment list, and every sealed segment whose count differs from its month's files.

#### Scenario: Stale index
- **WHEN** a claim file exists that is not in `index.yaml`
- **THEN** `validate` reports `index_stale`

#### Scenario: A current sharded index
- **WHEN** `validate` runs in a `dkf/0.2` workspace whose index was written by the CLI
- **THEN** it reports no index finding and exits 0

## ADDED Requirements

### Requirement: Sealed segment checks
In a `dkf/0.2` workspace `validate` SHALL report an error with code `late_merge`, naming the file and the segment, for each file present in a sealed month's directories and absent from that month's committed segment; and SHALL regenerate every sealed month's `retracted` list from the files and report an error with code `sealed_retraction`, naming the object and the segment, for each id present in one of the regenerated and committed lists and absent from the other. A workspace that passes both checks is one in which no sealed segment has gained a mint or a retraction since it was written.

#### Scenario: A late merge fails validation
- **WHEN** a branch open since August 2026 merges `claims/2026-08/clm_X.yaml` in November
- **THEN** `validate` reports `late_merge` naming the file and `index/2026-08.yaml` and exits 4

#### Scenario: A retraction dated into a sealed month
- **WHEN** a file under `claims/2026-08/` gains a `retracted` block dated in August 2026 after August sealed, and the file count is unchanged
- **THEN** `index --check` reports nothing for the segment and `validate` reports `sealed_retraction` naming the claim and `index/2026-08.yaml`

#### Scenario: Matching sealed segments
- **WHEN** every sealed segment's entries and retractions match the files
- **THEN** `validate` reports neither code
