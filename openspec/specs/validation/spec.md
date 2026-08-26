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
`validate` SHALL report a warning (not an error) when: a synthesis cites, directly or transitively, a retracted input (`stale_synthesis`); a particular has no claims (`orphan_particular`); an object file's serialisation differs from the canonical form (`non_canonical`); a synthesis's provenance was read from a legacy `produced-by` block (`legacy_produced_by`); an id parses under the lenient grammar but is not a canonical UUIDv7 (`legacy_id`); a non-retracted merge names a URI with no local particular (`unknown_merge_uri`); two non-retracted merges join the same pair (`duplicate_merge`); a non-retracted promotion covers a retracted object (`promotion_of_retracted`); two non-retracted promotions name the same object at the same scope (`duplicate_promotion`); a non-retracted synthesis has an **effective** scope wider than the effective scope of any input it cites (`scope_wider_than_inputs`).

`scope_wider_than_inputs` SHALL compare effective scope on both sides. It SHALL name the narrower inputs with their effective scopes, and where an effective scope differs from the asserted one it SHALL name the promotion responsible, so a reader is sent to the file that caused the condition. It SHALL remain a warning — reasoning across scopes is legitimate and no tool can judge whether prose discloses its sources — and SHALL NOT be reported for a retracted synthesis. Because the condition is a property of workspace state rather than of the synthesis file, a promotion SHALL be able to create or clear it without either file changing.

#### Scenario: Synthesis wider than its inputs
- **WHEN** a non-retracted `organisation` synthesis cites a `personal` claim and an `organisation` claim, and no promotions exist
- **THEN** `validate` reports a `scope_wider_than_inputs` warning naming the personal claim and not the organisation one, and exits 0 if there are no errors

#### Scenario: Synthesis no wider than its inputs
- **WHEN** a `personal` synthesis cites an `organisation` claim
- **THEN** `validate` reports no `scope_wider_than_inputs` warning

#### Scenario: A promotion clears the condition
- **WHEN** the narrower input of a warned synthesis is promoted to the synthesis's effective scope
- **THEN** `validate` no longer reports `scope_wider_than_inputs` for it, though neither the synthesis nor the claim changed

#### Scenario: A promotion creates the condition
- **WHEN** a synthesis whose inputs match its scope is promoted to `public` and its inputs are not
- **THEN** `validate` reports `scope_wider_than_inputs` for it, naming the promotion that widened it

#### Scenario: Non-canonical file
- **WHEN** a claim file was hand-written with keys in a non-spec order
- **THEN** `validate` reports a `non_canonical` warning and exits 0 if there are no errors

#### Scenario: Legacy synthesis
- **WHEN** a synthesis file written by v0.1.1 carries `produced-by` and no `source`
- **THEN** `validate` reports a `legacy_produced_by` warning and no `non_canonical` warning for the same cause, and exits 0 if there are no errors

#### Scenario: Legacy id
- **WHEN** a claim has id `clm_01j9xk2p3q4r5s6t`
- **THEN** `validate` reports a `legacy_id` warning

### Requirement: Document verification findings
`validate` SHALL verify each document it can check offline and report, as warnings, `context_drift` when a quote is still present but the document hash differs, and `quote_drift` when the quoted text is absent and the hash differs. It SHALL report `unverified_document` at informational severity for every document it cannot check — a remote URI, an unresolvable path, a document with no hash, or a hash whose algorithm it does not implement. It SHALL report `legacy_document_uri` as a warning when a document mapping was read from the pre-rename `uri` key, and `defect_unverifiable` when a retraction declaring `defect` cites a document that has since drifted. It SHALL verify retracted objects' documents as well as live ones, since `defect_unverifiable` is about the retraction; drift under a retracted object SHALL be reported as an observation rather than as a warning. None of these SHALL be an error, and none SHALL change the exit code. Findings divide by nature, and the classification is **by condition, never by severity**: **findings about an object** — drift, scope, dangling references, unverifiable defects — list per object whatever their severity, because the object is the unit of action; **facts about the corpus** — the legacy compatibility markers (`legacy_produced_by`, `legacy_id`, `legacy_document_uri`), `undeclared`, `confidence_on_undeclared`, and `unverified_document` — SHALL be reported in text output as one line per condition carrying a count, unless `--notes` is given. A condition that straddles the line is classified by whether an object-level action exists, and over-listing errs toward visibility. A corpus fact is permanent and unactionable per object, since the files carrying it can never be rewritten, so a per-object listing recurs on every run forever and buries the findings that need acting on; one line carrying a count discovers exactly as well. `--json` SHALL always carry every finding individually, and aggregation SHALL NOT change any count or the exit code.

#### Scenario: Drift does not fail validation
- **WHEN** a workspace contains a claim whose document has drifted and no errors
- **THEN** `validate` reports the drift and exits 0

#### Scenario: Unverified documents are informational
- **WHEN** every claim cites a remote URI
- **THEN** `validate` reports them as unverified and exits 0

#### Scenario: Notes are counted, not listed
- **WHEN** a workspace produces many `unverified_document` notes and `validate` runs without `--notes`
- **THEN** the text output shows one aggregate line per note condition and a note count, and `--json` still carries every note

#### Scenario: Corpus-fact warnings aggregate
- **WHEN** six files carry legacy `produced-by` provenance
- **THEN** text output shows one `legacy_produced_by` line carrying the count of six, the warning count still includes all six, and `--notes` lists each file

#### Scenario: A quoted claim is noted
- **WHEN** a claim carries a quote
- **THEN** `validate` records that the claim reproduces source text verbatim, so a reviewer weighing its scope can see it

### Requirement: Forbidden aliases
`validate` SHALL report an error, `forbidden_alias`, for any object file using a YAML anchor or alias. The check runs at the node level, because a struct decode silently expands aliases and the parsed object is indistinguishable from one written plainly; the file remains readable. The prohibition's rationale is resource exhaustion in the file format, not signing: aliases resolve before the data model exists and never affect a signature payload.

#### Scenario: An aliased file is an error
- **WHEN** an object file uses `&anchor` and `*alias` and `validate` runs
- **THEN** `forbidden_alias` is reported as an error, and every query verb still reads the file with the alias expanded

### Requirement: Evidential findings
`validate` SHALL report: `confidence_on_held` as an **error** when a claim carries both `evidential: held` and `confidence` — the one mechanically checkable rule in this area, and the file stays readable by every query verb; `undeclared` at informational severity for every claim without an `evidential`, aggregated as a corpus fact, since every workspace written before the field existed carries it on every claim and it can never be cleared; `confidence_on_undeclared` at informational severity when an undeclared claim carries `confidence`, whose meaning cannot be established; and `unknown_method` as a warning when a synthesis's `method` is outside the closed vocabulary. An invalid `evidential` value in a file SHALL be an error.

#### Scenario: Undeclared is aggregate
- **WHEN** a workspace holds 88 claims written before the field existed
- **THEN** text output shows one `undeclared` line carrying the count, `--json` carries every finding, and the exit code is unaffected

#### Scenario: Held with confidence is the one error
- **WHEN** one claim carries `evidential: held` with `confidence: 0.9`
- **THEN** `validate` exits 4 reporting `confidence_on_held`, and `recall` still returns the claim
