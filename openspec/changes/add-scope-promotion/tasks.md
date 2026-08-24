## 1. The record type

- [x] 1.1 Add `pub` to both regexes in `internal/dkf/id.go` (lenient **and** canonical), to `TypeOfID`, and to `IsRetractableID`; add `dkf.TypePublish`.
- [x] 1.2 Add `dkf.Promotion` with fields in spec order — `id`, `type`, `claims`, `scope`, `reason`, `source`, `timestamp`, `retracted` — and its codec node, mirroring `Merge`.
- [x] 1.3 `dkf.ValidatePromotion`: at least one id in `claims`, every id a `clm_` or `syn_`, a valid `scope`, `source` with author or harness.
- [x] 1.4 Round-trip tests: canonical byte-stable serialisation, field order, retracted block appended without disturbing it.

## 2. Store and effective scope

- [x] 2.1 `publishes/` in `dirFor`, `Init`, and workspace discovery; a workspace without the directory loads exactly as today.
- [x] 2.2 Load promotions into `store.Graph.Promotions`, and build `effective map[string]dkf.Scope` in the same pass as merge classes: for each non-retracted promotion, for each covered id, keep the widest scope.
- [x] 2.3 `(*Graph).EffectiveScope(id) dkf.Scope` — the built value, else the object's asserted scope. This is the only place effective scope is computed.
- [x] 2.4 `(*Graph).PromotionsFor(id)` returning the non-retracted promotions covering an object, sorted, so a message can name the one responsible.
- [x] 2.5 `CreatePromotion`: reject narrowing against **asserted** scope, reject unknown ids, reject particular and merge ids, write create-exclusively.
- [x] 2.6 `SortedPromotions`, and `Retract` accepting `pub_` ids.
- [x] 2.7 Tests: widest wins, retracted promotions ignored, no-promotions workspace is identical to today, narrowing refused, redundant promotion allowed.

## 3. The shared warning

- [x] 3.1 Extract `query.ScopeWiderThanInputs(g, s) []Finding` comparing **effective** scope on both sides, and have `validate` call it instead of its inline check.
- [x] 3.2 The message names each side's effective scope, and where that differs from the asserted scope, the promotion responsible.
- [x] 3.3 Tests: a promotion of an input clears the warning without either file changing; a promotion of the synthesis creates it; retracted syntheses are never warned; it is never an error.

## 4. Validation

- [x] 4.1 Promotion errors: dangling id, particular or merge in `claims`, empty `claims`, narrowing, missing source, id/prefix/directory disagreement.
- [x] 4.2 Warnings `promotion_of_retracted` and `duplicate_promotion`.
- [x] 4.3 Tests for each, including that `validate` exits 4 on the errors and 0 on the warnings.

## 5. Effective scope everywhere it is filtered

- [x] 5.1 `recall --scope` and the reported `scope` per entry.
- [x] 5.2 `topics --scope`.
- [x] 5.3 `export --format graph`: filter on effective scope, keep refusing `--scope personal`, and report each item's `scope` as the widest effective scope contributing.
- [x] 5.4 `export --format dot|mermaid --scope`.
- [x] 5.5 Index entries for promotions (`claims`, `scope`, `timestamp`) and a test that effective scope is computable from `index.yaml` alone.
- [x] 5.6 `conflicts` and `lineage` ignore promotions — assert it rather than assume it.
- [x] 5.7 Audit: grep for `Context.Scope` outside `internal/dkf` and the effective-scope builder; every remaining use must be deliberate (writing a file, or reporting an asserted scope in the index).

## 6. Surfaces

- [x] 6.1 `internal/cli/cmd_publish.go`: `publish <id>... --scope <s> [--reason]` with provenance flags, ids only, exit 3 on unknown, `--json` carrying the record and any findings.
- [x] 6.2 `retract` accepts `pub_`, refuses `--superseded-by` for it, and its usage message lists promotions.
- [x] 6.3 `synthesis create` evaluates the warning and returns it in the result **without blocking the write**.
- [x] 6.4 MCP `knowledge_publish`, matching the CLI result; the tool list grows to twelve.
- [x] 6.5 In-process CLI tests: publish, double-promote, retract-and-revert, label refused, unknown id, warning surfaced on create, and the whole flow ending in an export that now emits an item.

## 7. Documentation

- [x] 7.1 `docs/graph.md`: a personal workspace becomes exportable by promotion, with the widen-only reasoning and what retraction does and does not recall.
- [x] 7.2 `docs/visualise.md`: `--scope` is effective scope.
- [x] 7.3 README verb table: `publish`.
- [x] 7.4 `skills/particulars/SKILL.md`: promotion as the second remedy for `scope_wider_than_inputs` — and usually the better one, since promoting the inputs keeps the conclusion shareable where re-scoping the synthesis demotes it. Regenerate the `.claude` copy.
- [x] 7.5 CHANGELOG under Unreleased.

## 8. Verification

- [x] 8.1 `go build ./... && go vet ./... && golangci-lint run && go test ./...` all clean.
- [x] 8.2 Against a copy of the real knowledge workspace: promote the four harness syntheses and their inputs, confirm `export --format graph` grows from 3 items, then retract and confirm it returns to 3.
- [x] 8.3 Confirm the three `scope_wider_than_inputs` warnings in that workspace clear by promoting the narrower inputs, with no object file modified.
