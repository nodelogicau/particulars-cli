## MODIFIED Requirements

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
