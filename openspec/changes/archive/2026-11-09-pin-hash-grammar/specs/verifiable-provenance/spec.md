## MODIFIED Requirements

### Requirement: Hash normalisation
A document hash SHALL be an algorithm-prefixed digest, in the grammar of *Hash grammar and canonical form*, taken over the document's bytes with CRLF sequences normalised to LF and **nothing else normalised** — not trailing whitespace, not final newlines, not Unicode form. Line endings are an artefact of transport and differ between checkouts of the same commit; anything else is an edit, and normalising it would hide a class of real change.

#### Scenario: The same content hashes the same on either platform
- **WHEN** the same document is hashed from a CRLF checkout and an LF checkout
- **THEN** both produce the same hash

Writers SHALL write `sha256`. Readers SHALL accept any algorithm that matches the grammar and SHALL report one they do not implement as unverified rather than invalid, since refusing it would leave two conformant implementations unable to check each other's hashes.

#### Scenario: An unimplemented algorithm is unverified
- **WHEN** a document carries a hash whose algorithm this implementation does not compute
- **THEN** the document is reported as unverified and the object stays valid

#### Scenario: A truncated sha256 is refused on write
- **WHEN** `claim assert --document-hash sha256:abc` is run
- **THEN** the claim is not written, and the refusal says a sha256 digest is 64 lowercase hex digits

#### Scenario: Trailing whitespace changes the hash
- **WHEN** trailing whitespace is added to a line of a document
- **THEN** the hash differs

## ADDED Requirements

### Requirement: Hash grammar and canonical form
A hash SHALL be read under the OCI image-spec digest grammar: `algorithm ":" encoded`, where `algorithm` is components `[a-z0-9]+` joined by single separators from `+`, `.`, `_`, `-`, and `encoded` is `[a-zA-Z0-9=_-]+`. The registered algorithms are `sha256` (64 hex digits) and `sha512` (128); this implementation SHALL verify both. The encoded part of any other algorithm is opaque.

For the registered algorithms a reader SHALL accept hex in either case, the algorithm name in any case, and `sha-256` and `sha-512` in any case, and no other alias. It SHALL compare by algorithm and digest value, case-insensitively in the hex. A hash is **canonical** when it names a registered algorithm in lowercase with lowercase hex of the registered length, or an unregistered algorithm matching the grammar.

A hash SHALL be **malformed** when it does not match the grammar, or names a registered algorithm (after aliases and case) with an encoded part that is not hex or not the registered length. A reader SHALL verify a document whose hash is malformed as though it carried no hash: by its quote alone where it has one, otherwise as unverified. A malformed hash SHALL NOT be reported as drift and SHALL NOT make the object invalid, so an object carrying one can still be retracted.

`claim assert --document-hash`, the MCP `document.hash` and its shorthand, and every other write path SHALL refuse a hash that is not canonical, naming the canonical form (for a non-canonical registered hash, the lowercase hash itself), and SHALL NOT normalise it. `--hash-document` computes canonical `sha256`.

#### Scenario: Uppercase hex from another writer
- **WHEN** a claim file carries a `sha256` hash in uppercase hex whose value matches the document, with a quote that appears in it
- **THEN** no drift is reported

#### Scenario: The IANA spelling
- **WHEN** a claim file carries `SHA-256:` and a matching digest
- **THEN** it is compared as `sha256` and no drift is reported

#### Scenario: An unrecognised alias
- **WHEN** a claim file carries `sha2-256:` and a 64-digit digest
- **THEN** the document is reported as `unverified_document`, not compared as `sha256`

#### Scenario: sha512
- **WHEN** a claim file carries `sha512:` and the document's sha512 digest
- **THEN** no drift is reported; and when the digest differs, the drift rules apply as for `sha256`

#### Scenario: A truncated digest with a quote
- **WHEN** a claim file carries `sha256:9f2a` and a quote that appears in the document
- **THEN** no drift is reported, in particular no `context_drift`, and the hash is reported as unverified

#### Scenario: Base64 under sha256, no quote
- **WHEN** a claim file carries `sha256:` and a base64 digest, with no quote
- **THEN** the document is reported as `unverified_document`

#### Scenario: Retracting an object with a malformed hash
- **WHEN** `claim retract` is run on a claim whose file carries `sha256:9f2a`
- **THEN** the retraction is written

#### Scenario: A writer is given a non-canonical hash
- **WHEN** `claim assert --document-hash` is given `sha256:` and 64 uppercase hex digits, or `SHA-256:` and lowercase ones
- **THEN** the claim is not written, and the refusal names `sha256:` with the lowercase digits
