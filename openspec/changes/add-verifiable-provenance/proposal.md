## Why

The format's first principle is that provenance is non-negotiable, and `source.document` — the field that carries it — has been an optional, unvalidated string since the first draft. Nothing in it is checkable. A reviewer reading a pull request can see that a claim cites `docs/architecture.md`, and has no way to know whether the document still says what the claim says it said, or ever did.

DKF [`7dac8dd`](https://github.com/nodelogicau/particulars/commit/7dac8dd) closes that: `document` may be a mapping carrying a hash and a verbatim quote, drift is defined as two distinct signals, and `retracted` gains a `kind` recording *why* a claim died. This is the first round where the spec moved before the implementation — [particulars-cli#3](https://github.com/nodelogicau/particulars-cli/issues/3) is explicit that it is a proposal, and pushback is worth more than the text.

There is a second reason to want it here. `source.harness` has been recorded on every claim since v0.1 and no implementation could ever act on it, because nothing recorded whether a claim was *sound*. `kind` is that missing observation: `defect` counts against the process that produced a claim, `supersession` counts against nothing, `provenance-failure` counts against the document and makes every other claim citing it worth re-reading.

## What Changes

- **`source.document` becomes a union.** A bare URI string stays valid and is explicitly not inferior provenance; the mapping form adds `uri` (required), `hash` (optional), and `quote` (optional), in that order. The scalar form round-trips byte-identically, so no existing claim changes.
- **Relative paths are accepted in `uri`** and resolved against the workspace root. Agents write repo-relative evidence (`docs/architecture.md`), and refusing it would push them back to bare strings — losing the whole point.
- **Drift is computed offline, from local files only.** `validate` does not reach the network and will not start. A document that cannot be checked — a remote URI, a missing file, a source with no hash, a conversation — is reported `unverified_document`, never invalid. **Drift is never a validation error.**
- **Two signals, per the spec's table**: quote present and hash differs is `context_drift` (something moved around the claim); quote absent and hash differs is `quote_drift` (the cited text is gone).
- **`retracted.kind`**: `defect`, `supersession`, or `provenance-failure`, optional, serialised between `source` and `superseded-by`. Declared, never derived from `superseded-by` — the most common defect is a typo fix, which carries a replacement, and a claim retracted because its subject was decommissioned is an honest supersession with nothing to point at.
- **A quote discloses its source completely**, unlike a synthesis, which summarises. Promoting a claim that carries a quote is the moment to say so, reusing the `scope_wider_than_inputs` reporting path.
- **The skill stops teaching a line offset.** It currently shows `--document src/billing/cron.go:14`; the spec explicitly rejects offsets because they break when anyone inserts text above them. It will teach `--quote`.

**Not implemented, deliberately:** the spec's SHOULD cross-check of `kind` against drift. Fifteen claims in the reference workspace cite commit-pinned blob URLs whose hash cannot change by construction, so a `supersession` against an unchanged hash is the *normal* case for well-sourced claims — the check would flag the strongest sourcing discipline available. Raised on #3; awaiting the maintainer.

## Capabilities

### New Capabilities
- `verifiable-provenance`: the document union and its field order, relative-path resolution, hash normalisation, the two drift signals and their offline-only computation, unverified sources, retraction `kind`, and quote disclosure.

### Modified Capabilities
- `object-format`: canonical field order gains `document`'s mapping form and `retracted.kind`.
- `claims`: `assert` gains `--document-hash`, `--hash-document`, `--quote`, `--quote-file`; `retract` gains `--kind`.
- `validation`: `context_drift`, `quote_drift`, `unverified_document`, and the quote-disclosure warning.
- `mcp-server`: `claim_assert` accepts the document mapping; `claim_retract` accepts `kind`.
- `scope-promotion`: promoting a quoted claim reports the disclosure.

## Impact

- **New**: `internal/dkf/document.go` (the union type, marshalling, hash normalisation), `internal/query/drift.go`.
- **Modified**: `internal/dkf/types.go` and `codec.go` (`Source.Document`, `Retracted.Kind`), `internal/cli/cmd_claim.go`, `cmd_retract.go`, `cmd_publish.go`, `internal/mcp/tools.go`, `internal/query/validate.go`, `skills/particulars/SKILL.md`, `docs/review-workflow.md`, README.
- **Compatibility**: every existing claim carrying a string `document` stays valid and serialises to the same bytes. A workspace with no mapping documents behaves exactly as today, and `validate` gains no findings for it.
- **Not in scope**: signing; network fetching; and an `evidential` field — raised on #3 as unbackfillable, since adding a `retracted` block is the only permitted modification to an existing file, so a *required* new field would invalidate every claim ever written with no remedy but re-asserting them all.
