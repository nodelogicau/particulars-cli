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
`validate` SHALL report an error when a claim or synthesis `subject` does not resolve to a particular file; when a synthesis input id, a claim `basis` id, or a `retracted.superseded-by` id does not resolve to a claim or synthesis file; when a synthesis input or a basis id references a particular, a merge, or a promotion; when the graph formed by synthesis `inputs` and claim `basis` together contains a cycle; when a claim's `basis` lists the claim's own id (`self_basis`); when a `basis` is an empty list (`empty_basis`); when two particulars share a `uri`; or when a merge record's two URIs are equal.

#### Scenario: Dangling subject
- **WHEN** a claim's `subject` is `par_missing`
- **THEN** `validate` reports an error with code `dangling_reference`

#### Scenario: Duplicate URI
- **WHEN** two particular files have the same `uri`
- **THEN** `validate` reports an error with code `duplicate_uri` naming both ids

#### Scenario: Cycle
- **WHEN** hand-edited files make `syn_A` cite `syn_B` and `syn_B` cite `syn_A`
- **THEN** `validate` reports an error with code `cycle`

#### Scenario: Dangling basis id
- **WHEN** a claim's `basis` lists `clm_missing` and no such file exists
- **THEN** `validate` reports a `dangling_reference` error against that claim

#### Scenario: Basis names a particular
- **WHEN** a hand-written claim's `basis` lists a particular id
- **THEN** `validate` reports an error against that claim

#### Scenario: A claim cites itself
- **WHEN** a hand-written claim's `basis` lists its own id
- **THEN** `validate` reports a `self_basis` error and no `cycle` error for the same cause

#### Scenario: Cycle through a basis
- **WHEN** hand-edited files make `clm_A`'s basis list `syn_B`, and `syn_B` cite `clm_A` as an input
- **THEN** `validate` reports an error with code `cycle`

#### Scenario: Empty basis
- **WHEN** a claim file carries `basis: []`
- **THEN** `validate` reports an `empty_basis` error and exits 4, and the claim is still read

### Requirement: Index consistency check
`validate` SHALL report an error with code `index_stale` when `index.yaml` differs from a rebuild, and a warning with code `index_missing` when it is absent.

#### Scenario: Stale index
- **WHEN** a claim file exists that is not in `index.yaml`
- **THEN** `validate` reports `index_stale`

### Requirement: Advisory warnings
`validate` SHALL report a warning (not an error) when: a synthesis cites, directly or transitively, a retracted input (`stale_synthesis`); a non-retracted claim cites a retracted object, directly or transitively through `basis` and `inputs` (`stale_claim`); a claim's `basis` lists the same id more than once (`duplicate_basis`) — writers refuse it, but the format does not forbid it, so a hand-written file is warned rather than failed; a particular has no claims and is neither asserted by nor reported from — a particular whose merge class any object's `source.author` or `source.document.author` resolves to is not an orphan (`orphan_particular`); an object file's serialisation differs from the canonical form (`non_canonical`); a synthesis's provenance was read from a legacy `produced-by` block (`legacy_produced_by`); an id parses under the lenient grammar but is not a canonical UUIDv7 (`legacy_id`); a non-retracted merge names a URI with no local particular (`unknown_merge_uri`); two non-retracted merges join the same pair (`duplicate_merge`); a non-retracted promotion covers a retracted object (`promotion_of_retracted`); two non-retracted promotions name the same object at the same scope (`duplicate_promotion`); a non-retracted synthesis has an **effective** scope wider than the effective scope of any input it cites, or a non-retracted inferred claim has an effective scope wider than the effective scope of any basis member (`scope_wider_than_inputs`).

`scope_wider_than_inputs` SHALL compare effective scope on both sides. It SHALL name the narrower inputs or basis members with their effective scopes, and where an effective scope differs from the asserted one it SHALL name the promotion responsible, so a reader is sent to the file that caused the condition. It SHALL remain a warning — reasoning across scopes is legitimate and no tool can judge whether prose discloses its sources — and SHALL NOT be reported for a retracted synthesis or claim. Because the condition is a property of workspace state rather than of the synthesis or claim file, a promotion SHALL be able to create or clear it without either file changing.

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

#### Scenario: An inferred claim wider than its basis
- **WHEN** a non-retracted `organisation` inferred claim has a basis of one `personal` claim, and no promotions exist
- **THEN** `validate` reports a `scope_wider_than_inputs` warning naming the personal basis member, and exits 0 if there are no errors

#### Scenario: A stale rule
- **WHEN** an inferred claim's basis lists a claim that has since been retracted
- **THEN** `validate` reports a `stale_claim` warning for the inferred claim and exits 0 if there are no errors

#### Scenario: An attributed particular is not an orphan
- **WHEN** a particular has no claims about it but every claim's `author` resolves to it
- **THEN** `validate` reports no `orphan_particular` for it

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
`validate` SHALL verify each document it can check offline and report, as warnings, `context_drift` when a quote is still present but the document hash differs, and `quote_drift` when the quoted text is absent and the hash differs. It SHALL report `unverified_document` at informational severity for every document it cannot fully check — a remote URI, an unresolvable path, a hash whose algorithm it does not implement, or a document with no hash whose quote was found, since the text around a found quote is then unchecked. It SHALL report `legacy_document_uri` as a warning when a document mapping was read from the pre-rename `uri` key, `empty_quote` as a warning when a document mapping carries a `quote` that folds to nothing under the whitespace rule (the mapping is then verified as though it carried no quote), and `defect_unverifiable` when a retraction declaring `defect` cites a document that has since drifted. It SHALL verify retracted objects' documents as well as live ones, since `defect_unverifiable` is about the retraction; drift under a retracted object SHALL be reported as an observation rather than as a warning. None of these SHALL be an error, and none SHALL change the exit code. Findings divide by nature, and the classification is **by condition, never by severity**: **findings about an object** — drift, scope, dangling references, unverifiable defects — list per object whatever their severity, because the object is the unit of action; **facts about the corpus** — the legacy compatibility markers (`legacy_produced_by`, `legacy_id`, `legacy_document_uri`), `empty_quote`, `undeclared`, `confidence_on_undeclared`, and `unverified_document` — SHALL be reported in text output as one line per condition carrying a count, unless `--notes` is given. A condition that straddles the line is classified by whether an object-level action exists, and over-listing errs toward visibility. A corpus fact is permanent and unactionable per object, since the files carrying it can never be rewritten, so a per-object listing recurs on every run forever and buries the findings that need acting on; one line carrying a count discovers exactly as well. `--json` SHALL always carry every finding individually, and aggregation SHALL NOT change any count or the exit code.

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

#### Scenario: An empty quote is a corpus fact
- **WHEN** two claim files carry `document` mappings whose `quote` folds to nothing
- **THEN** text output carries one `empty_quote` line with a count of 2, `--json` carries both findings, and the exit code is unaffected

### Requirement: Forbidden aliases
`validate` SHALL report an error, `forbidden_alias`, for any object file using a YAML anchor or alias. The check runs at the node level, because a struct decode silently expands aliases and the parsed object is indistinguishable from one written plainly; the file remains readable. The prohibition's rationale is resource exhaustion in the file format, not signing: aliases resolve before the data model exists and never affect a signature payload.

#### Scenario: An aliased file is an error
- **WHEN** an object file uses `&anchor` and `*alias` and `validate` runs
- **THEN** `forbidden_alias` is reported as an error, and every query verb still reads the file with the alias expanded

### Requirement: Evidential findings
`validate` SHALL report: `confidence_on_held` as an **error** when a claim carries both `evidential: held` and `confidence` — the file stays readable by every query verb; `undeclared` at informational severity for every claim without an `evidential`, aggregated as a corpus fact, since every workspace written before the field existed carries it on every claim and it can never be cleared; `confidence_on_undeclared` at informational severity when an undeclared claim carries `confidence`, whose meaning cannot be established; `unknown_method` as a warning when a synthesis's `method` is outside the closed vocabulary; `basis_not_inferred` as an **error** when a `basis` appears on a claim whose `evidential` is not `inferred`, on a claim with no `evidential`, or on a synthesis — the file stays readable and citable, and the encoder never writes the field there; and `inferred_without_basis` at informational severity for every `inferred` claim without a `basis`, aggregated as a corpus fact, since no existing claim can be given one. An invalid `evidential` value in a file SHALL be an error.

#### Scenario: Undeclared is aggregate
- **WHEN** a workspace holds 88 claims written before the field existed
- **THEN** text output shows one `undeclared` line carrying the count, `--json` carries every finding, and the exit code is unaffected

#### Scenario: Held with confidence is the one error
- **WHEN** one claim carries `evidential: held` with `confidence: 0.9`
- **THEN** `validate` exits 4 reporting `confidence_on_held`, and `recall` still returns the claim

#### Scenario: Basis on a position
- **WHEN** a claim file declares `evidential: held` and carries a `basis`
- **THEN** `validate` reports `basis_not_inferred` as an error and exits 4, and `recall` still returns the claim

#### Scenario: Basis on a synthesis
- **WHEN** a synthesis file carries a `basis`
- **THEN** `validate` reports `basis_not_inferred`, and the synthesis is still read and citable

#### Scenario: Inferred claims that predate basis
- **WHEN** a workspace holds forty `inferred` claims without a basis
- **THEN** text output shows one `inferred_without_basis` line carrying the count of forty, and the exit code is unaffected

### Requirement: Author reference findings
`validate` SHALL resolve every `source.author` and `source.document.author` value in the workspace per `source-attribution` and report, as facts about the corpus aggregated **per author value**: `author_unresolved` at informational severity when a value resolves to no particular, one line per value carrying the count; and `author_ambiguous` as a warning when a bare name matches more than one particular, one line per value carrying the count and naming the candidate ids. Neither SHALL be reported per object in text output, because the action that clears each — defining the particular, or adding an alias or merge — is at the workspace and clears every occurrence at once; `--json` SHALL carry every finding individually, and `--notes` SHALL list the objects. Neither SHALL be an error or change the exit code.

#### Scenario: Unresolved authors aggregate per value
- **WHEN** ninety objects carry `author: ben` and five carry `author: jane`, neither matching a particular
- **THEN** text output shows one `author_unresolved` line for `ben` with count 90 and one for `jane` with count 5

#### Scenario: Ambiguous author names its candidates
- **WHEN** two particulars carry alias `ben` and 172 objects carry `author: ben`
- **THEN** text output shows one `author_ambiguous` warning line carrying the count and both candidate ids, and the exit code is unaffected

#### Scenario: A workspace-level action clears the condition
- **WHEN** the `author_unresolved` line for `ben` stands and a particular with alias `ben` is defined
- **THEN** the next `validate` run reports no `author_unresolved` for `ben`, no object having changed

### Requirement: URL-in-content note
`validate` SHALL report, at informational severity, `url_in_content` for each claim whose `content` contains a scheme-prefixed URL (`http://` or `https://`) — the catalogue smell: content that names its reading matter rather than stating a fact, when the reference belongs in `source.document`. The condition SHALL be reported as a corpus fact — one aggregate line carrying the count in text output, every finding individually in `--json`, listable with `--notes` — SHALL never be an error, SHALL not change the exit code, and SHALL NOT be reported for syntheses.

#### Scenario: Catalogue-shaped claims aggregate
- **WHEN** forty claims carry an article URL in their content
- **THEN** text output shows one `url_in_content` line with the count, `--json` carries all forty, and the exit code is unaffected

#### Scenario: A legitimate endpoint claim is a note, not a problem
- **WHEN** one claim's content is "The billing API base URL is https://api.example.com/v2"
- **THEN** `url_in_content` is reported for it at informational severity and validation exits 0

#### Scenario: Documents in their place produce no note
- **WHEN** a claim states a fact in content and cites the article in `source.document`
- **THEN** no `url_in_content` is reported

