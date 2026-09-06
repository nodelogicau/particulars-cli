## 1. One notion of "carries a quote"

- [x] 1.1 Add `func (d Document) HasQuote() bool` to `internal/dkf/document.go` (quote folds to non-empty) and reword the `QuoteMatches` comment: Unicode `White_Space` via `unicode.IsSpace`, "for everything other than whitespace" the quote agrees with the hash, format-blind, no separate line-ending step.
- [x] 1.2 Drop the empty-quote `invalid_document` problem from `Document.Validate`; update the dkf test that expects it, if any.

## 2. Drift state machine

- [x] 2.1 In `internal/query/drift.go`, compute presence from `d.HasQuote()` rather than `d.Quote == ""`; quote-only and present → `CodeUnverifiedDocument` with a message that the quote appears but, with no hash, the text around it is not checked; hash-only and differs → `context_drift` message that does not mention quoted text.
- [x] 2.2 In `internal/query/validate.go`, add `CodeEmptyQuote = "empty_quote"`, add it to `IsCorpusFact`, emit it as a warning for any document mapping with `Quote != "" && !HasQuote()`, and gate the `quoted_source` note on `HasQuote()`.
- [x] 2.3 Gate `QuoteDisclosuresForPromotion` on `HasQuote()`.
- [x] 2.4 Tests in `internal/query` or `internal/cli`: quote-only present → `unverified_document` info; quote-only absent → plain `quote_drift`; hash-only differs → `context_drift` message without "quoted text"; a hand-written claim file with `quote: "  "` and a matching hash → valid, one `empty_quote` warning aggregated in text, present in `--json`, no `quote_drift`, no `quoted_source`; NBSP in the document matches a plain-space quote; a wrapped blockquote is absent.

## 3. Write boundaries

- [x] 3.1 In `internal/mcp/server.go` `documentFrom`, refuse `quote` that is non-empty but folds to nothing with a usage error naming `source.document.quote`; reword the CLI check in `internal/cli/helpers.go` to use the same predicate.
- [x] 3.2 Tests: MCP `claim_assert` with a whitespace-only quote is an error result and writes nothing; CLI `--quote "   "` exits 2; an empty `--quote-file` yields a claim with no quote and no warning.

## 4. Docs and changelog

- [x] 4.1 `docs/provenance.md` "How a quote is matched": name the Unicode `White_Space` property (NBSP folds), scope the "agrees with the hash" sentence to non-whitespace, add the format-blind limit with the blockquote example, and say an empty quote is refused on write and read as no quote with an `empty_quote` warning. Add the quote-only and hash-only rows to the drift table.
- [x] 4.2 `CHANGELOG.md` entry under Unreleased referencing particulars-cli#10 and the upstream change `quote-appears`, noting the `invalid_document` → `empty_quote` downgrade and the new `unverified_document` count for quote-only local claims.

## 5. Verify

- [x] 5.1 `go test ./...` and `golangci-lint run ./...`.
- [x] 5.2 Run `validate` against the knowledge workspace and confirm the only new findings are aggregated `unverified_document` counts and no errors.
