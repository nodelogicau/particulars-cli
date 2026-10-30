## Why

DKF now lets an `inferred` claim name its `basis`: the claims and syntheses it was inferred from (nodelogicau/particulars#43, resolving nodelogicau/particulars#42). Staleness reaches claims through `inputs` and `basis`, and lineage follows the basis. A rule of thumb is now an inferred claim whose basis is its instances. The reference implementation can't write, check or traverse the field yet, so an agent can only state its warrant in `content` (#14).

## What Changes

- `claim assert --basis <id>` (repeatable) and a `basis` parameter on MCP `claim_assert`. The basis is written after `evidential` and before `confidence`.
- Writers refuse a basis unless `evidential` is `inferred`. They also refuse a duplicate id and an id that isn't an existing claim or synthesis. A retracted id is allowed, and the response warns that the claim is stale from the start. An `inferred` claim with no basis is written with an `inferred_without_basis` warning.
- `scope_wider_than_inputs` also covers an inferred claim wider than any basis member. It is evaluated at assertion, on `publish` (including when a basis member is promoted), and in `validate`.
- `validate` gains new checks:
  - `basis_not_inferred`: an error, but the file is still read.
  - Errors for an empty basis (`empty_basis`), a dangling basis id, a basis id that names a particular, merge or promotion, a self-citing basis (`self_basis`), and cycles through `inputs` and `basis` together.
  - `inferred_without_basis`: an aggregated corpus fact, like `undeclared`.
  - `stale_claim` and `duplicate_basis`: warnings. `stale_claim` sits alongside `stale_synthesis`. A duplicate is refused on write, but the format doesn't forbid it, so a hand-written file is warned rather than failed.
- `conflicts` and `conflict_detect`: `stale` now includes claims, through `inputs` and `basis`, transitively. `unsynthesised` is unchanged, because `closure` follows `inputs` only.
- `lineage` and `lineage_trace` gain a `basis` child list beside `inputs`. Each object is expanded at most once, and a repeat is marked `repeated: true` instead of being expanded again. **BREAKING (output):** a diamond in the provenance graph used to be expanded once per path.
- The index adds `basis` to claim entries that carry one. Recall entries carry `basis`, and lineage order puts basis members before the claim that cites them.
- `export --format dot|mermaid` draws basis edges, labelled `basis` and styled differently from input edges.
- Promotion never cascades to a basis.
- The skill text changes: before asserting an `inferred` claim, recall and pass the claims reasoned from as `--basis`. A rule of thumb is an inferred claim whose basis is its instances, and a counterexample is a `qualification` synthesis. The installed skill copy is regenerated.

## Capabilities

### New Capabilities

None. The upstream `claim-basis` capability is implemented through the existing capabilities below.

### Modified Capabilities

- `claims`: *Assert a claim* gains `--basis` and its write-time rules, and the MCP `claim_assert` parameter.
- `claim-evidential`: *Every new claim declares what backs it* says an inferred claim may name its basis and gives its place in field order.
- `conflict-detection`: *Structural conflict detection* redefines `stale` to include claims through `inputs` and `basis`, and says a basis doesn't synthesise.
- `knowledge-query`: *Lineage trace* follows `basis` and expands each object once. *Recall entries carry the warrant* adds `basis`.
- `index`: *Index structure* adds `basis` to claim entries.
- `validation`: *Referential checks*, *Evidential findings* and *Advisory warnings* gain the basis conditions and `stale_claim`.
- `scope-promotion`: *Promotion does not cascade* covers a basis, and a new requirement treats a basis as inputs for `scope_wider_than_inputs`.
- `mcp-server`: *Conflict detection over a claim set* puts claims in `stale` through `inputs` and `basis`.
- `visual-export`: *The lineage view*, *Conflict state is rendered* and *Depth bounds the lineage view* cover basis edges and stale claims.
- `skill-distribution`: a new requirement covers the skill's guidance on basis.

## Impact

- `internal/dkf`: `Claim.Basis`, canonical encoding, and object-level checks (`basis_not_inferred`, empty basis, duplicates). `Claim.InputIDs()` is unchanged.
- `internal/query`: `conflicts.go` (stale over claims), `lineage.go` (basis children, visit once), `scope.go` (claims in the scope check and in promotion findings), `validate.go` (reference, type, self-citation and cycle checks, plus the new codes), `recall.go` (the `basis` field and ordering).
- `internal/store/index.go`: a `basis` entry field, added to `knownEntryFields`.
- `internal/cli/cmd_claim.go`, `internal/mcp/tools.go`: the flag, the parameter, the refusals and the warnings.
- `internal/viz`: a `basis` edge kind.
- `skills/particulars` and `.claude/skills/particulars/SKILL.md`.
- No migration. Existing workspaces stay valid, and their inferred claims are reported in aggregate as `inferred_without_basis`.
- Overlap with the open `recall-query` change: both modify `knowledge-query` and `mcp-server`, but different requirements. `recall-query` modifies *Recall by query* and *Spec tool surface*; this change leaves *Spec tool surface* alone and specifies the `claim_assert` parameter in `claims`. Archive `recall-query` first.
