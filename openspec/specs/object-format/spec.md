# Object Format

## Purpose

Identifier scheme, file naming, deterministic YAML serialisation, create-only write discipline, and field-level constraints for DKF objects.

## Requirements

### Requirement: Identifier format
Object and record identifiers SHALL have the form `<prefix>_<uuid>` where `prefix` is `par` for particulars, `clm` for claims, `syn` for syntheses, `mrg` for merge records, `pub` for promotion records, and `uuid` is a lowercase, hyphenated, canonical UUID version 7 (RFC 9562). Minting SHALL use a monotonic counter so that identifiers minted by one process are strictly increasing in lexical order. On read, the CLI SHALL accept any identifier matching `^(par|clm|syn|mrg|pub)_[A-Za-z0-9-]+$` so that workspaces written by other implementations remain readable. The canonical pattern SHALL admit every minted prefix, so that an identifier this implementation mints is never reported as `legacy_id`.

#### Scenario: Minted identifier shape
- **WHEN** a claim is created
- **THEN** its id matches `^clm_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`

#### Scenario: Merge identifier shape
- **WHEN** a merge record is created
- **THEN** its id starts with `mrg_` followed by a canonical lowercase UUIDv7

#### Scenario: Promotion identifier shape
- **WHEN** a promotion record is created
- **THEN** its id starts with `pub_` followed by a canonical lowercase UUIDv7, and `validate` reports no `legacy_id` warning for it

### Requirement: File name matches identifier
Each object file SHALL be named `<id>.yaml` and live in the directory corresponding to its prefix; the `id` and `type` fields inside the file SHALL agree with the file name and directory.

#### Scenario: Mismatch detected
- **WHEN** `claims/clm_A.yaml` contains `id: clm_B`
- **THEN** `validate` reports an error for that file

### Requirement: Deterministic serialisation
The CLI SHALL serialise objects with 2-space indentation, keys in spec order (particular: `id, type, uri, label, aliases`; claim: `id, type, subject, content, source, context, timestamp, confidence, retracted`; synthesis: `id, type, subject, content, inputs, unresolved, source, method, timestamp, context, confidence, retracted`; merge: `id, type, uris, reason, source, timestamp, retracted`; `source`: `author, harness, model, document`; `context`: `scope, topics`; input: `id, role, weight`; `retracted`: `timestamp, reason, source, superseded-by`), timestamps as RFC 3339 UTC with a `Z` suffix, multi-line strings as literal block scalars, optional fields omitted when unset, and no document start/end markers. Serialising the same object twice SHALL produce identical bytes. The encoder SHALL never emit a `produced-by` key.

#### Scenario: Stable bytes
- **WHEN** an object is serialised, parsed, and serialised again
- **THEN** both serialisations are byte-identical

#### Scenario: Spec field order
- **WHEN** a claim with all fields set is written
- **THEN** the top-level keys appear in the order `id, type, subject, content, source, context, timestamp, confidence`

#### Scenario: Synthesis field order
- **WHEN** a synthesis with all fields set is written
- **THEN** the top-level keys appear in the order `id, type, subject, content, inputs, unresolved, source, method, timestamp, context, confidence`

### Requirement: Create-only writes
Commands SHALL create new object files with create-exclusive semantics and SHALL fail if the target file already exists. No command SHALL rewrite an existing claim or synthesis file except `claim retract` (which appends only). Particular files MAY be rewritten by `particular define` because particulars are mutable per the format.

#### Scenario: Existing file never overwritten
- **WHEN** a write targets a path that already exists
- **THEN** the command exits with code 1 and the existing file is unchanged

### Requirement: Field constraints
The CLI SHALL enforce on write: `confidence` is a number in [0, 1] when present; `context.scope` is one of `personal`, `organisation`, `public`; `context.topics` is a list of non-empty strings; every `source` block (on claims, syntheses, retractions, and merges) contains at least one of `author` or `harness`, and a synthesis's `source.harness` is present; `timestamp` is a valid RFC 3339 time.

#### Scenario: Confidence out of range
- **WHEN** `claim assert` is run with `--confidence 1.5`
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Invalid scope
- **WHEN** `claim assert` is run with `--scope team`
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Agent-only source accepted
- **WHEN** a claim is written with `source: {harness: claude, model: claude-sonnet-4-6}` and no `author`
- **THEN** the claim is valid

#### Scenario: Source with only a document rejected
- **WHEN** a claim would be written with `source: {document: https://…}` and neither `author` nor `harness`
- **THEN** the command exits with code 2 and writes nothing

### Requirement: Legacy synthesis provenance on read
When a synthesis file carries a `produced-by` block and no `source`, the decoder SHALL populate `source` from `produced-by` (`harness`, `model`) and flag the object as legacy. A file carrying both `source` and `produced-by`, or neither, SHALL be invalid. Legacy objects SHALL behave identically to native ones in every verb; only `validate` distinguishes them.

#### Scenario: v0.1.x synthesis read
- **WHEN** a synthesis written by particulars-cli v0.1.1 (with `produced-by: {harness: claude}`) is recalled
- **THEN** the entry's `source.harness` is `claude` and no `produced-by` key appears in JSON output

#### Scenario: Both blocks present
- **WHEN** a synthesis file carries both `source` and `produced-by`
- **THEN** `validate` reports an error with code `conflicting_provenance`
