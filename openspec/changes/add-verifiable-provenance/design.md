## Context

Three constraints shape this more than the feature does.

**The codec is byte-stable and every existing claim carries a string `document`.** A union type that serialises the scalar form differently — or reorders anything — rewrites files that must not change. The `non_canonical` warning exists to catch exactly that, so the round-trip test is the design's first obligation, not its last.

**`validate` has never made a network request**, and neither has any other verb; the Graph export makes a point of it. Drift verification is the first feature that could plausibly justify one, and accepting that would make `validate` slow, non-deterministic, and dependent on credentials for private sources. It stays offline.

**The real corpus is mostly remote and mostly immutable.** Of the documents in the reference workspace, fifteen are commit-pinned GitHub blob URLs, three are unfetchable (`chat session 2026-08-22`), and three are relative paths. So drift checking will be a no-op for most claims here, and the *quote* — not the hash — is what makes a claim auditable in a pull request. That ordering drives the task list.

## Goals / Non-Goals

**Goals:**

- The scalar document form is untouched, byte for byte, forever.
- Verification is offline, deterministic, and cheap enough to run in `validate` unconditionally.
- A hash means the same thing on every platform, so a check never fires because of a checkout.
- A reviewer can audit a claim against its quote with no tooling.

**Non-Goals:**

- Fetching remote documents, now or by default later. If it arrives it is an explicit opt-in flag, and `validate` without that flag stays offline.
- Signing. The spec still lists the signed payload's definition as open.
- An `evidential` field.
- The `kind`-versus-drift cross-check as specified — see below.

## Decisions

### `Document` is a struct with a scalar-preserving codec, not an `any`

```go
type Document struct { URI, Hash, Quote string }
```

`UnmarshalYAML` accepts a scalar (sets `URI`) or a mapping. `MarshalYAML` emits a scalar when `Hash` and `Quote` are both empty, and a mapping otherwise. JSON mirrors it: a string when only `uri` is set, so existing MCP clients and `--json` consumers see no change.

*Alternative considered:* keeping `Document string` and adding sibling fields `document-hash`/`document-quote`. Rejected — it diverges from the spec's shape, and a reader of the YAML would have to know that three keys are one fact.

### The hash is taken over LF-normalised bytes

CRLF sequences become LF before hashing; **nothing else is normalised**. Not trailing whitespace, not final newlines, not Unicode.

Raw bytes would mean the same commit hashes differently on a Windows checkout than on a Linux one — so a drift check would fire on every claim for Windows users. This project already carries `.gitattributes` with `eol=lf` because Windows CI broke on CRLF once; that is the same failure arriving through a different door, and it is precisely the cries-wolf outcome the quote design was chosen to avoid.

Trailing whitespace is deliberately *not* normalised even though it changes a hash without changing meaning. Every additional normalisation is a class of real edit the check can no longer see. Normalise what is an artefact of transport; leave what is an edit.

The consequence to accept: `shasum` on a CRLF file will not match the recorded hash. That is documented, and it only bites files git is usually normalising anyway.

### Quote matching normalises line endings and trims the whole quote

A YAML block scalar acquires a trailing newline, so an exact-substring test would fail on a quote that is textually present. Line endings normalise, the whole quote's leading and trailing whitespace is trimmed, and **internal whitespace is compared verbatim** — indentation inside a quoted code block is part of what was quoted.

### Verification is offline, and "not checked" is a first-class outcome

A document is checkable only when its `uri` resolves to a file inside the workspace — a relative path, or a `file:` URI. Everything else reports `unverified_document`. That finding is informational: it says the claim's provenance was not machine-checked, not that anything is wrong. A workspace of remote citations will carry many, which is why it must never be an error and why it is reported once per claim rather than per finding type.

### Drift findings are warnings and never fail the build

`context_drift` and `quote_drift` are warnings, matching the spec's "a condition for a reader to resolve, never a validation failure". A drifted claim stays valid, readable, and citable.

### The `kind`-versus-drift cross-check is not implemented

The spec SHOULDs it: a `supersession` against an unchanged hash is suspect. The three kinds are joints in `claim → source → world`; drift is a signal about the *source* joint and supersession is the *world* joint, so the cross-check only holds when the document is a living description of current state — which the format cannot distinguish from a dated one.

The counter-example is the ordinary case: fifteen claims in the reference workspace cite commit-pinned URLs whose content is immutable by construction. Every one would be flagged the day it is superseded, and pinning to a commit is the best sourcing discipline available.

If a check is wanted, the sound direction is the other one: a `defect` against a document that *has* drifted is **unverifiable**, because the text the author misread is no longer the text a reviewer can read. That is a statement about what can be checked rather than a guess about intent. Neither ships until the maintainer rules on #3.

### Quote disclosure is reported at promotion

A quote reproduces its source completely, where a synthesis summarises. There is no scope on a document to compare against, so the honest trigger is the moment exposure widens: promoting a claim that carries a quote reports that verbatim source text is being published. `validate` mentions it too, but promotion is where a person is deciding.

## Risks / Trade-offs

- **A union type breaks the byte-stable round-trip.** → The first test written, over every existing claim shape, before any feature work. `non_canonical` would catch a regression in the real workspace too.
- **Hash normalisation is a spec-open question and we are answering it locally.** → Recorded on #3 with the reasoning, so if the spec settles differently the change is one function. The alternative — waiting — leaves the field unusable.
- **`unverified_document` could bury the real findings.** Most claims here cite remote URIs, so the count will be large. → Reported at info severity and summarised rather than listed per claim; the CI comment already groups by code.
- **Agents may quote too much**, turning claim files into copies of their sources and worsening the disclosure problem. → The skill says what a quote is for: the sentence that supports the claim, not the section containing it.
- **Relative paths are ambiguous** if a workspace is not inside the repository the path refers to. → Resolution is relative to the workspace root, stated in the docs; a path that does not resolve is `unverified_document`, not an error.

## Open Questions

- Whether `--hash-document` should also be offered on `synthesis create`. Syntheses cite inputs rather than documents, but they do carry `source.document`. Left out until someone wants it.
- Whether drift should be surfaced in the visual export — a drifted claim could be drawn distinctly. Deferred; the model supports it without renderer changes.
