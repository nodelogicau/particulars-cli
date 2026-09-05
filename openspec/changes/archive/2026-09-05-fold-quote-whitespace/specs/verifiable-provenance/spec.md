## MODIFIED Requirements

### Requirement: Drift is two signals
Where a document can be verified, an implementation SHALL report: nothing when the quote is present and the hash matches; `context_drift` when the quote is present and the hash differs; and `quote_drift` when the quote is absent, whether or not the hash differs. When the quote is absent and the hash matches, the message SHALL say that the quote has never been an exact match for the unchanged document, since a quote that is not in a document nobody has edited was miscopied or taken from a different revision. Quote matching SHALL normalise line endings, trim the whole quote's leading and trailing whitespace, and fold every run of whitespace — spaces, tabs, newlines, and blank lines — in both the quote and the document to a single space before comparing; it SHALL NOT fold case, Unicode form, or punctuation. The stored `quote` SHALL be written verbatim as given: folding is a property of the comparison, never of the file. Drift SHALL be reported as a warning and SHALL NEVER fail validation: a claim whose source has drifted remains valid, readable, and citable.

#### Scenario: The text around a quote changed
- **WHEN** a document is edited elsewhere but the quoted sentence still appears verbatim
- **THEN** `context_drift` is reported as a warning and validation exits 0 if there are no errors

#### Scenario: The quoted text is gone
- **WHEN** the quoted sentence no longer appears in the document and the hash differs
- **THEN** `quote_drift` is reported

#### Scenario: The quote was never in the document
- **WHEN** the quoted text does not appear in the document and the hash still matches
- **THEN** `quote_drift` is reported with a message saying the quote has never matched the unchanged document

#### Scenario: A block-scalar quote still matches
- **WHEN** a quote written as a YAML block scalar (and therefore carrying a trailing newline) appears verbatim in the document
- **THEN** no drift is reported

#### Scenario: A quote spans a hard line wrap
- **WHEN** a document reads `the billing service listens\non 443` and a claim quotes `the billing service listens on 443`
- **THEN** the quote is present and no drift is reported, and the same holds when the document is re-wrapped at a different column or checked out with CRLF line endings

#### Scenario: A quote spans a blank line
- **WHEN** a document contains two sentences separated by a blank line and a claim quotes both, separated by a single space
- **THEN** the quote is present

#### Scenario: An indented code quote
- **WHEN** a claim quotes a tab-indented code block and the document is later re-indented with spaces but otherwise unchanged
- **THEN** the quote is present and `context_drift` is reported, because the hash differs

#### Scenario: Words must still match
- **WHEN** a document reads `listens on 443` and a claim quotes `listens on 8443`
- **THEN** the quote is absent
