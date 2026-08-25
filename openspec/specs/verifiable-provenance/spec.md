# Verifiable Provenance

## Purpose

Making a claim's evidence checkable: the document union of a bare reference or a mapping carrying a hash and a verbatim quote, how relative references resolve, what a hash is normalised over, the two drift signals and their offline-only computation, unverified sources as a first-class outcome, the retraction `kind` that records which joint in `claim → source → world` broke, and the disclosure a quote carries because it reproduces its source completely.

## Requirements

### Requirement: Document may be a URI or a mapping
`source.document` SHALL accept either a bare URI string or a mapping with `uri` (required), `hash` (optional), and `quote` (optional), serialised in that order. A bare URI SHALL remain valid and SHALL NOT be treated as inferior provenance. The scalar form SHALL serialise as a scalar, so that a claim written before this capability existed serialises to identical bytes.

#### Scenario: Scalar form is unchanged
- **WHEN** a claim whose `source.document` is a bare URI string is read and re-serialised
- **THEN** the bytes are identical to the original and no mapping is emitted

#### Scenario: Mapping form
- **WHEN** a claim is written with a document uri, hash, and quote
- **THEN** the file carries a `document` mapping with those three keys in the order `uri`, `hash`, `quote`

#### Scenario: uri is required in the mapping form
- **WHEN** a document mapping carries `hash` or `quote` but no `uri`
- **THEN** the object is rejected as invalid

### Requirement: Relative document references
A document `uri` MAY be a path relative to the workspace root rather than an absolute URI, because that is the evidence an agent working in a repository has to hand. Relative references SHALL be resolved against the workspace root when verifying, and SHALL NOT be rewritten on read or write.

#### Scenario: A repo-relative path is accepted and resolved
- **WHEN** a claim cites `docs/architecture.md` and that file exists under the workspace root
- **THEN** the claim is valid and the document is verifiable

#### Scenario: An unresolvable relative path is not an error
- **WHEN** a claim cites a relative path that does not exist
- **THEN** the claim remains valid and the document reports as unverified

### Requirement: Hash normalisation
A document hash SHALL be `sha256:<lowercase hex>` taken over the document's bytes with CRLF sequences normalised to LF and **nothing else normalised** — not trailing whitespace, not final newlines, not Unicode form. Line endings are an artefact of transport and differ between checkouts of the same commit; anything else is an edit, and normalising it would hide a class of real change.

#### Scenario: The same content hashes the same on either platform
- **WHEN** the same document is hashed from a CRLF checkout and an LF checkout
- **THEN** both produce the same hash

#### Scenario: Trailing whitespace changes the hash
- **WHEN** trailing whitespace is added to a line of a document
- **THEN** the hash differs

### Requirement: Drift is two signals
Where a document can be verified, an implementation SHALL report: nothing when the quote is present and the hash matches; `context_drift` when the quote is present and the hash differs; and `quote_drift` when the quote is absent and the hash differs. Quote matching SHALL normalise line endings and trim the whole quote's leading and trailing whitespace, and SHALL compare internal whitespace verbatim. Drift SHALL be reported as a warning and SHALL NEVER fail validation: a claim whose source has drifted remains valid, readable, and citable.

#### Scenario: The text around a quote changed
- **WHEN** a document is edited elsewhere but the quoted sentence still appears verbatim
- **THEN** `context_drift` is reported as a warning and validation exits 0 if there are no errors

#### Scenario: The quoted text is gone
- **WHEN** the quoted sentence no longer appears in the document and the hash differs
- **THEN** `quote_drift` is reported

#### Scenario: A block-scalar quote still matches
- **WHEN** a quote written as a YAML block scalar (and therefore carrying a trailing newline) appears verbatim in the document
- **THEN** no drift is reported

### Requirement: Verification is offline and best-effort
Verification SHALL NOT make a network request. A document SHALL be verified only when its `uri` resolves to a file within the workspace; every other document — a remote URI, a missing file, a source with no hash, an unfetchable reference such as a conversation — SHALL be reported as `unverified_document` at informational severity. `unverified_document` SHALL NOT be an error and SHALL NOT indicate that anything is wrong: it records that the claim's provenance was not machine-checked.

#### Scenario: A remote URI is not fetched
- **WHEN** a claim cites `https://example.com/doc` with a hash and validation runs with no network
- **THEN** the command succeeds and reports `unverified_document` for that claim

#### Scenario: An unfetchable source is legitimate
- **WHEN** a claim's document is `chat session 2026-08-22`
- **THEN** the claim is valid and reports as unverified rather than invalid

### Requirement: Retraction kind
`retracted` MAY carry `kind`, one of `defect` (the claim misread its source), `supersession` (the source was right then and the world moved on), or `provenance-failure` (the source itself was wrong), serialised after `source` and before `superseded-by`. It SHALL be declared by the caller and SHALL NEVER be inferred from the presence or absence of `superseded-by`: a typo-grade defect routinely carries a replacement, and a claim retracted because its subject was decommissioned is a supersession with nothing to point at. Any other value SHALL be rejected.

#### Scenario: Declared on retraction
- **WHEN** `retract <id> --reason "…" --kind defect` is run
- **THEN** the retracted block carries `kind: defect` between `source` and `superseded-by`

#### Scenario: Kind is optional
- **WHEN** a retraction is written without a kind
- **THEN** the object is valid and no kind key is emitted

#### Scenario: Never inferred
- **WHEN** a retraction carries `superseded-by` and no `kind`
- **THEN** no kind is recorded or implied

#### Scenario: Unknown kind refused
- **WHEN** a retraction names a kind outside the three values
- **THEN** the operation is refused

### Requirement: A quote discloses its source completely
A quote reproduces source text verbatim inside the claim file, so the claim's effective scope governs that text. Where a synthesis summarises its inputs, a quote discloses its source in full. An implementation SHALL report this when a quoted claim's exposure widens — that is, when it is promoted — and SHALL mention quoted claims during validation. It SHALL NOT refuse the promotion.

#### Scenario: Promoting a quoted claim says so
- **WHEN** a claim carrying a quote is promoted to `public`
- **THEN** the result reports that verbatim source text is being published, and the promotion succeeds
