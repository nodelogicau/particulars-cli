## ADDED Requirements

### Requirement: A quote absent from a local document is reported at assertion
When `claim assert` is given a quote and the document resolves to a readable file in the workspace, the command SHALL check the quote against that file using the matching rule in `verifiable-provenance`, and when the quote is absent SHALL still write the claim and SHALL carry a warning in the result naming the document and saying `validate` will report `quote_drift` until the quote or the document is corrected. The check SHALL NOT fetch anything and SHALL NOT run when the document does not resolve to a workspace file. The MCP `claim_assert` tool SHALL carry the same warning in its structured result.

#### Scenario: A miscopied quote is caught at assertion
- **WHEN** `claim assert … --document note.md --quote "listens on 8443"` is run and `note.md` reads `listens on 443`
- **THEN** the claim is written, the command exits 0, and the result carries a warning that the quote does not appear in `note.md`

#### Scenario: A wrapped quote is not warned about
- **WHEN** `claim assert … --document note.md --hash-document --quote "the billing service listens on 443"` is run and `note.md` reads `the billing service listens\non 443`
- **THEN** the claim is written with no warning

#### Scenario: An unfetchable source is not checked
- **WHEN** `claim assert … --document "conversation with Jane" --quote "we went microservices in Q2"` is run
- **THEN** the claim is written with no warning about the quote
