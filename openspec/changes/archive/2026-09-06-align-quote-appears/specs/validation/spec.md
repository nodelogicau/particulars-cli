## MODIFIED Requirements

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
