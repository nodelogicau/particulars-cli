# Verifiable Provenance

## Purpose

Making a claim's evidence checkable: the document union of a bare reference or a mapping carrying a hash and a verbatim quote, how relative references resolve, what a hash is normalised over, the two drift signals and their offline-only computation, unverified sources as a first-class outcome, the retraction `kind` that records which joint in `claim → source → world` broke, and the disclosure a quote carries because it reproduces its source completely.

## Requirements

### Requirement: Document may be a URI or a mapping
`source.document` SHALL accept either a bare reference string or a mapping with `ref` (required), `author` (optional), `hash` (optional), and `quote` (optional), serialised in that order. `ref` holds one of three things: a URI, a path resolved against the workspace root, or an identifier for something that cannot be fetched at all — a conversation, a recollection, a page behind a login. The third case is why the field is not named `uri`: an unfetchable source can still carry a quote, and quoting what someone said is provenance a reviewer can weigh. `author` SHALL identify who produced what was read, as a particular reference per `source-attribution`, distinct from `source.author` (who read it); an unresolvable `document.author` is unresolved, never invalid. Readers SHALL accept `uri` as a legacy alias for `ref` and SHALL report `legacy_document_uri`, because such a file can never be rewritten — appending a retraction is the only permitted modification — so readers must accept it in perpetuity and the warning is the only way anyone learns it is there. A bare URI SHALL remain valid and SHALL NOT be treated as inferior provenance. The scalar form SHALL serialise as a scalar, and a mapping carrying only `ref` and `author` SHALL serialise as a mapping, so that a claim written before this capability existed serialises to identical bytes.

#### Scenario: Scalar form is unchanged
- **WHEN** a claim whose `source.document` is a bare URI string is read and re-serialised
- **THEN** the bytes are identical to the original and no mapping is emitted

#### Scenario: Mapping form
- **WHEN** a claim is written with a document ref, author, hash, and quote
- **THEN** the file carries a `document` mapping with those keys in the order `ref`, `author`, `hash`, `quote`

#### Scenario: Author alone keeps the mapping
- **WHEN** a claim is written with a document ref and author but no hash or quote
- **THEN** the file carries a `document` mapping of `ref` and `author`, not a scalar

#### Scenario: ref is required in the mapping form
- **WHEN** a document mapping carries `hash`, `quote`, or `author` but no `ref`
- **THEN** the object is rejected as invalid

#### Scenario: A legacy uri key is accepted
- **WHEN** a document mapping written before the rename carries `uri`
- **THEN** it is read as `ref`, the object is valid, `legacy_document_uri` is reported, and re-serialising writes `ref`
- **AND** no `non_canonical` warning is reported for the same cause, since the finding that names the cause has already said it

#### Scenario: An unfetchable reference carries a quote
- **WHEN** a document names `chat session 2026-08-22` and carries a quote but no hash
- **THEN** the object is valid and the document reports as unverified

### Requirement: Relative document references
A document `ref` MAY be a path relative to the workspace root rather than an absolute URI, because that is the evidence an agent working in a repository has to hand. Relative references SHALL be resolved against the workspace root when verifying, and SHALL NOT be rewritten on read or write.

#### Scenario: A repo-relative path is accepted and resolved
- **WHEN** a claim cites `docs/architecture.md` and that file exists under the workspace root
- **THEN** the claim is valid and the document is verifiable

#### Scenario: An unresolvable relative path is not an error
- **WHEN** a claim cites a relative path that does not exist
- **THEN** the claim remains valid and the document reports as unverified

### Requirement: Hash normalisation
A document hash SHALL be an algorithm-prefixed digest — `<algorithm>:<lowercase hex>` — taken over the document's bytes with CRLF sequences normalised to LF and **nothing else normalised** — not trailing whitespace, not final newlines, not Unicode form. Line endings are an artefact of transport and differ between checkouts of the same commit; anything else is an edit, and normalising it would hide a class of real change.

#### Scenario: The same content hashes the same on either platform
- **WHEN** the same document is hashed from a CRLF checkout and an LF checkout
- **THEN** both produce the same hash

Writers SHALL write `sha256`. Readers SHALL accept any algorithm and SHALL report one they do not implement as unverified rather than invalid, since refusing it would leave two conformant implementations unable to check each other's hashes. A digest whose algorithm the implementation does implement SHALL be checked for shape, so that a truncated hash is caught as the typo it is.

#### Scenario: An unimplemented algorithm is unverified
- **WHEN** a document carries a hash whose algorithm this implementation does not compute
- **THEN** the document is reported as unverified and the object stays valid

#### Scenario: A truncated sha256 is invalid
- **WHEN** a document carries `sha256:abc`
- **THEN** the object is rejected

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
Verification SHALL NOT make a network request. A document SHALL be verified only when its `ref` resolves to a file within the workspace; every other document — a remote URI, a missing file, a source with no hash, an unfetchable reference such as a conversation — SHALL be reported as `unverified_document` at informational severity. `unverified_document` SHALL NOT be an error and SHALL NOT indicate that anything is wrong: it records that the claim's provenance was not machine-checked.

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

#### Scenario: Drift is never checked against a supersession
- **WHEN** a claim retracted as `supersession` cites a document whose hash is unchanged
- **THEN** no finding is reported about the declaration: drift is a signal about the source, whereas supersession asserts the world moved, and the format cannot tell a document that describes current state from one dated by design

#### Scenario: A defect against a drifted document is unverifiable
- **WHEN** a claim retracted as `defect` cites a document that has since drifted
- **THEN** `defect_unverifiable` is reported, because the text the claim is said to have misread is no longer the text a reviewer can read

#### Scenario: Retracted objects are verified
- **WHEN** validation runs over a workspace containing retracted claims with cited documents
- **THEN** those documents are checked, since the unverifiable-defect finding is about the retraction rather than about a live claim, and any drift is reported as an observation rather than as something to act on

#### Scenario: Unknown kind refused
- **WHEN** a retraction names a kind outside the three values
- **THEN** the operation is refused

### Requirement: A quote discloses its source completely
A quote reproduces source text verbatim inside the claim file, so the claim's effective scope governs that text. Where a synthesis summarises its inputs, a quote discloses its source in full. An implementation SHALL report this when a quoted claim's exposure widens — that is, when it is promoted — and SHALL mention quoted claims during validation. It SHALL NOT refuse the promotion.

#### Scenario: Promoting a quoted claim says so
- **WHEN** a claim carrying a quote is promoted to `public`
- **THEN** the result reports that verbatim source text is being published, and the promotion succeeds
