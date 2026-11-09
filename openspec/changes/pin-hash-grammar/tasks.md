## 1. Grammar

- [x] 1.1 `dkf.ParseHash` under the OCI grammar: registered `sha256`/`sha512` with lengths, `sha-256`/`sha-512` aliases, case-insensitive hex and name, and canonical, non-canonical, and malformed forms.
- [x] 1.2 `dkf.DigestBytes` for `sha256` and `sha512`, with CRLF normalised. `HashDocumentBytes` uses it.

## 2. Write side

- [x] 2.1 `dkf.HashWriteProblem` names the canonical form. `dkf.ValidateForWrite` (every object's source) is run from `store.Create`, and `dkf.SourceForWrite` from `store.Retract`.
- [x] 2.2 Remove the hash checks from `Document.Validate`, so reads and retraction accept any hash.
- [x] 2.3 `--document-hash` help text.

## 3. Read side

- [x] 3.1 Drift: compare by value; a malformed hash means no hash; verify `sha512`.
- [x] 3.2 `validate`: `hash_malformed` per object, and `hash_noncanonical` as a corpus fact with a uniform message.

## 4. Tests and docs

- [x] 4.1 Parser, write-refusal, and digest tests in `internal/dkf`.
- [x] 4.2 Read by value, malformed read as none, retraction of a malformed-hash claim, and `Create` refusals in `internal/query`.
- [x] 4.3 MCP `claim_assert` refusals. A CLI end-to-end test of foreign hashes, aggregation, and the `--document-hash` refusal.
- [x] 4.4 Check that the key tests fail with the change reverted.
- [x] 4.5 `docs/provenance.md` and the CHANGELOG.

## 5. Wrap-up

- [ ] 5.1 Archive and sync specs.
- [ ] 5.2 Close #31 with a summary.
