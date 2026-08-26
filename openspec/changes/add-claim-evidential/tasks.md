## 1. The field

- [ ] 1.1 `dkf.Evidential` type with the three constants and `ValidEvidential`; `Claim.Evidential` with `omitempty`, codec emitting it between `timestamp` and `confidence`, absent staying absent.
- [ ] 1.2 `dkf.ValidateClaim`: *if present*, the value must be one of the three (error otherwise); **absence is not an error** — the requirement is enforced at the write surfaces, not on read, or every pre-existing file would be invalidated.
- [ ] 1.3 `ValidateClaim` rejects `held` + `confidence` — the combination is wrong wherever it appears, so this one rule lives in the shared validator and gives write surfaces the refusal through the existing `Create` path.
- [ ] 1.4 Round-trip tests: field position, absent-stays-absent, byte-stability guard still green over the real workspace (every one of its claims is pre-evidential).

## 2. Method vocabulary

- [ ] 2.1 `dkf.ValidMethod` over `reconciliation`, `qualification`, `positions`; `synthesis create --method` refuses others, empty still defaults to `reconciliation`.
- [ ] 2.2 Writers never accept an evidential for a synthesis (no flag exists — assert one stays absent in the codec for syntheses).
- [ ] 2.3 `validate` warns `unknown_method` on files carrying other strings; read stays lenient.

## 3. Validation findings

- [ ] 3.1 `undeclared` at info severity for every claim without the field, aggregated by the existing severity collapse — one line, `88 objects  undeclared`, on the reference workspace.
- [ ] 3.2 `confidence_on_held` as an **error** (exit 4), with the claim still readable by `recall`, `lineage`, and the exports — assert readability in the test, since "reject" must mean report-and-fail, never refuse-to-read.
- [ ] 3.3 `confidence_on_undeclared` at info severity, its own code so the two facts stay distinguishable in `--json`.
- [ ] 3.4 Tests: a pre-evidential workspace validates with no errors; the one error case exits 4; aggregation shows counts and changes no totals.

## 4. Surfaces

- [ ] 4.1 `claim assert --evidential`, required, no default; the exit-2 message names the three values and what each means, so the fix is self-describing. `--confidence` with `held` refused before anything is written.
- [ ] 4.2 MCP `claim_assert` gains required `evidential` (enum in the schema), refusing `held`+`confidence` with the same message; `synthesis_create` validates `method`. Tool descriptions updated; results equal the CLI's `--json`.
- [ ] 4.3 `query.Entry.Evidential` with `omitempty`: present exactly when the file declares one.
- [ ] 4.4 Graph brief: `[position]` marker on `held` lines, `[undeclared]` on pre-evidential lines, `observed`/`inferred` unmarked; confidence never renders for `held` (it cannot exist there, but the renderer should not depend on that).
- [ ] 4.5 Update every CLI test invoking `claim assert` with the new flag — mechanical, and the churn is itself the breaking-change evidence.

## 5. The skill

- [ ] 5.1 Replace the calibration rule that introduced the conflation: declare `--evidential` on every claim, with one line on what each value claims; `confidence` is the inverse probability the claim is mistaken, only for `observed` and `inferred`; a `held` claim gets no number, and how strongly a position is held belongs in `content`.
- [ ] 5.2 Synthesis guidance gains the method vocabulary, and the note that two conflicting positions still want a synthesis — `--method positions` — since the label changes what the work is, not whether there is work.
- [ ] 5.3 Update the example session and verb table; regenerate the `.claude` copy; both copies in sync.

## 6. Documentation

- [ ] 6.1 README verb table (`--evidential` in the assert row); a short evidential section in `docs/provenance.md`, which is where warrant already lives.
- [ ] 6.2 CHANGELOG under Unreleased, marked **BREAKING** for writers with the reader-compatibility statement.

## 7. Verification

- [ ] 7.1 `go build && go vet && golangci-lint run && go test ./...` clean, with the real-workspace round-trip guard.
- [ ] 7.2 Against a copy of the real workspace: `validate` gains exactly one aggregate `undeclared` line (88 objects) and the `confidence_on_undeclared` note for claims carrying confidence, no errors, no file rewritten, same warnings as before.
- [ ] 7.3 End to end: assert one claim of each evidential; recall shows the field; export a `held` claim and see the position marker; `held`+`confidence` refused at CLI and MCP; a hand-written file with both fails validate with exit 4 while `recall` still returns it.
