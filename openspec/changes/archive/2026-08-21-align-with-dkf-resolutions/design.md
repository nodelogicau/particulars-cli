## Context

particulars-cli v0.1.1 implements the DKF draft as it stood on 2026-08-20. On 2026-08-21 the spec repository ([27743db](https://github.com/nodelogicau/particulars/commit/27743db)) resolved the ten issues this implementation raised and published ten baseline specs. Nine decisions match the CLI; three do not, and one record type is new. The knowledge workspace's current synthesis about particulars-cli enumerates the gap; this design closes it.

Constraints carried over from the bootstrap design: format knowledge stays in `internal/dkf`; serialisation stays byte-deterministic; existing files are never rewritten (the only mutation is the `retracted` append); every verb stays non-interactive with `--json`; v0.1.x workspaces must remain readable.

## Goals / Non-Goals

**Goals:**

- Write only spec-valid files; read everything v0.1.x wrote, warning where it is now legacy.
- Implement merge records end-to-end: write, index, validate, retract, and honour in every query verb.
- Match the spec's conflict semantics exactly (ordering, transitive staleness, equivalence classes).
- Keep the agent contract intact: same verbs and flags keep working; additions only, except the synthesis output field rename.

**Non-Goals:**

- Federation (`feed`, `publish`, `.well-known`), signing, MCP transport.
- Rewriting legacy `produced-by` files (immutability) or a migration verb.
- URI rehoming / "immutable once published" enforcement — there is no publish yet.
- Cross-workspace merges (URIs in a merge may be foreign; we only warn).

## Decisions

### D1. Synthesis provenance: `source` replaces `produced-by`; legacy accepted on read

**Choice:** `Synthesis.ProducedBy` is removed; `Synthesis.Source dkf.Source` takes its place in the same serialisation position (`id, type, subject, content, inputs, unresolved, source, method, timestamp, context, confidence, retracted`). The decoder maps a file's `produced-by: {harness, model}` into `Source` when `source` is absent and records `LegacyProducedBy = true` on the struct (not serialised). `validate` emits warning `legacy_produced_by` for such files; a file carrying both `source` and `produced-by`, or neither, is an error (`conflicting_provenance` / `missing_field`). The encoder never writes `produced-by`.

**Why:** The spec's compatibility principle — a consumer that ignores synthesis-specific fields gets a complete claim — only holds if the provenance is literally named `source`. Reading legacy files is required by the spec ("MAY treat … as `source` and SHOULD warn") and by the existing knowledge workspace.

**Alternative:** write both fields for a transition period — rejected: it makes every new file non-canonical under the spec and doubles the surface for no reader benefit.

### D2. Source minimum: author-or-harness everywhere

**Choice:** `checkSource` requires `author != "" || harness != ""`. Claims: `assert` fails with exit 2 only if neither is resolvable from flags/env/defaults. Retractions: same. Syntheses: `harness` is additionally required (spec). Merges: author-or-harness. Error text names both env vars.

**Why:** Spec. The previous `author` requirement was recorded in `SPEC-FEEDBACK` item 9 as a placeholder, not a policy.

### D3. Merge records are records, not knowledge objects

**Choice:** New `dkf.Merge{ID, URIs [2]string, Reason, Source, Timestamp, Retracted}` with `type: merge`, stored under `merges/`, prefix `mrg`. Field order follows the spec README's example: `id, type, uris, reason, source, timestamp, retracted` (the prose lists `reason` last; the example is the concrete artifact — noted as spec feedback). Merges implement `Object` and a new `Retractable` interface (`GetRetracted/SetRetracted`) shared with `Assertion`, but not `Assertion` — they have no subject, content, or context.

**Verb:** `particulars particular merge <a> <b> [--reason] [--author] [--harness] [--model]`. `a` and `b` resolve like any subject (id/uri/label/alias) *or* may be bare URIs that match no local particular (a merge routinely spans sources where only one side is local). The record stores the two URIs, sorted lexically so the same pair always serialises identically. Merging a URI with itself, or a pair already joined by a non-retracted merge, is a usage error (exit 2).

**Why under `particular`:** it is an operation on particulars, and it mirrors the spec's `particular_merge` tool name.

### D4. Equivalence classes are computed once per graph load

**Choice:** `store.Graph` gains `ClassOf(particularID) []string` backed by a union-find built from non-retracted merges at load. Edges join URIs; a URI that has no local particular still participates (so A↔foreign↔B transitively joins A and B). Query verbs take the class: `recall` and `conflicts` union `BySubject` over every member; `lineage` is unaffected in structure but its nodes already carry `subject`, so nothing more is needed for it to "operate over the class" (the spec sentence is about which objects are reachable, which lineage already is by id). `recall` entries keep their own `subject`; the JSON result gains `class: [par_…]` when it has more than one member, so the reader sees why foreign-subject entries appear.

**Why:** Union-find is O(α) and keeps class logic out of every verb; retracted merges simply contribute no edge, which is exactly the spec's rule.

### D5. Conflict semantics per spec

**Choice:** `CurrentSynthesis(class)` picks the non-retracted synthesis with the greatest `(timestamp, id)`. `stale` uses a memoised transitive walk over inputs looking for any retracted ancestor (cycle-guarded). `unsynthesised` and the reporting rule are unchanged except for operating over the class; reports are keyed by the *queried* particular (or, in the all-particulars sweep, by the class's lowest id, listing `members`). `recall` marks `unsynthesised: true` on entries not in the transitive inputs of `current` and not `current` itself.

**Why:** Spec text. Ordering by `timestamp` matters for backdated syntheses; transitive staleness matters because a retraction two levels down still undermines the conclusion.

**Trade-off:** transitive staleness over-reports for deep chains where a later synthesis already replaced the retracted leaf. Accepted — the spec chose it, and `current` still tells the reader which synthesis stands.

### D6. One `retract` verb, `claim retract` kept as alias

**Choice:** New top-level `particulars retract <id> --reason … [--superseded-by] [provenance]` handling `clm_`, `syn_`, and `mrg_`. `claim retract` is registered as an alias that delegates to the same implementation, so existing skills and scripts keep working. `superseded-by` is rejected for merge targets (a merge is undone, not superseded).

**Why:** "claim retract mrg_…" misdescribes the action; the spec treats retraction as a record that applies to three kinds of file.

### D7. Identifier grammar gains `mrg`; legacy ids warn

**Choice:** `Prefix(TypeMerge) = "mrg"`; lenient regex `^(par|clm|syn|mrg)_[A-Za-z0-9-]+$`; canonical regex likewise. `validate` adds warning `legacy_id` for any id that parses leniently but is not canonical UUIDv7 — spec says validators MAY warn; we do, because it tells a reviewer which files came from another implementation.

### D8. `base-uri` trailing slash

**Choice:** `init --base-uri X` stores `X` if it ends in `/`, else `X + "/"` and prints "normalised to …". `Config.Validate` rejects a `base-uri` without a trailing `/` (exit 4 from `validate`; exit 1 from any verb that opens such a workspace, with a message telling the user to fix `dkf.yaml`). `MintURI` becomes plain concatenation.

**Why:** Spec makes the trailing slash a validation rule; minting must therefore not paper over its absence. Normalising at `init` is the only writer-side place the value enters.

### D9. Lineage shows `superseded_by`

**Choice:** `query.Node` gains `SupersededBy string` populated from a retracted node's block; text mode renders `[retracted → clm_Y]`. It is never expanded as a child. `validate` continues to reject dangling targets; `conflicts` continues to treat the target as ordinary unsynthesised.

### D10. JSON contract change is versioned by the binary

**Choice:** Synthesis JSON output uses `source`; no `produced-by` key is emitted even for legacy files (the decoded struct has already mapped it). The `version` verb and release notes mark this as the reason for **v0.2.0**. No `--legacy-json` flag.

**Why:** Two names for one field is the kind of ambiguity the spec just removed; agents read `--help` and the skill, both of which change in the same release.

## Risks / Trade-offs

- [Knowledge workspaces with legacy syntheses show warnings forever] → They are warnings, not errors; a new synthesis supersedes the old one naturally. Documented in the review-workflow doc.
- [Union-find over URIs with foreign members could join classes surprisingly via an unknown URI] → That is the spec's intent (a merge spans sources); `validate` warns `unknown_merge_uri` so reviewers see it, and `recall --json` lists the class.
- [Timestamp-ordered `current` lets a backdated synthesis *not* become current even though it is newest on disk] → Correct per spec: backdating says "this was concluded then". Documented in `synthesis create --help`.
- [The `claim retract` alias ages badly] → Keep it for v0.2; deprecate in the skill text now; remove no earlier than v0.3.
- [Sorting the two merge URIs changes the user's argument order] → Only in the file; the text output echoes the order given. Determinism wins.
- [Transitive staleness cost on large graphs] → Memoised per load; O(V+E). Fine for file-scale workspaces.

## Migration Plan

No data migration: old files stay as they are and remain valid-with-warnings. Release v0.2.0; bump `PARTICULARS_VERSION` in the sample CI; add a CHANGELOG entry naming the JSON field rename. Rollback is reinstalling v0.1.1 — it reads everything v0.2.0 writes except merge records (ignored) and syntheses (it would report `missing_field: produced-by.harness`), so rollback is only safe before any v0.2.0 synthesis is written.

## Open Questions

- Spec README example orders merge fields `id, type, uris, reason, source, timestamp` while the requirement prose says `… source, timestamp, and optional reason`. We follow the example; raise upstream.
- Should `conflicts` in all-particulars mode report a class once (by lowest member id) or once per member? This design says once, with `members` listed; confirm when reviewing output on the knowledge workspace.
- `lineage_trace` "operating over the class" has no observable effect for an id-rooted tree; if the spec intends a particular-rooted lineage view, that is a new verb, not this change.
