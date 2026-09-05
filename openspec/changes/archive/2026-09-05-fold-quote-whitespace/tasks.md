## 1. Matching rule

- [x] 1.1 Change `dkf.QuoteMatches` in `internal/dkf/document.go` to fold every whitespace run (`unicode.IsSpace`) in both the quote and the document to a single space after CRLF normalisation, keeping the whole-quote trim and the empty-quote refusal; update its doc comment to state the rule and that the stored quote stays verbatim.
- [x] 1.2 In `internal/dkf/dkf_test.go`, invert the "internal whitespace must not be normalised away" assertion and add cases for: a quote spanning a hard wrap, the same document re-wrapped at another column, a CRLF document with a wrapped quote, a quote spanning a blank line, a tab-indented code quote against a space-indented document, and a changed word still failing.

## 2. Drift reporting

- [x] 2.1 In `internal/query/drift.go`, reword the hash-matches-but-quote-absent message to say the quote has never been an exact match for the unchanged document (miscopied or from a different revision), and update the `CodeQuoteDrift` comment.
- [x] 2.2 In `internal/cli/cli_test.go`, add a validate test where the quote spans a wrap: no drift; then re-wrap the document: `context_drift`; then check the never-matched message text for a quote that was never in the file.

## 3. Assert-time warning

- [x] 3.1 In `internal/cli/helpers.go` `resolveDocument`, after the hash step, when `doc.Quote != ""` and the ref resolves via `query.LocalDocumentPath` to a readable file, run `dkf.QuoteMatches` and append a warning to `a.warnings` naming the document and saying `validate` will report `quote_drift` until the quote or the document is corrected; reuse the bytes already read when `--hash-document` was given.
- [x] 3.2 In `internal/mcp` (`server.go` `source` or `tools.go` `claimAssert`), run the same check when the document ref resolves to a workspace file and add a `warnings` array to the `claim_assert` structured result only when non-empty.
- [x] 3.3 Tests: CLI `--json` result carries the warning for a miscopied quote and exits 0; no warning for a wrapped quote; no warning for an unfetchable ref; MCP `claim_assert` result carries the same warning.

## 4. Docs and changelog

- [x] 4.1 In `docs/provenance.md`, add the matching rule under "Why a quote and not a line number" or the drift table: whitespace runs fold on both sides, words must match, the stored quote is verbatim; note that a whitespace-only edit inside the quote reports `context_drift`.
- [x] 4.2 Add a `CHANGELOG.md` entry under the next version referencing issue #9, including that some existing `quote_drift` warnings will disappear and that whitespace-only edits now report `context_drift`.
- [x] 4.3 Confirm `skills/particulars/SKILL.md` needs no change (its "survives insertions and reformatting" line becomes true); if a line is added, regenerate the installed copy per the project's skill-install rule.

## 5. Verify

- [x] 5.1 Run `go test ./...` and `golangci-lint run`.
- [x] 5.2 Re-run the repro from issue #9 against a binary built from HEAD and confirm `validate` reports no drift.
- [ ] 5.3 Open a follow-up issue against `nodelogicau/particulars` proposing the folding rule for the `source-verification` spec, so implementations agree on what "appears" means.
