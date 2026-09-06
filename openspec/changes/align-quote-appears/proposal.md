## Why

The DKF spec accepted the whitespace-folding rule this CLI shipped in `fold-quote-whitespace` (nodelogicau/particulars#24, landed as change `quote-appears`), with corrections the review surfaced. Three of them differ from what the CLI does or says today ([particulars-cli#10](https://github.com/nodelogicau/particulars-cli/issues/10)), and two more states the spec now names are ones the CLI handles by accident rather than by rule. The matcher itself is already conformant; this change aligns the words around it and the edges of the state machine.

## What Changes

- **The whitespace set is named, not listed.** Spec, docs, and the matcher's comment say "the Unicode `White_Space` property" where they said "spaces, tabs, newlines, and blank lines". Go's `strings.Fields` already folds on `unicode.IsSpace`, which is that property, so NBSP, NEL, U+2028, and the U+2000 block fold today; an ASCII reading of the list would disagree on any web-copied quote. The separate line-ending step is dropped from the wording, since CR is whitespace. No matcher change.
- **A quote that folds to nothing is read as no quote, with a warning.** Today `Document.Validate` reports it as `invalid_document`, which is an error. The spec says a reader SHALL treat the mapping as carrying no quote, SHOULD warn, and SHALL NOT reject: claim files are immutable, so a rejection could never be satisfied, and drift is a condition about the source, which never moved. `validate` now verifies such a document by hash alone and reports `empty_quote` as a corpus fact — one line with a count, like `legacy_document_uri`, because the file can never be rewritten. The quote-disclosure note is not raised for it either.
- **MCP `claim_assert` refuses a blank quote,** as `claim assert` already does for `--quote` and a whitespace-only `--quote-file`. Same footing as a `confidence` outside [0, 1].
- **Quote only, quote present: the hash is reported as unverified.** Today the CLI reports nothing. The spec says no drift with the hash unverified, and the CLI's own validation spec already lists "a document with no hash" among what `unverified_document` covers; the code never did. Aggregated under `unverified_document` at informational severity, so it costs one count.
- **The "agrees with the hash" sentence is scoped to non-whitespace** in the docs and the matcher comment. Unscoped, it contradicts the *Trailing whitespace does constitute drift* scenario: whitespace is exactly where the two signals part on purpose.
- **The rule is stated as format-blind.** Folding removes no `>`, `//`, or `#` from a continuation line, so a quote across such a wrap is taken from within one line. Docs and spec, with a blockquote scenario and test.
- **A hash-only document that changed says so plainly.** The `context_drift` message for a mapping with a hash and no quote spoke of "the quoted text"; it now says the document changed and that, with no quote, whether the cited text survived is unknown. Found while walking the read-as-no-quote path; the state itself is unchanged.
- Tests pin: quote-only present → `unverified_document`; quote-only absent → plain `quote_drift`; NBSP folds; a wrapped blockquote is absent; an empty quote in a file → `empty_quote` warning, verified by hash, no `quote_drift`, no `quoted_source`; MCP refuses a blank quote.

Nothing is **BREAKING** for objects. A workspace carrying a whitespace-only quote goes from failing validation to passing with a warning; quote-only claims against local files gain one informational count.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `verifiable-provenance`: the mapping requirement gains the writer-refuses / reader-treats-as-no-quote rule for an empty quote; "Drift is two signals" names `White_Space`, the two no-hash states, the hash-only-differs state, the format-blind limit, and gains scenarios for NBSP, a wrapped blockquote, quote-only present, quote-only absent, and an empty quote in a file.
- `validation`: `empty_quote` joins the corpus facts reported one line per condition; `unverified_document` explicitly covers a quote-only document whose quote was found.
- `claims`: `claim assert` and the MCP `claim_assert` tool refuse an empty or whitespace-only quote as a usage error.

## Impact

- `internal/dkf/document.go` (`Validate`, `QuoteMatches` comment, a `HasQuote` helper), `internal/query/drift.go` (state machine on the effective quote, two messages), `internal/query/validate.go` (`empty_quote` finding, corpus-fact list, disclosure gate), `internal/query/publish.go` or wherever `QuoteDisclosuresForPromotion` lives (disclosure gate), `internal/mcp/server.go` (`documentFrom` refuses a blank quote), their tests, `docs/provenance.md`, `CHANGELOG.md`.
- The skill body needs no change.
