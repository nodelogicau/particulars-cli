## Why

Upstream `pin-hash-grammar` (nodelogicau/particulars#56, PR #61) pins `source.document.hash` to the OCI digest grammar and separates what writers may write from what readers must accept (#31). The CLI is stricter than the spec on read and, in one place, contradicts it:

- A malformed hash fails `validate` as `invalid_document`. Retraction re-validates the file, so a claim like that can't even be retracted.
- Uppercase hex and `SHA-256:` from another writer are rejected.
- Drift compares hashes as exact strings, so a truncated `sha256` is reported as `context_drift`.

## What Changes

- **`dkf.ParseHash`** classifies a hash as canonical, non-canonical (a registered algorithm in a form readers accept), or malformed, under the OCI grammar. It accepts the `sha-256`/`sha-512` aliases and no others.
- **Writers refuse what isn't canonical.** `store.Create` runs a new `dkf.ValidateForWrite` over the object's source (claims, syntheses, merges, promotions), and `store.Retract` checks the new retraction block's source. Every CLI and MCP write goes through one of the two. It names the canonical form, and for a non-canonical registered hash that is the lowercase hash itself. It never normalises. `Document.Validate`, the read-side and retraction check, no longer looks at the hash.
- **Drift** compares by algorithm and digest value, verifies `sha512` as well as `sha256`, and reads a malformed hash as no hash: by the quote alone, or unverified.
- **`validate`** adds `hash_malformed` (a warning, listed per object) and `hash_noncanonical` (a warning and a corpus fact, aggregated with a uniform message).
- Docs: `docs/provenance.md` gets a table of writer and reader behaviour. The `--document-hash` help names the canonical form.

## Capabilities

### Modified Capabilities

- `verifiable-provenance`: *Hash normalisation* points at the grammar, and a truncated hash is refused on write rather than rejected on read. Adds *Hash grammar and canonical form*.
- `validation`: *Document verification findings* adds `hash_malformed` and `hash_noncanonical`, with the latter among the corpus facts.

## Impact

- `internal/dkf/document.go`, `internal/dkf/validate.go`, `internal/store/workspace.go` (`Create`, `Retract`), `internal/query/drift.go`, `internal/query/validate.go`, and `internal/cli/cmd_claim.go` (help text).
- Tests in `internal/dkf`, `internal/query` (`hash_test.go`), `internal/mcp` (`hash_test.go`), and `internal/cli`.
- Not in scope: the spec's "redacting `source.document.hash` clears `hash_malformed`", which arrives with redaction (#23, `dkf/0.3`).
