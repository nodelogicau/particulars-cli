## ADDED Requirements

### Requirement: Index structure
`index.yaml` SHALL contain `format: dkf/0.1` and an `entries` list sorted by ascending id. Each entry SHALL carry `id` and `type`; particulars add `uri`; claims add `subject`, `scope`, `topics` (omitted when empty), `timestamp`; syntheses add the claim fields plus `inputs` as a list of input ids; any retracted object adds `retracted: true`. The file SHALL be serialised deterministically.

#### Scenario: Synthesis entry
- **WHEN** synthesis `syn_C` with inputs A and B and topic `architecture` is indexed
- **THEN** its entry is `{id: syn_C, type: synthesis, subject: <par>, scope: ..., topics: [architecture], timestamp: ..., inputs: [clm_A, clm_B]}`

### Requirement: Incremental maintenance
Every command that creates or modifies an object (`particular define`, `claim assert`, `claim retract`, `synthesis create`) SHALL update `index.yaml` so that it matches a full rebuild immediately afterwards.

#### Scenario: Index current after assert
- **WHEN** `claim assert` succeeds and `index --check` is run immediately
- **THEN** `index --check` exits 0

### Requirement: Rebuild
`particulars index` SHALL regenerate `index.yaml` entirely from the object files, ignoring any existing index content, and report the number of entries written.

#### Scenario: Rebuild after manual merge
- **WHEN** `index.yaml` contains git conflict markers and `particulars index` is run
- **THEN** `index.yaml` is replaced with a valid index reflecting every object file

#### Scenario: Missing index
- **WHEN** `index.yaml` is absent and `particulars index` is run
- **THEN** `index.yaml` is created

### Requirement: Check
`particulars index --check` SHALL compare the committed `index.yaml` byte-for-byte with a fresh rebuild and exit with code 4 if they differ, 0 if identical, without modifying the file. In JSON mode the result SHALL list missing, extra, and differing entry ids.

#### Scenario: Drift detected
- **WHEN** a claim file is added by hand without updating the index and `index --check` is run
- **THEN** the command exits with code 4 and names the missing id

### Requirement: Index is advisory
No command SHALL return incorrect results solely because `index.yaml` is missing or stale; commands MAY use the index to avoid opening files but SHALL fall back to scanning object files when the index is absent.

#### Scenario: Recall with no index
- **WHEN** `index.yaml` is deleted and `recall "Project X"` is run
- **THEN** the command returns the same results as when the index is present
