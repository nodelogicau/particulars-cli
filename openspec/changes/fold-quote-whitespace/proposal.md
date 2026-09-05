## Why

`validate` reports `quote_drift` for a quote whose words are verbatim in the document but which spans a hard line wrap ([particulars-cli#9](https://github.com/nodelogicau/particulars-cli/issues/9)). The matcher is an exact substring test, so prose wrapped at 80 columns — Markdown, plain text, commit messages, most READMEs — cannot be quoted across a line boundary even when the hash matches. `docs/provenance.md` promises a quote "survives insertion and reformatting"; re-wrapping is reformatting, and today the promise is false. The current rule is deliberate — the spec says internal whitespace is compared verbatim, so that indentation inside a quoted code block is part of what was quoted — but folding whitespace on both sides never loses a code match, and the only thing it gives up is telling a whitespace-only edit inside the quote apart from any other edit, which the hash reports anyway.

## What Changes

- **Quote matching folds whitespace.** Every run of whitespace — spaces, tabs, newlines, blank lines — in both the quote and the document is folded to a single space before the substring test. Line-ending normalisation and whole-quote trimming stay. The stored `quote` is written verbatim as given; only the comparison changes.
- **The undefined drift state gets a name in the spec and a straight message.** A quote absent from a document whose hash still matches is not a state the "Drift is two signals" table defined; the CLI reports it as `quote_drift` with a message that reads as a contradiction. The spec now says it is `quote_drift`, and the message says what it means: the quote was never an exact match of this document.
- **`claim assert` and `claim_assert` warn when the quote is not in a local document.** When the document resolves to a file in the workspace and the quote is not found, the claim is written and the result carries a warning. Today a miscopied quote is discovered at the next `validate`, by someone who may not be the writer.
- Tests: a quote spanning a wrap, a CRLF wrapped quote, a quote spanning a blank line, an indented code quote, and the assert-time warning. The existing test asserting that internal whitespace is *not* normalised is inverted.
- `docs/provenance.md` states the matching rule; `CHANGELOG.md` records it.

Nothing is **BREAKING** for objects: files are unchanged, and every quote that matched before still matches. Some `quote_drift` warnings in existing workspaces will stop being reported, and any whitespace-only edit inside a quoted region now reports `context_drift` rather than `quote_drift`.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `verifiable-provenance`: the "Drift is two signals" requirement replaces "compare internal whitespace verbatim" with whitespace folding, names the absent-quote-unchanged-document state, and gains scenarios for a wrapped quote, a quote across a blank line, and an indented code quote.
- `claims`: `claim assert` gains a requirement to warn when a quote is absent from a document that resolves to a workspace file; the MCP `claim_assert` result carries the same warning through the existing parity rule.

## Impact

- `internal/dkf/document.go` (`QuoteMatches`), `internal/query/drift.go` (message and comment), `internal/cli/helpers.go` (assert-time check), `internal/mcp/server.go` or `tools.go` (the same check on the tool path), their tests, `docs/provenance.md`, `CHANGELOG.md`.
- Outside this repo: the upstream DKF `source-verification` spec does not define what "appears" means, so two conforming implementations can already disagree on drift. The rule adopted here should be proposed upstream, as hash normalisation was. That is a follow-up issue against `nodelogicau/particulars`, not part of this change.
