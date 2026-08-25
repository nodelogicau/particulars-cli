## 1. The document union, without disturbing anything

- [ ] 1.1 **First, and before any feature work**: a test asserting that every existing claim shape round-trips byte-identically, so the union cannot silently rewrite files. Run it against a copy of the real workspace as well as fixtures.
- [ ] 1.2 `internal/dkf/document.go`: `Document{URI, Hash, Quote}` with `UnmarshalYAML` accepting a scalar or a mapping, and `MarshalYAML` emitting a scalar when `Hash` and `Quote` are both empty.
- [ ] 1.3 JSON mirrors YAML: a bare string when only `uri` is set, a mapping otherwise, so `--json` consumers and MCP clients see no change for existing claims.
- [ ] 1.4 `Source.Document` becomes `Document`; `Source.IsZero` accounts for it. Codec emits `document` in the `source` block in its existing position, and the mapping's keys as `uri, hash, quote`.
- [ ] 1.5 Validation: `uri` required in the mapping form; `hash` matches `^sha256:[0-9a-f]{64}$` when present; `quote` non-empty when present.
- [ ] 1.6 Tests: scalar in → scalar out; mapping round-trip; a hash-only or quote-only mapping without `uri` rejected; malformed hash rejected.

## 2. Hashing and quote matching

- [ ] 2.1 `dkf.HashDocument(r io.Reader) string` — sha256 over bytes with CRLF normalised to LF, nothing else, returning `sha256:<hex>`.
- [ ] 2.2 `dkf.QuoteMatches(doc, quote string) bool` — normalise line endings on both, trim the whole quote's leading and trailing whitespace, compare internal whitespace verbatim.
- [ ] 2.3 Tests: the same content hashes identically from CRLF and LF inputs; trailing whitespace changes the hash; a block-scalar quote with its trailing newline matches; an indented code quote matches only with its indentation intact.

## 3. Offline verification

- [ ] 3.1 `internal/query/drift.go`: resolve a document to a local file — a relative path against the workspace root, or a `file:` URI — and return one of `verified`, `context_drift`, `quote_drift`, `unverified`. Never touch the network.
- [ ] 3.2 Wire into `validate`: `context_drift` and `quote_drift` as warnings, `unverified_document` at informational severity, none affecting the exit code.
- [ ] 3.3 Report `unverified_document` once per claim rather than per cause, so a workspace of remote citations does not bury its real findings.
- [ ] 3.4 Tests: drift on an edited local file; no drift when the quote is intact and the hash matches; unverified for a remote URI, a missing file, and a document with no hash; validation still exits 0 with drift present.

## 4. Retraction kind

- [ ] 4.1 `dkf.Retracted.Kind` with the three values, serialised between `source` and `superseded-by`; validation rejects any other value; absent stays absent.
- [ ] 4.2 `retract --kind defect|supersession|provenance-failure`, and the MCP `claim_retract` parameter.
- [ ] 4.3 Tests: key order with and without `superseded-by`; unknown kind refused; kind never inferred from `superseded-by`.
- [ ] 4.4 **Not implemented**: the spec's cross-check of kind against drift. Leave a comment in the validate path naming particulars-cli#3 so the omission is deliberate and findable, not an oversight.

## 5. Surface

- [ ] 5.1 `claim assert --document-hash`, `--hash-document`, `--quote`, `--quote-file`, with the dependency and mutual-exclusion rules; all four are usage errors without `--document`.
- [ ] 5.2 MCP `claim_assert` accepts `source.document` as a string or a mapping.
- [ ] 5.3 Promotion reports quoted claims: extend the `publish` and `knowledge_publish` result the way `scope_wider_than_inputs` findings are already carried.
- [ ] 5.4 In-process CLI tests: assert with a hash and quote; `--hash-document` against a local file; `--quote` without `--document` refused; retract with a kind; publish a quoted claim and see the disclosure.

## 6. Documentation

- [ ] 6.1 `skills/particulars/SKILL.md`: replace the `--document src/billing/cron.go:14` example — a line offset, which the spec rejects because it breaks when text is inserted above it — with `--quote`. Say what to quote: the sentence that supports the claim, not the section containing it. Note that a quote discloses its source completely, unlike a synthesis. Regenerate the `.claude` copy.
- [ ] 6.2 `docs/review-workflow.md`: a reviewer can audit a claim against its quote without leaving the pull request, and drift tells them when the ground moved under a claim they already approved.
- [ ] 6.3 New `docs/provenance.md`: the two forms, what the hash is taken over and why LF-only, the two drift signals, why verification is offline, and what `kind` records.
- [ ] 6.4 README verb table and CHANGELOG.

## 7. Verification

- [ ] 7.1 `go build ./... && go vet ./... && golangci-lint run && go test ./...` clean.
- [ ] 7.2 Against a copy of the real workspace: `validate` reports the same errors and warnings as before this change, plus `unverified_document` for the remote citations and nothing else. **No object file is rewritten** — compare the digest before and after.
- [ ] 7.3 Add a quote and hash to one claim citing a local file, edit that file, and confirm `context_drift`; delete the quoted sentence and confirm `quote_drift`.
