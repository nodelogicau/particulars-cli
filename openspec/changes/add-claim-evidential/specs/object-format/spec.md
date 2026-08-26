## MODIFIED Requirements

### Requirement: Deterministic serialisation
The CLI SHALL serialise objects with 2-space indentation, keys in spec order (particular: `id, type, uri, label, aliases`; claim: `id, type, subject, content, source, context, timestamp, evidential, confidence, retracted`; synthesis: `id, type, subject, content, inputs, unresolved, source, method, timestamp, context, confidence, retracted`; merge: `id, type, uris, reason, source, timestamp, retracted`; promotion: `id, type, claims, scope, reason, source, timestamp, retracted`; `source`: `author, harness, model, document`; `document` in its mapping form: `ref, hash, quote`; `context`: `scope, topics`; input: `id, role, weight`; `retracted`: `timestamp, reason, source, kind, superseded-by`), timestamps as RFC 3339 UTC with a `Z` suffix, multi-line strings as literal block scalars, optional fields omitted when unset, and no document start/end markers. A `source.document` that carries only a URI SHALL be serialised as a scalar, not as a single-key mapping. Serialising the same object twice SHALL produce identical bytes. The encoder SHALL never emit a `produced-by` key.

#### Scenario: Byte-stable round trip
- **WHEN** any object file is read and re-serialised
- **THEN** the bytes are identical

#### Scenario: A scalar document stays scalar
- **WHEN** a claim written before the mapping form existed is read and re-serialised
- **THEN** `document` is emitted as a scalar and the file is byte-identical

#### Scenario: Retraction key order
- **WHEN** a retraction carrying both `kind` and `superseded-by` is serialised
- **THEN** the keys appear in the order `timestamp, reason, source, kind, superseded-by`

#### Scenario: Evidential position
- **WHEN** a claim carrying `evidential` and `confidence` is serialised
- **THEN** `evidential` appears between `timestamp` and `confidence`, and a claim without the field emits no `evidential` key
