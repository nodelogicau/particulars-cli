## 1. Object model (`internal/dkf`)

- [ ] 1.1 Add `Claim.Basis []string` and encode it after `evidential` and before `confidence` in the canonical order. Decode `basis` on a synthesis into an ignored-on-write field, so the validator can see it and the encoder never emits it. Round-trip tests for both, plus a `non_canonical` test for a basis out of order.
- [ ] 1.2 Add `BasisIDs()` to the `Assertion` interface (nil for a synthesis) and a `Cited(a)` helper that returns `InputIDs()` then `BasisIDs()`. Leave `Claim.InputIDs()` returning nil, and add a test that pins this.
- [ ] 1.3 Object-level problems in `validate.go`: `basis_not_inferred` (a claim that isn't `inferred`, a claim with no evidential, or a synthesis), `empty_basis`, `self_basis` and `duplicate_basis`, at the severities in the `validation` delta. Tests for each.

## 2. Workspace checks (`internal/query/validate.go`)

- [ ] 2.1 Referential checks: a dangling basis id is `dangling_reference`, and a basis id naming a particular, merge or promotion is an error. Report both against the citing claim.
- [ ] 2.2 Extend the cycle check to walk `Cited`, so a loop through `basis` and `inputs` reports `cycle`, and a self-citation reports `self_basis` only.
- [ ] 2.3 `inferred_without_basis` as an aggregated informational corpus fact, following the `undeclared` pattern in text output. `stale_claim` as a per-object warning beside `stale_synthesis`.
- [ ] 2.4 Tests translating every scenario in the `validation` and `claim-evidential` deltas, including a file with `basis` on a `held` claim that `recall` still returns.

## 3. Conflict sets (`internal/query/conflicts.go`)

- [ ] 3.1 Generalise `CitesRetracted` to any `dkf.Assertion`, following `Cited`, with its memo kept as the cycle guard. Use it for every non-retracted member in `Analyse` and `AnalyseSet`. Leave `closure` on `inputs` only.
- [ ] 3.2 Tests for every `conflict-detection` and `mcp-server` scenario: a retracted basis member, staleness through a rule, a cross-particular basis member, leaving the stale set, instances staying unsynthesised, a synthesis over a rule, priority 3 for a stale unsynthesised rule, and a claim set whose basis member is outside the set.

## 4. Index (`internal/store/index.go`)

- [ ] 4.1 Add `IndexEntry.Basis` (`yaml:"basis,omitempty"`) after `inputs`, add `basis` to `knownEntryFields`, and fill it from `BasisIDs()`. Claim entries never carry `inputs`.
- [ ] 4.2 Tests: an inferred claim entry, `index --check` passing right after `claim assert --basis`, and a committed `basis` from before this change regenerating byte-for-byte.

## 5. Writing (`internal/cli/cmd_claim.go`, `internal/mcp/tools.go`)

- [ ] 5.1 A shared validator for a requested basis: `inferred` only, no duplicates, each id exists (not-found), is a claim or synthesis (usage), and is never resolved from a label. Return a `stale_claim` warning for each retracted id.
- [ ] 5.2 `claim assert --basis` (repeatable), with help text. Emit the `inferred_without_basis` warning, and run `scope_wider_than_inputs` after writing, as `synthesis create` does.
- [ ] 5.3 MCP `claim_assert`: a `basis` array parameter with a description that teaches it, the same refusals as usage or not-found tool errors, and the same warnings in the structured result.
- [ ] 5.4 Tests for every scenario in the `claims` delta, through both the CLI and MCP.

## 6. Scope (`internal/query/scope.go`, `internal/cli/cmd_publish.go`)

- [ ] 6.1 Generalise `ScopeWiderThanInputs` to any assertion: inputs for a synthesis, basis members for an inferred claim, with messages that say "basis member(s)" for a claim. Skip retracted claims.
- [ ] 6.2 `ScopeFindingsForPromotion` considers every non-retracted assertion that is covered or cites something covered through `Cited`.
- [ ] 6.3 Tests: a shared rule from personal instances at assertion, promoting a rule, and promoting a basis member clearing the condition on a claim the promotion didn't name.

## 7. Lineage and recall (`internal/query/lineage.go`, `recall.go`)

- [ ] 7.1 `Node.Basis []*Node` (JSON `basis`, always present). Expand a claim's basis with no role or weight. Basis edges consume depth. Replace the path guard with a tree-wide visited set: `repeated: true` stubs for objects already expanded, `cycle: true` for back-edges on the current path.
- [ ] 7.2 Text rendering labels basis children `basis`. MCP `lineage_trace` returns the same tree.
- [ ] 7.3 Recall `Entry.Basis` (`json:"basis,omitempty"`). `LineageOrder` treats basis members as predecessors.
- [ ] 7.4 Tests for every `knowledge-query` scenario, including the diamond expanded once, and updates to existing lineage tests whose fixtures contain diamonds.

## 8. Visual export (`internal/viz`)

- [ ] 8.1 `EdgeBasis` kind, from member to claim, labelled `basis`: dashed in DOT, `-.->` in Mermaid. Foreign marking for basis members outside the class, and depth counted along basis edges.
- [ ] 8.2 Stale claims take the stale state, and the map view picks them up through `rep.Stale`. Golden tests for DOT and Mermaid with a rule and its instances.

## 9. Guidance and docs

- [ ] 9.1 In `skills/particulars/SKILL.md`: recall before asserting `inferred`, pass what was reasoned from as `--basis`, a rule of thumb is an inferred claim whose basis is its instances, a counterexample is a `qualification` synthesis, a basis doesn't synthesise, and never `held` to avoid a basis.
- [ ] 9.2 Regenerate `.claude/skills/particulars/SKILL.md` with `skill install --force` from a binary built from HEAD (not the release binary, and not `cp`). Run the skill test.
- [ ] 9.3 Add `CHANGELOG.md` entries: `--basis` and the MCP parameter, the new validation codes, stale claims, recall and index `basis`, basis edges in export, and a **BREAKING (output)** note that lineage expands each object once.
- [ ] 9.4 Update the README command reference for `claim assert --basis` and the lineage output shape.

## 10. Verify

- [ ] 10.1 Run `go test ./...`, `golangci-lint run` and `openspec validate inferred-claim-basis --strict`.
- [ ] 10.2 Smoke-test against a copy of `~/IdeaProjects/particulars-knowledge`. `validate` should report only `inferred_without_basis` as new, and `conflicts` output should be unchanged.
- [ ] 10.3 Archive `recall-query` before archiving this change, then close #14 with a link to the commit.
