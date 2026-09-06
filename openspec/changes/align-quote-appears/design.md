## Context

`fold-quote-whitespace` (ac48d45) made `dkf.QuoteMatches` fold whitespace with `strings.Fields`, which splits on `unicode.IsSpace` — the Unicode `White_Space` property. Upstream accepted that rule as change `quote-appears` and, in doing so, named every cell of the drift table and settled what an empty quote means. The CLI's words lag the spec in three places and its state machine in two.

Current behaviour at the edges:

- `Document.Validate` reports a whitespace-only quote as `invalid_document`, an error. `QuoteMatches` would return false for it, but validation rejects the object first.
- `VerifyDocument` with a quote and no hash reports nothing when the quote is found. The CLI's `validation` spec has always listed "a document with no hash" under `unverified_document`; the code never emitted it in this case.
- `VerifyDocument` with a hash and no quote, when the hash differs, reports `context_drift` with a message about "the quoted text", which does not exist.
- The MCP `documentFrom` accepts any string as `quote`, including blank. The CLI refuses blank `--quote` and a whitespace-only `--quote-file`; an empty `--quote-file` yields no quote silently, which the spec permits.

## Goals / Non-Goals

**Goals:**
- Every sentence the CLI prints or specifies about quote matching agrees with `quote-appears`.
- Every cell of the drift table, including the no-hash and no-quote cells, is reached by rule and pinned by a test.
- An existing workspace with a whitespace-only quote validates, with a warning, without anyone editing an immutable file.

**Non-Goals:**
- Changing the matcher. It is already conformant.
- Format-aware folding (stripping `>` or `//` from continuation lines). The spec states the limit; this change documents it.
- Reporting `empty_quote` per object. The file cannot be rewritten, so a per-object listing recurs forever and buries actionable findings; that is the condition-reporting rule.

## Decisions

### `Document.HasQuote()` is the single notion of "carries a quote"

A helper on `dkf.Document` returns true when the quote folds to something non-empty. `VerifyDocument`, the quoted-source disclosure in `validate`, and `QuoteDisclosuresForPromotion` all gate on it instead of `Quote != ""`. `Validate` drops the `invalid_document` problem for an empty quote entirely, so the object loads and the reader path decides what to say. One predicate, three call sites, and no chance of the drift check and the disclosure note disagreeing about whether a quote exists.

Alternative: normalise the empty quote away at load time. Rejected because the file's bytes are what they are, `validate` must still be able to say the quote is empty, and canonical-serialisation checks compare against the raw file.

### `empty_quote` is a corpus fact, warning severity

The spec says SHOULD warn. It is reported like `legacy_document_uri`: warning severity, aggregated to one line with a count in text output, every finding present in `--json`. It joins `IsCorpusFact`. Warning rather than info because a writer did something the spec forbids; aggregated because no object-level action exists.

### Quote only, present: emit `unverified_document`, not a new code

The spec says "no drift with the hash unverified". The CLI already has the code for a document it could not fully check, and its own spec already names "a document with no hash" as a case of it. The message says the quote appears but, with no hash, the text around it is not checked. Informational, aggregated. This is the one change that adds findings to existing workspaces; each quote-only local claim adds one to a count that is already printed.

### Blank quote is refused at the MCP boundary in `documentFrom`

Same predicate as `HasQuote`: `d.Quote != "" && !d.HasQuote()` is a usage error naming `source.document.quote`. Placed in `documentFrom` because that is where the mapping's shape is already policed (unknown keys, `ref` versus `uri`). The CLI's `resolveDocument` keeps its check, reworded to the same predicate so both boundaries agree with the reader.

### Hash-only, hash differs: keep `context_drift`, fix the message

With no quote there is nothing for quote drift to be about, and the document did change, so `context_drift` is the right code — upstream's table says context drift is "quote present, hash differs", and a mapping with no quote vacuously has nothing absent. Only the message was wrong. It now says the document changed and, carrying no quote, whether the cited text survived is unknown.

### Wording: name the property, cite the Go function once

Spec and docs say "the Unicode `White_Space` property". The matcher's comment adds that `strings.Fields` splits on `unicode.IsSpace`, which is that property, so a maintainer does not have to look it up. The docs' "agrees with the hash" sentence becomes "for everything other than whitespace, the quote agrees with the hash about what an edit is".

## Risks / Trade-offs

- [Existing workspaces gain `unverified_document` counts for quote-only local claims] → Aggregated info; the count is already on the line. The alternative, silently treating a found quote as full verification, is what the spec now rules out.
- [Dropping `invalid_document` for an empty quote lets a malformed write through on read] → That is the point: the write is refused at both boundaries, and a file that already carries one can never be fixed, only warned about.
- [`HasQuote` folds the quote on every call] → Quotes are sentences; the cost is nil. `VerifyDocument` already folds the whole document.
