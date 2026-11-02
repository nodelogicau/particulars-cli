## MODIFIED Requirements

### Requirement: dkf.yaml structure
`dkf.yaml` SHALL contain `format`, `workspace.id` (a UUIDv7 minted at init), optional `workspace.base-uri` (which, when present, SHALL end in `/`), `defaults.scope`, and `defaults.source` with optional `author`, `harness`, `model`. `format` SHALL be one the CLI reads — `dkf/0.1` (the flat layout) or `dkf/0.2` (the sharded layout) — and the CLI SHALL write `dkf/0.1` into a new workspace. Opening a workspace whose `format` is any other value SHALL fail with a message naming `dkf.yaml`, the value found, and the formats the CLI reads. Unknown keys SHALL be preserved on read and ignored. Opening a workspace whose `base-uri` lacks the trailing `/` SHALL fail with a message naming `dkf.yaml`.

#### Scenario: Workspace id minted
- **WHEN** `particulars init` completes
- **THEN** `dkf.yaml` contains a `workspace.id` that is a valid lowercase UUIDv7

#### Scenario: Base URI recorded
- **WHEN** `particulars init --base-uri https://example.com/particulars/` is run
- **THEN** `dkf.yaml` contains `workspace.base-uri: https://example.com/particulars/`

#### Scenario: Hand-edited base URI without slash
- **WHEN** `dkf.yaml` is edited to `base-uri: https://example.com/particulars` and `recall` is run
- **THEN** the command exits with code 1 and the error names `dkf.yaml`

#### Scenario: A format the CLI does not read
- **WHEN** `dkf.yaml` says `format: dkf/0.3` and `recall` is run
- **THEN** the command exits with code 1 and the error names `dkf.yaml`, `dkf/0.3`, and `dkf/0.1, dkf/0.2`

#### Scenario: A sharded workspace opens
- **WHEN** `dkf.yaml` says `format: dkf/0.2` and `recall` is run
- **THEN** the command reads the workspace and succeeds

### Requirement: Directory layout
Objects SHALL be stored one per file in the type directory for their id's prefix — `particulars`, `claims`, `syntheses`, `merges`, or `publishes` — laid out as `dkf.yaml`'s `format` declares, never inferred from the directory contents. In a `dkf/0.1` workspace the file SHALL be `<root>/<type-dir>/<id>.yaml`. In a `dkf/0.2` workspace it SHALL be `<root>/<type-dir>/<YYYY-MM>/<id>.yaml`, where `<YYYY-MM>` is the UTC calendar month of the UUIDv7 timestamp in the id, or `<root>/<type-dir>/legacy/<id>.yaml` when the id's identifier part is not a UUID of version 7. The path SHALL be derived from the id alone. The CLI SHALL create a missing type directory on first write rather than failing.

#### Scenario: Synthesis file location
- **WHEN** a synthesis with id `syn_X` is created in a `dkf/0.1` workspace
- **THEN** the file `<root>/syntheses/syn_X.yaml` exists

#### Scenario: Reading a sharded claim
- **WHEN** a `dkf/0.2` workspace holds `claims/2026-09/clm_01a0f3c1-4d20-7b8e-9a11-6c2f4e7d9b03.yaml`, whose id's UUIDv7 timestamp falls in September 2026, and `recall` names its subject
- **THEN** the claim is returned

#### Scenario: Reading a legacy id
- **WHEN** a `dkf/0.2` workspace holds `claims/legacy/clm_07m3zp9s2q1r4t8v.yaml`
- **THEN** the claim is read like any other

#### Scenario: Assertion time does not move a file
- **WHEN** a claim carrying `timestamp: 2019-03-01T00:00:00Z` has an id minted in September 2026
- **THEN** its derived path is under `claims/2026-09/`

## ADDED Requirements

### Requirement: A dkf/0.2 workspace is read-only in this release
Until the CLI writes the `dkf/0.2` index, every operation that would write in a `dkf/0.2` workspace — creating an object or record, updating a particular, retracting, and rebuilding or updating the index — SHALL refuse before writing anything, with a message saying the release reads `dkf/0.2` but does not write it yet and naming particulars-cli#11. The MCP server's write tools SHALL return the same refusal as an error result. Read verbs, including `export`, which writes derived output outside the workspace, SHALL be unaffected.

#### Scenario: Asserting into a sharded workspace
- **WHEN** `claim assert` is run in a `dkf/0.2` workspace
- **THEN** the command exits non-zero with the read-only message and no file in the workspace changes

#### Scenario: Recalling from a sharded workspace
- **WHEN** `recall` is run in a `dkf/0.2` workspace
- **THEN** it succeeds and returns the workspace's claims
