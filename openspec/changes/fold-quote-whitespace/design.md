## Context

`dkf.QuoteMatches` normalises CRLF to LF, trims the whole quote, and then runs `strings.Contains`. `query.VerifyDocument` turns the result into one of four outcomes: nothing, `context_drift`, `quote_drift`, or `quote_drift` with a special message when the hash still matches. `claim assert` builds the document mapping in `resolveDocument`, which already reads the local file when `--hash-document` is given but never looks at the quote. The MCP `claim_assert` path builds the document from client-supplied fields and never reads the file.

The verbatim rule was chosen in `add-verifiable-provenance` so that indentation inside a quoted code block would count. Issue #9 shows the cost: a sentence wrapped at 80 columns cannot be quoted at all, and the workaround — reproducing the wrap in `--quote-file` — breaks the next time the document is re-wrapped, which is exactly the drift the check is meant to survive.

## Goals / Non-Goals

**Goals:**
- A verbatim quote matches across a hard wrap, a re-wrap, a CRLF checkout, and a re-indent, without changing what is stored.
- The rule is stated in the spec precisely enough that another implementation can reproduce it.
- A writer learns at assert time that a quote does not match a local document.

**Non-Goals:**
- Fuzzy or case-insensitive matching. Words must still match exactly.
- Unicode normalisation (NFC/NFD) or smart-quote folding. Those are edits by the hash rule, and the quote should agree with the hash about what an edit is.
- Fetching anything. Verification stays offline.
- Changing the upstream DKF spec from this repo. That is a separate proposal.

## Decisions

### Fold every whitespace run to one space, on both sides

Alternatives: fold only single newlines and keep a blank line as a token, so a quote cannot silently join two paragraphs; or fold only newlines and keep spaces and tabs verbatim.

The single rule is chosen because it is the simplest to state, to implement, and to reproduce elsewhere: `strings.Join(strings.Fields(s), " ")` on both sides, with `unicode.IsSpace` deciding what whitespace is. The paragraph-join case it admits is a misquote the writer authored; a substring test cannot police meaning either way, and a reviewer reading the quote in the pull request will see the join. Keeping tabs verbatim would re-open the bug for any editor that converts tabs to spaces on save.

Folding both sides never loses a match that the old rule found: any exact substring is still a substring after both sides undergo the same folding. So no existing quote regresses.

### The stored quote stays verbatim

Folding is a property of the comparison, not of the file. The quote in the claim is what the writer read, reproduced as they gave it; a reviewer comparing by eye should see the source's own shape. This also keeps the serialised bytes of every existing claim unchanged.

### The absent-quote-unchanged-document state is `quote_drift`, and the message says why

The spec's table only defines three cells. The CLI has always reported the fourth — quote absent, hash matches — as `quote_drift`, and that stays: the reader's action is the same (the cited text is not there). But "does not appear … though the document is unchanged" reads as a contradiction. The message becomes: *the quoted text has never been an exact match for `<path>`, which is unchanged since the claim was written; the quote was miscopied or taken from a different revision*. The spec names the state so a second implementation reports the same code.

### Warn at assert time; never refuse

When the document resolves to a readable file in the workspace and the quote is not found after folding, the claim is still written and the result carries a warning. Refusing would block a writer whose document is at a different revision than the one they read, and the format's rule is that provenance conditions are reported, never refused. The check runs whenever the ref resolves locally, not only under `--hash-document`: a quote against a local file is checkable either way, and the cost is one file read the hash path already pays.

The MCP tool gets the same check so the structured result stays equal to the CLI's `--json` output, which the `mcp-server` spec requires. The tool result gains a `warnings` array only when there is one, matching how `knowledge_publish` already reports.

### Whitespace-only edits now report `context_drift`

Under the old rule a re-indent of a quoted code block was `quote_drift`; now the folded quote is still present and the hash differs, so it is `context_drift`. That is the more honest signal: the words the claim rests on did not change, the document around them did. The spec scenario for the indented code quote states this.

## Risks / Trade-offs

- [A claim about whitespace itself — "the block is tab-indented" — can no longer be caught by its quote] → The hash still catches the edit as `context_drift`. Claims about formatting are rare enough that a weaker quote signal is an acceptable price for every prose quote working.
- [Folding the whole document per claim on every `validate`] → Folding is linear and the file is already read per claim. Documents cited by many claims could be folded once per run; not done now, since no workspace is close to noticing.
- [Two implementations disagreeing on what "appears" means] → Already true today because the upstream spec is silent. Proposing this rule upstream is the mitigation and is tracked as a follow-up.
- [An assert-time warning surprising scripted callers that parse `--json`] → `warnings` is already an optional array on results; callers that ignore it lose nothing.
