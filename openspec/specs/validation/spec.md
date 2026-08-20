# Validation

## Purpose

Workspace-wide structural, referential, and index-consistency checks with error/warning findings and a check-failed exit code.

## Requirements

### Requirement: Validate a workspace
`particulars validate` SHALL examine the whole workspace and emit a list of findings, each with `severity` (`error` or `warning`), `path`, `code`, and `message`. It SHALL exit with code 4 if any finding has severity `error`, otherwise 0. It SHALL never modify files.

#### Scenario: Clean workspace
- **WHEN** `validate` is run on a workspace produced solely by the CLI
- **THEN** no findings are emitted and the exit code is 0

#### Scenario: JSON findings
- **WHEN** `validate --json` is run on a workspace with problems
- **THEN** stdout contains `{"findings": [{"severity": ..., "path": ..., "code": ..., "message": ...}, ...]}` and the exit code is 4

### Requirement: Structural checks
`validate` SHALL report an error when: `dkf.yaml` is missing or malformed or has an unsupported `format`; an object file fails to parse as YAML; an object's `id`, `type`, file name, or directory disagree; a required field is missing (`uri`/`label` for particulars; `subject`/`content`/`source.author`/`context.scope`/`timestamp` for claims; additionally `inputs`/`unresolved`/`produced-by.harness` for syntheses); `scope`, `role`, or `weight` has an invalid value; `confidence` is outside [0, 1]; a timestamp is not RFC 3339; a `retracted` block lacks `timestamp`, `reason`, or `source.author`.

#### Scenario: Missing unresolved
- **WHEN** a synthesis file has no `unresolved` field
- **THEN** `validate` reports an error with code `missing_field` for that file

#### Scenario: Invalid role
- **WHEN** a synthesis input has `role: support`
- **THEN** `validate` reports an error with code `invalid_enum`

### Requirement: Referential checks
`validate` SHALL report an error when a claim or synthesis `subject` does not resolve to a particular file; when a synthesis input id or a `retracted.superseded-by` id does not resolve to a claim or synthesis file; when a synthesis input references a particular; when the synthesis input graph contains a cycle; or when two particulars share a `uri`.

#### Scenario: Dangling subject
- **WHEN** a claim's `subject` is `par_missing`
- **THEN** `validate` reports an error with code `dangling_reference`

#### Scenario: Duplicate URI
- **WHEN** two particular files have the same `uri`
- **THEN** `validate` reports an error with code `duplicate_uri` naming both ids

#### Scenario: Cycle
- **WHEN** hand-edited files make `syn_A` cite `syn_B` and `syn_B` cite `syn_A`
- **THEN** `validate` reports an error with code `cycle`

### Requirement: Index consistency check
`validate` SHALL report an error with code `index_stale` when `index.yaml` differs from a rebuild, and a warning with code `index_missing` when it is absent.

#### Scenario: Stale index
- **WHEN** a claim file exists that is not in `index.yaml`
- **THEN** `validate` reports `index_stale`

### Requirement: Advisory warnings
`validate` SHALL report a warning (not an error) when: a synthesis cites a retracted input (`stale_synthesis`); a particular has no claims (`orphan_particular`); an object file's serialisation differs from the canonical form (`non_canonical`).

#### Scenario: Non-canonical file
- **WHEN** a claim file was hand-written with keys in a non-spec order
- **THEN** `validate` reports a `non_canonical` warning and exits 0 if there are no errors
