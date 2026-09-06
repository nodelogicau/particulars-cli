## MODIFIED Requirements

### Requirement: Document may be a URI or a mapping
`source.document` SHALL accept either a bare reference string or a mapping with `ref` (required), `author` (optional), `hash` (optional), and `quote` (optional), serialised in that order. `ref` holds one of three things: a URI, a path resolved against the workspace root, or an identifier for something that cannot be fetched at all — a conversation, a recollection, a page behind a login. The third case is why the field is not named `uri`: an unfetchable source can still carry a quote, and quoting what someone said is provenance a reviewer can weigh. `author` SHALL identify who produced what was read, as a particular reference per `source-attribution`, distinct from `source.author` (who read it); an unresolvable `document.author` is unresolved, never invalid. Readers SHALL accept `uri` as a legacy alias for `ref` and SHALL report `legacy_document_uri`, because such a file can never be rewritten — appending a retraction is the only permitted modification — so readers must accept it in perpetuity and the warning is the only way anyone learns it is there. A bare URI SHALL remain valid and SHALL NOT be treated as inferior provenance. `quote` SHALL be written as given: an implementation SHALL NOT fold, trim, or otherwise normalise the stored quote. A writer SHALL refuse a quote that is empty or folds to nothing under the whitespace rule of *Drift is two signals*, as it refuses a `confidence` outside [0, 1]; a reader that meets one SHALL treat the mapping as carrying no quote — verifying by hash alone and raising no quote-disclosure note — and SHALL report `empty_quote` as a warning, never `invalid_document` or `quote_drift`, because the file cannot be rewritten and the source never moved. The scalar form SHALL serialise as a scalar, and a mapping carrying only `ref` and `author` SHALL serialise as a mapping, so that a claim written before this capability existed serialises to identical bytes.

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

#### Scenario: A writer is given an empty quote
- **WHEN** `claim assert` or the MCP `claim_assert` tool is given a `quote` that is empty or consists only of whitespace
- **THEN** the write is refused as a usage error

#### Scenario: A reader meets an empty quote
- **WHEN** a claim file carries a `document` mapping whose `quote` folds to nothing and a `hash` that still matches
- **THEN** the claim is valid, `empty_quote` is reported as a warning, and neither `quote_drift` nor `quoted_source` is reported

### Requirement: Drift is two signals
Where a document can be verified, an implementation SHALL report: nothing when the quote is present and the hash matches; `context_drift` when the quote is present and the hash differs; and `quote_drift` when the quote is absent, whether or not the hash differs. When the quote is absent and the hash matches, the message SHALL say that the quote has never been an exact match for the unchanged document, since a quote that is not in a document nobody has edited was miscopied or taken from a different revision. When the mapping carries a quote and no hash, the quote is the whole signal: present SHALL be reported as no drift with the hash unverified (`unverified_document`, informational), and absent as `quote_drift` without the never-matched inference, because nothing says whether the document changed. When the mapping carries a hash and no quote and the hash differs, `context_drift` SHALL be reported with a message that says the document changed and that, with no quote, whether the cited text survived is unknown.

A quote **appears** in a document when, after folding both, the folded quote is a substring of the folded document. Folding SHALL replace every maximal run of characters having the Unicode `White_Space` property with a single U+0020 and strip leading and trailing whitespace, and SHALL alter nothing else; it SHALL NOT fold case, punctuation, or Unicode normalisation form — for everything other than whitespace, the quote agrees with the hash about what constitutes an edit. The stored `quote` SHALL be written verbatim as given: folding is a property of the comparison, never of the file. The rule is format-blind: it removes no blockquote marker, comment prefix, or other syntax from a continuation line, so a quote spanning such a line must be taken from within one line. Drift SHALL be reported as a warning and SHALL NEVER fail validation: a claim whose source has drifted remains valid, readable, and citable.

#### Scenario: The text around a quote changed
- **WHEN** a document is edited elsewhere but the quoted sentence still appears verbatim
- **THEN** `context_drift` is reported as a warning and validation exits 0 if there are no errors

#### Scenario: The quoted text is gone
- **WHEN** the quoted sentence no longer appears in the document and the hash differs
- **THEN** `quote_drift` is reported

#### Scenario: The quote was never in the document
- **WHEN** the quoted text does not appear in the document and the hash still matches
- **THEN** `quote_drift` is reported with a message saying the quote has never matched the unchanged document

#### Scenario: Quote only, still present
- **WHEN** a `document` mapping carries a `quote` and no `hash`, and the quote appears in the document
- **THEN** no drift is reported, and `unverified_document` is reported at informational severity saying the hash was not checked

#### Scenario: Quote only, absent
- **WHEN** a `document` mapping carries a `quote` and no `hash`, and the quote does not appear in the document
- **THEN** `quote_drift` is reported, without any claim about whether the document changed

#### Scenario: Hash only, document changed
- **WHEN** a `document` mapping carries a `hash` and no `quote`, and the hash differs
- **THEN** `context_drift` is reported with a message that does not speak of quoted text

#### Scenario: A block-scalar quote still matches
- **WHEN** a quote written as a YAML block scalar (and therefore carrying a trailing newline) appears verbatim in the document
- **THEN** no drift is reported

#### Scenario: A quote spans a hard line wrap
- **WHEN** a document reads `the billing service listens\non 443` and a claim quotes `the billing service listens on 443`
- **THEN** the quote is present and no drift is reported, and the same holds when the document is re-wrapped at a different column or checked out with CRLF line endings

#### Scenario: A quote spans a blank line
- **WHEN** a document contains two sentences separated by a blank line and a claim quotes both, separated by a single space
- **THEN** the quote is present

#### Scenario: A non-breaking space folds
- **WHEN** a document reads `listens on 443` with U+00A0 between `on` and `443`, and a claim quotes it with U+0020
- **THEN** the quote is present

#### Scenario: An indented code quote
- **WHEN** a claim quotes a tab-indented code block and the document is later re-indented with spaces but otherwise unchanged
- **THEN** the quote is present and `context_drift` is reported, because the hash differs

#### Scenario: Words must still match
- **WHEN** a document reads `listens on 443` and a claim quotes `listens on 8443`
- **THEN** the quote is absent

#### Scenario: Case and punctuation are verbatim
- **WHEN** a document reads `The service listens on 443.` and a claim quotes `the service listens on 443`
- **THEN** the quote is absent

#### Scenario: A wrapped blockquote does not fold
- **WHEN** a document reads `> the billing service listens\n> on 443` and a claim quotes `the billing service listens on 443`
- **THEN** the quote is absent, because folding removes whitespace and not the `>` on the continuation line
