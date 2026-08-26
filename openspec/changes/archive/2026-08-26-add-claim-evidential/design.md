## Context

This is the first change where the write side and the read side of the same rule genuinely diverge, and most of the design is keeping them apart cleanly. A writer must refuse a claim with no `evidential`; a reader must accept one, because 88 of the reference workspace's 88 claims were written before the field existed and can never be rewritten. The same split applies to `held` + `confidence`: refused at write, reported as an error by `validate` when found in a file, never refused on read.

The precedent for everything here already exists in the codebase: `source.harness` is required on syntheses but not claims (the asymmetry), `legacy_produced_by` is the lenient-read-with-report pattern, and `f6fe937`'s corpus-fact aggregation is exactly the rendering `undeclared` needs.

## Goals / Non-Goals

**Goals:**

- Writer-required, reader-lenient, with the enforcement living at the two write surfaces and nowhere else.
- `undeclared` reported so that a workspace where it is universal — every existing workspace — still has readable validate output.
- The consumers act on the field: this implementation was named as the first in a position to compute anything from warrant, and the Graph brief is where it pays.

**Non-Goals:**

- Backfilling, migration, or any writable `undeclared`.
- A strength-of-conviction field.
- Deriving `held` conflicts differently — the label changes what the synthesis work is, not whether there is work.
- The index unknown-entry preservation from `db748da` — its own change.

## Decisions

### Enforcement lives at the write surfaces, not in `dkf.ValidateClaim`

`dkf.ValidateClaim` runs on every claim `validate` loads, so requiring `evidential` there would invalidate every pre-existing file — the opposite of lenient. It validates only what is universally true: *if present*, the value is one of the three. The requirement itself is enforced in `cmd_claim.go` and the MCP `claimAssert` handler, the only two places a new claim is born. `held`+`confidence` is the one exception: that combination is wrong *wherever it appears*, so `ValidateClaim` rejects it — which makes it a validate-time **error** for files, and gives the write surfaces the refusal for free through the existing `ValidateObject` call in `Create`.

### `undeclared` is informational, not a warning

The legacy markers ship as warnings, and consistency argued for the same here. Two things argue louder. An undeclared claim is not a degraded artifact of format evolution the way a `produced-by` block is — it is a fully valid claim whose warrant cannot be established, which is the `unverified_document` family: a statement that nothing is known, not that something is wrong. And the counting consequence: 88 warnings would turn the reference workspace's summary line into `97 warnings`, which reads as a workspace in trouble. It is not in trouble; it is old. Info severity, aggregated by the existing severity collapse: one line, `88 objects  undeclared`.

`confidence_on_undeclared` (the spec's "reported as unverified rather than rejected") is info for the same reason, under its own code so the two facts stay distinguishable in `--json`.

### `method` is enforced at write and warned at read

`synthesis create --method` accepts only the three values (empty still defaults to `reconciliation`). Files carrying anything else — this CLI never wrote one, but the flag was freeform, so they may exist — read leniently and are warned `unknown_method`. Not a corpus fact: it is rare, per-object, and unlike the legacy markers it names an authorial claim (`method: consensus`?) a reviewer might genuinely want to look at. The spec's rule that an evaluative conclusion recorded as `reconciliation` is invalid is authorial judgement no machine can check, and nothing here pretends to.

### The brief marks the register inline, on the SUPPORTING lines

A `held` claim renders with a ` [position]` marker and an `undeclared` one with ` [undeclared]`, next to the existing `[unsynthesised]` marker; `observed`/`inferred` render unmarked, because marking the normal case is noise. The failure this prevents is concrete: Copilot citing "the split was a mistake" from a brief with nothing to distinguish it from "listens on 8443". Confidence never renders for `held` because it cannot exist there.

### Recall entries carry the field verbatim, empty when undeclared

`Entry.Evidential` with `omitempty`: absent from JSON exactly when the file has no field, so a consumer sees the same three-values-or-nothing the file says, and `undeclared` remains a validator's report rather than a value that travels.

### The skill teaches the axis, not a heuristic

The old calibration rule *was* the conflation — `0.9+` seen directly, `0.6–0.8` inferred is the evidential axis expressed as numbers. The rewrite: declare `--evidential` on every claim; `confidence` is the inverse probability the claim is mistaken and exists only for `observed` and `inferred`; a `held` claim gets no number, and how strongly a position is held belongs in `content`, where reasoning lives. Two conflicting positions still want a synthesis — `--method positions` — which the synthesis guidance gains.

## Risks / Trade-offs

- **Every scripted `claim assert` breaks at once.** → Deliberate and spec-required; the error message names the three values and what each means, so the fix is self-describing. In-repo, the test churn is mechanical and done in this change.
- **MCP clients with cached tool schemas** may call `claim_assert` without the parameter and get a usage error rather than a schema rejection. → The error is the same text as the CLI's; a restart refreshes the schema.
- **Agents may reflexively pass `--evidential observed`** to satisfy the flag, recreating the silent default socially. → The skill says what each value means and that `observed` claims to have looked; nothing mechanical can prevent it, and the spec accepts this openly — the field makes the laziest path *visible*, not impossible.
- **Info severity for `undeclared` could under-alert** a user who wants to know their corpus predates the field. → The aggregate line is printed on every run; it is one line, not invisible.

## Open Questions

- Whether `conflicts` should eventually surface "two `held` claims, unreconciled" differently from factual conflicts — the spec's `positions` method suggests the shape. Deferred: the structural sets are identical, and nothing yet consumes the distinction.
