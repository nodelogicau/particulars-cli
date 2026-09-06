## ADDED Requirements

### Requirement: An empty quote is refused at assertion
`claim assert` and the MCP `claim_assert` tool SHALL refuse a `quote` that is empty or folds to nothing under the whitespace rule of `verifiable-provenance`, as a usage error naming the field, on the same footing as a `confidence` outside [0, 1]. A `--quote-file` that is empty SHALL be treated as no quote given, since an empty file names nothing to record. Neither SHALL write the claim.

#### Scenario: A whitespace-only quote on the command line
- **WHEN** `claim assert … --document note.md --quote "   "` is run
- **THEN** the command exits 2 and writes nothing

#### Scenario: A whitespace-only quote over MCP
- **WHEN** a client calls `claim_assert` with `source.document` as a mapping whose `quote` is `"\n\t"`
- **THEN** the result is a usage error naming `source.document.quote` and no claim is written
