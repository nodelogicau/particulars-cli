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
`validate` SHALL report an error when: `dkf.yaml` is missing or malformed, has an unsupported `format`, or has a `workspace.base-uri` not ending in `/`; an object or record file fails to parse as YAML; an object's `id`, `type`, file name, or directory disagree; a required field is missing (`uri`/`label` for particulars; `subject`/`content`/`context.scope`/`timestamp` and a `source` with at least one of `author`/`harness` for claims; additionally `inputs`/`unresolved`/`source.harness` for syntheses; `uris`/`source`/`timestamp` for merges); a synthesis carries both `source` and `produced-by` (`conflicting_provenance`); `scope`, `role`, or `weight` has an invalid value; `confidence` is outside [0, 1]; a timestamp is not RFC 3339; a `retracted` block lacks `timestamp`, `reason`, or a `source` with at least one of `author`/`harness`.

#### Scenario: Missing unresolved
- **WHEN** a synthesis file has no `unresolved` field
- **THEN** `validate` reports an error with code `missing_field` for that file

#### Scenario: Invalid role
- **WHEN** a synthesis input has `role: support`
- **THEN** `validate` reports an error with code `invalid_enum`

#### Scenario: Source without author or harness
- **WHEN** a claim file has `source: {document: x}` only
- **THEN** `validate` reports an error with code `missing_field` naming `source`

#### Scenario: Base URI without trailing slash
- **WHEN** `dkf.yaml` has `base-uri: https://example.com/particulars`
- **THEN** `validate` reports an error for `dkf.yaml` with code `invalid_base_uri`

### Requirement: Referential checks
`validate` SHALL report an error when a claim or synthesis `subject` does not resolve to a particular file; when a synthesis input id or a `retracted.superseded-by` id does not resolve to a claim or synthesis file; when a synthesis input references a particular or a merge; when the synthesis input graph contains a cycle; when two particulars share a `uri`; or when a merge record's two URIs are equal.

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
`validate` SHALL report a warning (not an error) when: a synthesis cites, directly or transitively, a retracted input (`stale_synthesis`); a particular has no claims (`orphan_particular`); an object file's serialisation differs from the canonical form (`non_canonical`); a synthesis's provenance was read from a legacy `produced-by` block (`legacy_produced_by`); an id parses under the lenient grammar but is not a canonical UUIDv7 (`legacy_id`); a non-retracted merge names a URI with no local particular (`unknown_merge_uri`); two non-retracted merges join the same pair (`duplicate_merge`); a non-retracted synthesis declares a scope wider than one of its direct inputs (`scope_wider_than_inputs`).

Scope orders `personal` < `organisation` < `public`. The warning exists because scope is declared per assertion and is not inherited: a synthesis's content can summarise assertions that are withheld from the audience the synthesis is shared with. It SHALL name the narrower inputs and their scopes so a reviewer can judge whether the summary discloses them. It SHALL remain a warning — reasoning across scopes is legitimate, and the judgement is the reviewer's — and SHALL NOT be reported for a retracted synthesis.

#### Scenario: Non-canonical file
- **WHEN** a claim file was hand-written with keys in a non-spec order
- **THEN** `validate` reports a `non_canonical` warning and exits 0 if there are no errors

#### Scenario: Legacy synthesis
- **WHEN** a synthesis file written by v0.1.1 carries `produced-by` and no `source`
- **THEN** `validate` reports a `legacy_produced_by` warning and no `non_canonical` warning for the same cause, and exits 0 if there are no errors

#### Scenario: Synthesis wider than its inputs
- **WHEN** a non-retracted `organisation` synthesis cites a `personal` claim and an `organisation` claim
- **THEN** `validate` reports a `scope_wider_than_inputs` warning naming the personal claim and not the organisation one, and exits 0 if there are no errors

#### Scenario: Synthesis no wider than its inputs
- **WHEN** a `personal` synthesis cites an `organisation` claim
- **THEN** `validate` reports no `scope_wider_than_inputs` warning

#### Scenario: Legacy id
- **WHEN** a claim has id `clm_01j9xk2p3q4r5s6t`
- **THEN** `validate` reports a `legacy_id` warning
