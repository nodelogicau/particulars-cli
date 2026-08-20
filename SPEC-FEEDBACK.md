# Feedback on the DKF v0.1 draft from the `particulars` reference implementation

The [DKF README](https://github.com/nodelogicau/particulars) asks for feedback on
the object model, field names, and missing cases before v0.1 is declared. These
are the points where implementing the format forced a decision the spec does not
make, or where this implementation deliberately diverges. Each is phrased as a
proposal; "we" below means this implementation.

## 1. Identifier format: `<prefix>_<uuidv7>` ([#1](https://github.com/nodelogicau/particulars/issues/1))

**Spec today:** examples like `par_01j9xk2p3q4r5s6t` (a truncated ULID); "ID
formats … subject to change."

**Proposal:** `<prefix>_` + lowercase canonical UUID version 7
([RFC 9562](https://www.rfc-editor.org/rfc/rfc9562)), e.g.
`clm_019196a5-8b4c-7def-8abc-0123456789ab`. Same time-sortable property as ULID,
but "it's a UUID" is the stronger interoperability story for a format whose
purpose is portability: every language, database, and validator already
understands it. Implementations should mint with a monotonic counter so ids
created within one millisecond still sort in creation order.

Also worth stating: the id's embedded time is the *minting* instant; the
`timestamp` field is the *assertion* time and may be earlier (recording a dated
document). Consumers must not require them to agree.

We accept any `^(par|clm|syn)_[A-Za-z0-9-]+$` on read, so workspaces written
with other schemes remain readable.

## 2. Retraction representation ([#2](https://github.com/nodelogicau/particulars/issues/2))

**Spec today:** "Retraction is recorded, not deletion"; `claim_retract(claim_id,
reason, source)` "marks a claim retracted". No field or object is shown.

**Proposal:** an append-only `retracted` block on the claim or synthesis file:

```yaml
retracted:
  timestamp: 2026-08-21T09:12:00Z
  reason: "Port is 8443, not 443 — deploy/config.yaml:12"
  source:
    author: ben
  superseded-by: clm_…        # optional
```

Rules: it is the only permitted modification to an existing object file; it is
never removed (reinstatement is a new claim or synthesis citing the retracted
one); syntheses are claims and may be retracted; the index mirrors it as
`retracted: true`.

Why on the file rather than a separate `ret_` object: the consumer most likely
to misuse a retracted claim is one that opens only the claim file, and the spec's
own compatibility principle ("a consumer that ignores the format entirely gets
readable YAML") argues the marker must be visible there. It also keeps to three
object types and gives the best git diff.

Consequence for the reserved signing field: define the signed payload as the
object *minus* `retracted` and `signature`, so a retraction does not invalidate
the original signature.

## 3. `superseded-by` on retraction ([#3](https://github.com/nodelogicau/particulars/issues/3))

A lightweight correction pointer for typo-grade errors, where a full
thesis/antithesis synthesis is ceremony. Optional; must reference an existing
claim or synthesis. The spec should either bless it or say that correction is
*only* by synthesis.

## 4. URIs for unpublished particulars ([#4](https://github.com/nodelogicau/particulars/issues/4))

**Spec today:** `uri` is "canonical, globally resolvable".

**Problem:** most things an agent learns about — a module, a decision, a
service — have no resolvable URI, and agents are bad at re-inventing the same
URI consistently across sessions.

**Proposal:** soften to "globally unique; resolvable once published", and bless
a minting convention: `<base-uri><slug>` when the workspace has a base URI,
otherwise `urn:dkf:<workspace-id>:<slug>`, with `slug` derived from the label
(lowercase, ASCII-folded, hyphenated). `particular_define` being idempotent on
URI then means the same label resolves to the same particular automatically.
Existing global URIs (Wikidata, ORCID, GitHub) remain preferred when they exist.

If `urn:dkf:` is adopted the spec should claim the namespace in a sentence; the
alternative is a `tag:` URI ([RFC 4151](https://www.rfc-editor.org/rfc/rfc4151))
convention, which needs a DNS or email authority, i.e. configuration.

A URI should be changeable only while the particular has never been published;
afterwards `particular_merge` is the only path.

## 5. Workspace configuration: `dkf.yaml` ([#5](https://github.com/nodelogicau/particulars/issues/5))

The spec defines the file layout but no configuration or marker file. Minting
URIs needs a stable workspace identity and optional base URI; agents need a
default `source.author`. Proposal:

```yaml
format: dkf/0.1
workspace:
  id: <uuidv7>
  base-uri: https://example.com/particulars/   # optional
defaults:
  scope: personal
  source:
    author: ben
    harness: claude
```

placed at the workspace root and used as the discovery marker (walk up from the
working directory).

## 6. `index.yaml` is derived ([#6](https://github.com/nodelogicau/particulars/issues/6))

**Spec today:** "a lightweight manifest … enables recall and conflict detection
without parsing every file."

**Proposal:** state explicitly that the index is a regenerable cache, never the
source of truth, and that implementations may add fields (we add `scope`,
`topics`, `timestamp`, `retracted`). Rationale: two branches adding claims both
touch `index.yaml`; if it is authoritative every merge conflicts and a human must
hand-merge YAML. If it is derived, conflicts are resolved by rebuild. It stays
committed so public consumers can fetch it over HTTP.

## 7. The undefined merge record ([#7](https://github.com/nodelogicau/particulars/issues/7))

`particular_merge` "produces a merge record without rewriting claims" — a fourth
object type the spec does not define. It needs an id prefix, a file location,
fields (the two URIs, `source`, `timestamp`), and a rule for how `recall` treats
claims on either side.

## 8. Conflict semantics ([#8](https://github.com/nodelogicau/particulars/issues/8))

`conflict_detect` "surfaces unresolved contradictions". Without an LLM in the
loop, "contradiction" is not computable; what *is* computable is **what has not
been reconciled**. Proposal: document a structural baseline every consumer can
rely on —

- **current** = the most recent non-retracted synthesis about a particular
- **unsynthesised** = non-retracted claims and syntheses not in the transitive
  inputs of `current`
- **stale** = syntheses with a retracted input

— and leave the semantic judgement to the reasoning harness. This also gives a
precise meaning to "the current belief about any particular is the most recent
synthesis" when some claims post-date it.

## 9. Which `source` fields are required ([#9](https://github.com/nodelogicau/particulars/issues/9))

The spec shows `author`, `harness`, `model`, `document` but does not say which
are mandatory. We require `author` on claims and on retractions, and
`produced-by.harness` on syntheses. Worth pinning down, since it determines what
a minimal valid claim looks like.

## 10. Smaller notes ([#10](https://github.com/nodelogicau/particulars/issues/10))

- `DSYNTHESIS` "extends `DCLAIM`" but the example carries `produced-by` and no
  `source`. Consumers that treat a synthesis as a claim therefore find no
  `source`; either the spec should say `produced-by` plays that role, or a
  synthesis should carry `source` too.
- `context.scope` defaults to `personal` per the tool table; the object examples
  should say whether `context` itself is required.
- Synthesis inputs are not required to share the synthesis's `subject`. We
  allow cross-particular inputs (a claim about library Y informing a synthesis
  about project X). The spec should say whether this is intended.
- `unresolved` is required even when nothing is unresolved; a conventional value
  ("None identified") would help tooling distinguish "considered and empty"
  from "forgot".
