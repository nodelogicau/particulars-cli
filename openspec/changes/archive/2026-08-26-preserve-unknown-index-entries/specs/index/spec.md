## MODIFIED Requirements

### Requirement: Rebuild
`particulars index` SHALL regenerate every entry for the five known types entirely from the object files, and SHALL **preserve entries whose `type` it does not recognise**, unchanged and merged in ascending-id order — a rebuild that dropped them would turn a read-compatibility gap into data loss in the cache, stripping a newer writer's rows every time an older tool touched the workspace. It SHALL report the number of entries written.

#### Scenario: Rebuild after manual merge
- **WHEN** `index.yaml` contains git conflict markers and `particulars index` is run
- **THEN** `index.yaml` is replaced with a valid index reflecting every object file

#### Scenario: Missing index
- **WHEN** `index.yaml` is absent and `particulars index` is run
- **THEN** `index.yaml` is created

#### Scenario: A future record type survives a rebuild
- **WHEN** `index.yaml` carries an entry of a type this implementation does not know and `particulars index` is run
- **THEN** the entry is present in the rebuilt file, unchanged, in ascending-id order among the rest

### Requirement: Check
`particulars index --check` SHALL compare the committed `index.yaml` byte-for-byte with a fresh rebuild — which preserves unknown entry types, so their presence is never drift — and exit with code 4 if they differ, 0 if identical, without modifying the file. A check MUST NOT fail on evidence of a newer conforming writer. In JSON mode the result SHALL list missing, extra, and differing entry ids.

#### Scenario: Drift detected
- **WHEN** a claim file is added by hand without updating the index and `index --check` is run
- **THEN** the command exits with code 4 and names the missing id

#### Scenario: A newer writer's entries are not drift
- **WHEN** `index.yaml` carries entries of an unknown type and is otherwise current
- **THEN** `index --check` exits 0 and `validate` reports no `index_stale`

## ADDED Requirements

### Requirement: Unknown entry types are ignored, not interpreted
Consumers reading `index.yaml` SHALL ignore entries whose `type` they do not recognise: such entries SHALL NOT appear among the typed entries any query consumes, SHALL NOT affect any computed result, and SHALL survive every write path — rebuild and incremental update alike — unchanged.

#### Scenario: Incremental update carries them through
- **WHEN** `claim assert` updates an index that carries an unknown-type entry
- **THEN** the new claim's entry is added and the unknown entry is still present, unchanged
