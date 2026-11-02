## MODIFIED Requirements

### Requirement: Structural checks
`validate` SHALL report an error when: `dkf.yaml` is missing or malformed, has an unsupported `format` (code `unsupported_format`, naming the value found and the formats the CLI reads), or has a `workspace.base-uri` not ending in `/`; an object or record file fails to parse as YAML; an object's `id`, `type`, file name, or directory disagree; a required field is missing (`uri`/`label` for particulars; `subject`/`content`/`context.scope`/`timestamp` and a `source` with at least one of `author`/`harness` for claims; additionally `inputs`/`unresolved`/`source.harness` for syntheses; `uris`/`source`/`timestamp` for merges); a synthesis carries both `source` and `produced-by` (`conflicting_provenance`); `scope`, `role`, or `weight` has an invalid value; `confidence` is outside [0, 1]; a timestamp is not RFC 3339; a `retracted` block lacks `timestamp`, `reason`, or a `source` with at least one of `author`/`harness`.

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

#### Scenario: Unsupported format
- **WHEN** `dkf.yaml` has `format: dkf/0.3`
- **THEN** `validate` reports an error for `dkf.yaml` with code `unsupported_format`

### Requirement: Index consistency check
In a `dkf/0.1` workspace, `validate` SHALL report an error with code `index_stale` when `index.yaml` differs from a rebuild, and a warning with code `index_missing` when it is absent. In a `dkf/0.2` workspace, until the CLI reads the `dkf/0.2` index, `validate` SHALL NOT compare the index and SHALL report one warning with code `index_unchecked` saying so.

#### Scenario: Stale index
- **WHEN** a claim file exists that is not in `index.yaml`
- **THEN** `validate` reports `index_stale`

#### Scenario: A sharded index is not yet checked
- **WHEN** `validate` runs in a `dkf/0.2` workspace with no other problems
- **THEN** it reports one `index_unchecked` warning and exits 0

## ADDED Requirements

### Requirement: Layout checks
In a `dkf/0.2` workspace, `validate` SHALL report an error with code `misplaced_file`, naming the file and the path its id derives where the file name is an id, for: a file directly in a type directory; a file in a month directory other than the one its id derives; a file whose id derives no month inside a month directory; a file under `legacy/` whose id derives a month; and a file in a subdirectory that is neither a `YYYY-MM` month nor `legacy`. A misplaced file SHALL NOT be read as an object by any verb. Hidden files and files not ending in `.yaml` SHALL be ignored, as in a `dkf/0.1` workspace.

#### Scenario: A file in the wrong month
- **WHEN** `claims/2026-08/clm_X.yaml` exists and `clm_X` was minted in September 2026
- **THEN** `validate` reports `misplaced_file` naming the file and `claims/2026-09/clm_X.yaml`, and `recall`, like every verb that reads the graph, refuses naming the file until it is moved

#### Scenario: A stray top-level file
- **WHEN** a `dkf/0.2` workspace contains `claims/clm_X.yaml`
- **THEN** `validate` reports `misplaced_file` naming it and its derived path

#### Scenario: A legacy id in a month directory
- **WHEN** `claims/2026-08/clm_07m3zp9s2q1r4t8v.yaml` exists
- **THEN** `validate` reports `misplaced_file` naming `claims/legacy/` as its derived location

#### Scenario: A flat workspace is not checked for shards
- **WHEN** a `dkf/0.1` workspace has a subdirectory under `claims/`
- **THEN** its contents are ignored as before and no `misplaced_file` is reported

### Requirement: Time checks
`validate` SHALL, in either layout, report a warning with code `future_time` naming the object for any id whose UUIDv7 instant is later than the validator's clock and for any object or retraction `timestamp` later than it, and SHALL report an error with code `far_future_id` naming the object for an id whose month is two or more months after the validator's current month. A `future_time` warning SHALL NOT fail validation.

#### Scenario: An id minted on a slightly fast clock
- **WHEN** an id's UUIDv7 timestamp is two days after the validator's clock
- **THEN** `validate` warns `future_time` naming the object and exits 0 if nothing else is wrong

#### Scenario: A future retraction timestamp
- **WHEN** a claim's `retracted.timestamp` is a year after the validator's clock
- **THEN** `validate` warns `future_time` naming the claim

#### Scenario: An id one month ahead
- **WHEN** an id's month is the month after the validator's current month
- **THEN** `validate` warns and does not fail

#### Scenario: A far-future id
- **WHEN** an id's month is two months after the validator's current month
- **THEN** `validate` reports `far_future_id` naming the object and exits 4

#### Scenario: Backdated assertions are not future time
- **WHEN** a claim minted today carries `timestamp: 2023-05-24T00:36:00Z`
- **THEN** no time finding is reported
