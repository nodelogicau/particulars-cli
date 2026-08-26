## Why

Every DKF claim has been an assertion of fact because the format never considered an alternative. "The billing service listens on 8443", read from a config file, and "the microservices split was a mistake", held as a view, are the same object with different prose — and nothing downstream can tell them apart: not `conflict_detect`, not the Graph export composing a brief for Copilot, not an agent recalling forty claims to answer a question. In a format written mostly by agents, the highest-risk artifact is a fluent, well-formed judgement with nothing behind it, presented — because the format had no other register — as fact.

There was a smaller hole underneath: `confidence` has existed since the first draft with **no definition anywhere**. Its only calibration lived in this project's skill, which scored it evidentially — `0.9+` seen directly, `0.6–0.8` inferred — so the field carried the observed/inferred axis smuggled into a probability, and on a claim backed by nothing it carried a third meaning, which was none.

DKF closed both in `claim-evidential` (applied at [`fdab9f9`](https://github.com/nodelogicau/particulars/commit/fdab9f9); this CLI's review shaped it via [#3](https://github.com/nodelogicau/particulars-cli/issues/3) and [#4](https://github.com/nodelogicau/particulars-cli/issues/4)). It is the last breaking change before v0.1 is declared, and nothing here implements it yet.

## What Changes

- **Claims carry a required `evidential`** — `observed` (someone or something looked), `inferred` (reasoning from other claims), or `held` (nothing external; it is a position) — serialised between `timestamp` and `confidence`. **There is no default**: if absence meant `observed`, the laziest path would produce the most authoritative-looking output. `claim assert` gains a required `--evidential`; the MCP `claim_assert` gains the required parameter. **BREAKING** for every existing assert invocation, which is the point.
- **Readers stay lenient.** A claim without the field is valid, readable, and citable; its warrant is reported as **`undeclared`** — not a fourth value, not something a writer may emit, not a synonym for `observed`. It means the warrant cannot now be established: claims are immutable, so the distinction ages out rather than being migrated, and backfilling would mean inventing warrants for claims nobody can still interrogate.
- **`confidence` gets its definition**: the inverse probability that the claim is mistaken — applicable to `observed` and `inferred`, undefined for `held`. A position is not mistaken in the way a probability describes; it is not on the scale. **Writers refuse `--confidence` with `held` at write time** — the one mechanically enforceable rule in this area fires where the mistake is made, not a review cycle later — and `validate` reports `confidence_on_held` as an **error** for files that carry it anyway. Confidence on an `undeclared` claim is reported as unverified, never rejected: every existing workspace has some.
- **A synthesis declares no evidential** — it is backed by argument from its inputs, which is what a synthesis is — mirroring the existing `source.harness` asymmetry. What varies is **`method`, now a closed vocabulary**: `reconciliation` (the inputs disagreed about a fact and this settles it), `qualification` (each true in different contexts), `positions` (no evidence settles this). `synthesis create --method` validates the value; existing files with other strings read leniently and are warned.
- **A `held` claim is not exempt from reconciliation.** Two conflicting positions are still unsynthesised and still want a synthesis — a `positions` one. No conflict-semantics change.
- **The consumers act on it** — the reason the field exists. `recall` entries carry `evidential`; the Graph export's brief marks a `held` claim as a position and an `undeclared` one as undeclared, so Copilot inherits the register along with the text.
- **The skill's calibration rule is rewritten**, removing the conflation it introduced: declare the evidential; confidence only where it is defined; a position gets no number, and conviction belongs in `content`.
- `undeclared` is reported **in aggregate** (informational, one line with a count): 88 of the reference workspace's 88 claims become permanently undeclared, and per-object that report would be the entire output — the case that produced the corpus-fact rendering in `f6fe937`.

## Capabilities

### New Capabilities
- `claim-evidential`: the evidential axis and its three values, required-on-write / lenient-on-read and what `undeclared` means, the definition of `confidence` and its exclusion from `held`, the synthesis asymmetry, and the closed `method` vocabulary.

### Modified Capabilities
- `object-format`: claim field order gains `evidential`; field constraints gain the enum and the held/confidence exclusion.
- `claims`: `assert` requires `--evidential` and refuses `--confidence` with `held`.
- `syntheses`: `method` becomes a closed vocabulary.
- `validation`: `undeclared` (info, aggregated), `confidence_on_held` (error), `confidence_on_undeclared` (info, aggregated), `unknown_method` (warning).
- `mcp-server`: `claim_assert` takes required `evidential`, per the spec's tool table.
- `knowledge-query`: recall entries carry `evidential`.
- `graph-export`: the brief marks positions and undeclared warrants.

## Impact

- **Modified**: `internal/dkf` (field, enum, codec, validation), `internal/cli/cmd_claim.go` and `cmd_synthesis.go`, `internal/mcp/tools.go`, `internal/query` (validate, recall entry), `internal/graph` (brief), `skills/particulars/SKILL.md`, README, docs. Every CLI test that asserts a claim gains the flag.
- **Compatibility**: read-side, total — no existing workspace becomes invalid, no file is rewritten, and the byte-stability guard holds. The break falls entirely on writers, which was the recorded reason to fold this in before v0.1 rather than declare and break a week later.
- **Version**: v0.9.0.
- **Not in scope**: the index unknown-entry-type preservation that also just landed upstream (`db748da`, from [particulars#16](https://github.com/nodelogicau/particulars/issues/16)) — same release, its own change; and any strength-of-conviction field, which the spec deliberately declines.
