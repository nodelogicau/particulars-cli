## ADDED Requirements

### Requirement: Document verification findings
`validate` SHALL verify each document it can check offline and report, as warnings, `context_drift` when a quote is still present but the document hash differs, and `quote_drift` when the quoted text is absent and the hash differs. It SHALL report `unverified_document` at informational severity for every document it cannot check — a remote URI, an unresolvable path, or a document with no hash. None of these SHALL be an error, and none SHALL change the exit code. Notes SHALL always appear in `--json`; text output SHALL summarise them by a count rather than listing them, unless `--notes` is given, so that a workspace citing mostly remote sources does not bury the findings that need acting on.

#### Scenario: Drift does not fail validation
- **WHEN** a workspace contains a claim whose document has drifted and no errors
- **THEN** `validate` reports the drift and exits 0

#### Scenario: Unverified documents are informational
- **WHEN** every claim cites a remote URI
- **THEN** `validate` reports them as unverified and exits 0

#### Scenario: Notes are counted, not listed
- **WHEN** a workspace produces many `unverified_document` notes and `validate` runs without `--notes`
- **THEN** the text output shows the warnings and a note count, and `--json` still carries every note

#### Scenario: A quoted claim is noted
- **WHEN** a claim carries a quote
- **THEN** `validate` records that the claim reproduces source text verbatim, so a reviewer weighing its scope can see it
