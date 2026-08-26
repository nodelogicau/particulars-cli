# Feedback on the DKF v0.1 draft from the `particulars` reference implementation

**DKF v0.1 was declared on 2026-08-26 at
[nodelogicau/particulars@v0.1](https://github.com/nodelogicau/particulars/tree/v0.1),
with every item below resolved.** This file is the record of how the draft got
there: the [DKF README](https://github.com/nodelogicau/particulars) asked for
feedback on the object model, field names, and missing cases before v0.1 was
declared, and these are the points where implementing the format forced a
decision the draft did not make. Each was raised as an issue on 2026-08-21 and **all ten were resolved the
same day** in [nodelogicau/particulars@27743db](https://github.com/nodelogicau/particulars/commit/27743db);
the resolution is recorded under each item. particulars-cli v0.2.0 implements the
resolutions. "We" below means this implementation as it was when the item was raised.

| # | Outcome |
|---|---|
| 1 | Adopted: `<prefix>_<uuidv7>`, plus `mrg` prefix; legacy ids readable |
| 2 | Adopted: append-only `retracted` block; merges retractable; signatures exclude it |
| 3 | Adopted: optional `superseded-by`, informational only |
| 4 | Adopted: unique-not-resolvable; `<base-uri><slug>` / `urn:dkf:` minting; namespace claimed |
| 5 | Adopted: `dkf.yaml` marker; **`base-uri` must end in `/`** |
| 6 | Adopted: derived index with additive fields; rebuild + drift check required |
| 7 | Defined: `merges/mrg_….yaml`, two URIs, equivalence classes, retractable |
| 8 | Adopted: structural sets; **`current` by `timestamp` then id; `stale` is transitive** |
| 9 | **Decided differently:** `source` needs author *or* harness; agent-only sources valid |
| 10 | **Decided differently:** syntheses carry `source` (harness required), not `produced-by`; `context.scope` required on disk; cross-particular inputs allowed; `None identified` is the conventional empty `unresolved` |

Bold entries are where v0.1.x diverged and v0.2.0 changed.

Items 11–13 were raised later, from implementing merge records, workspace
discovery, and the MCP server; they are open upstream as
[#11](https://github.com/nodelogicau/particulars/issues/11),
[#12](https://github.com/nodelogicau/particulars/issues/12),
[#13](https://github.com/nodelogicau/particulars/issues/13), and
[#14](https://github.com/nodelogicau/particulars/issues/14). Item 15 came out of
running the Graph export against real knowledge and is open as
[#15](https://github.com/nodelogicau/particulars/issues/15).

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

## 11. Merge record field order ([#11](https://github.com/nodelogicau/particulars/issues/11))

The `merge-records` spec prose lists fields as "`id`, `type: merge`, `uris`,
`source`, `timestamp`, and optional `reason`" while the README example writes
`id, type, uris, reason, source, timestamp`. The reference implementation follows the example (the
concrete artifact). The prose should be brought into line, or the example
changed, before v0.1 is declared. *(Resolved: the prose was aligned to the
example in the #11 round.)*

## 12. Workspace pointer file (`.dkf`) for discovery from above ([#12](https://github.com/nodelogicau/particulars/issues/12))

`workspace-config` mandates discovery by walking up for `dkf.yaml`. That fails
when a tool runs from *above* the workspace — the common "knowledge/ inside a
repository" layout, and the default cwd of a remote agent session. v0.3.0 adds an
implementation extension: a `.dkf` file whose first non-comment line is the path
(relative to the file, or absolute) of the workspace root. At each ancestor,
`dkf.yaml` wins; otherwise a `.dkf` redirects; pointers do not chain.

It is deliberately *not* a `dkf.yaml` with a `root:` key, because a `dkf.yaml`
at the repository root would itself be a workspace marker to every conformant
reader. Conformant readers ignore `.dkf` and still work from inside the
workspace. Proposal: bless the pointer as an optional convention in
`workspace-config`, or state that tools MAY offer equivalent redirection.

## 13. `synthesis_create` has no subject parameter ([#13](https://github.com/nodelogicau/particulars/issues/13))

The tool table gives `synthesis_create(content, inputs[], unresolved, source)`,
but a `DSYNTHESIS` carries `subject` like any claim, and the tool has no way to
receive it. The reference MCP server adds `particular_id` (accepting id, URI,
label, or alias, as the other tools do). Proposal: add `particular_id` to the
signature; implementations should not infer the subject from the inputs, since
cross-particular inputs are explicitly allowed.

## 14. How does `knowledge_publish` promote an immutable claim? ([#14](https://github.com/nodelogicau/particulars/issues/14))

The federation tool list includes:

> `knowledge_publish(claim_ids[], scope)` — Promote claims from personal or
> organisation scope to public. Explicit and deliberate — not a default.

But claims are immutable, and `context.scope` lives on the claim file, so there
is no defined way to carry out the promotion. The candidates each have costs:

- **Rewrite `context.scope` in place** — contradicts immutability and the rule
  that the only permitted modification to an existing object file is the
  `retracted` block.
- **Assert a new claim at the wider scope** — changes the id, so anything citing
  the original (syntheses, `superseded-by`) still points at the narrower one.
- **A publish record**, like the merge record — a fourth record type, needing
  its own file location, fields, and precedence rule for readers computing a
  claim's effective scope.

This is not hypothetical: a workspace whose `defaults.scope` is `personal`
accumulates claims that can never be shared, and the reference implementation's
Graph export correctly emits nothing for it. Proposal: define promotion as a
record (`pub_`?) whose presence widens the effective scope of the named claims,
with the same "readers must not consult `dkf.yaml`" property that
`claim-context` already requires — or state plainly that scope is fixed at
assertion time and `knowledge_publish` selects which already-public claims to
serve in a feed.

## 15. A synthesis does not inherit its inputs' scope ([#15](https://github.com/nodelogicau/particulars/issues/15))

`context.scope` is declared per assertion, and the spec says nothing about the
relationship between a synthesis's scope and the scopes of its `inputs`. Nothing
stops an `organisation` or `public` synthesis from citing `personal` claims.

Observed on 2026-08-22 in this implementation's own knowledge workspace. An
`organisation` synthesis reconciling three `personal` claims exports to Microsoft
Graph as expected — and its item reads `claimCount: 0`, because the export
correctly withholds every one of its inputs. The belief is published; the
evidence is not. The synthesis `content`, however, quotes and argues from those
claims in detail, so the substance crosses the boundary that the scope check on
each individual claim was there to hold.

Scope therefore protects assertions **individually, but not by derivation**. Any
implementation that both honours per-claim scope and lets syntheses cite freely
has this hole; ours did, without anyone writing a line of code for it.

Three ways the spec could close it:

- **Require it**: a synthesis's scope MUST be no wider than its narrowest
  non-retracted input. Sound, but it forbids legitimate work — reconciling
  private notes into a shareable conclusion is a normal reason to synthesise, and
  the author may well have written the conclusion so it discloses nothing.
- **Warn and leave it to review**: define the condition, name it, and let
  implementations surface it. This is what we did — `validate` emits a
  `scope_wider_than_inputs` warning naming the narrower inputs, so the human
  approving the pull request reads the synthesis before it ships. It puts the
  judgement where the judgement is: no tool can tell whether prose discloses its
  sources.
- **Say nothing, deliberately**: state that scope is per assertion and derived
  disclosure is the author's responsibility. Defensible, but implementers will
  each discover this the way we did.

We suggest the second, specified rather than left to implementations, so
`scope_wider_than_inputs` means the same thing everywhere. Whichever is chosen,
the spec should say so explicitly — the silence reads as "not considered", and
scope is exactly the field where a reader assumes someone thought it through.

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
