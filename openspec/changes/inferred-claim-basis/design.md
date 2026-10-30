## Context

The upstream change is archived at nodelogicau/particulars `openspec/changes/archive/2026-10-30-inferred-claim-basis/`, and its specs are synced into `claim-basis`, `claim-evidential`, `conflict-semantics`, `index-manifest` and `scope-promotion`. This repo's issue #14 lists the work, and its review comment adds the gaps between the issue and the spec.

Today only a synthesis cites other objects. Several parts of the CLI assume that "cites" means "synthesis `inputs`":

- `CitesRetracted` takes a `*dkf.Synthesis`.
- `ScopeWiderThanInputs` and `ScopeFindingsForPromotion` loop over syntheses only.
- `lineage.expand` returns early for a claim.
- `Claim.InputIDs()` returns nil, and `closure`, the index `inputs` field and recall's `LineageOrder` all rely on that.
- The viz model has a single `EdgeInput` kind.
- The validator's cycle check walks synthesis inputs only.

## Goals / Non-Goals

**Goals:**
- Write, read, check and traverse `basis` as the DKF spec defines it, through both the CLI and MCP.
- Keep `unsynthesised` exactly as it is.
- Leave existing workspaces valid, with no new errors and one aggregated corpus fact.

**Non-Goals:**
- Marking in `recall` that an entry is the basis of another entry. Upstream leaves this open, and this change carries only the forward `basis` field.
- Graph export (`export --format graph`). The brief describes current belief, and a basis doesn't change belief.
- `dkf/0.2` and the root `retracted` list. Those are deferred (particulars-cli#11).
- Judging whether a basis actually supports its claim.

## Decisions

**Basis is its own accessor, and `InputIDs()` stays synthesis-only.** `Claim` gets a `Basis []string` field. A new `Assertion` method, `BasisIDs()`, returns nil for a synthesis. A separate helper, `Cited(a)`, returns `InputIDs()` followed by `BasisIDs()`. Staleness, cycle detection, scope checks and promotion findings use `Cited`. `closure` and the index `inputs` field keep using `InputIDs()`. *Alternative:* return the basis from `Claim.InputIDs()`. That is the smallest diff, but it would silently treat basis members as synthesised and write `inputs` on claim entries, which is the exact confusion the upstream design rejected.

**`basis_not_inferred` needs a field the decoder keeps on syntheses.** `Synthesis` gets an unexported or ignored-on-write `Basis` field, so the validator can see a hand-written `basis` on a synthesis and report it. The encoder never emits it. A synthesis carrying one is still read and still citable. The same check covers a claim whose `evidential` is `observed`, `held` or absent.

**Write-time refusals.** For both `claim assert` and `claim_assert`:
- A basis with an evidential other than `inferred` is a usage error (exit 2).
- A duplicate id is a usage error. The spec doesn't say whether duplicates are allowed. Refusing them is the conservative choice: the list is a set of evidence, and a duplicate is always a mistake.
- An id that doesn't exist is not-found (exit 3), matching `--superseded-by`.
- An id that names a particular, merge or promotion is a usage error.
- A retracted id is accepted, with a warning that the new claim is stale at once. The spec allows citing one, so refusing would be wrong, and staying silent would hide something the author can still act on.
- Only ids are accepted, not labels. A basis names exact objects, as `publish` does.

**The warnings travel in the result.** `inferred_without_basis`, `scope_wider_than_inputs` and the stale-at-birth warning use the existing `warnings` channel, the same one `quote_drift` uses at assertion.

**`stale` covers claims, and priority keeps its formula.** `CitesRetracted` works on any assertion, follows `Cited`, and keeps its memo as a cycle guard. Both `Analyse` and `AnalyseSet` add every non-retracted member that cites a retracted object. Priority stays `|unsynthesised| + |stale|`. A stale inferred claim is usually unsynthesised too, so it counts twice, but it really does need two actions: a revised rule and a synthesis. Non-current stale syntheses already count twice today. *Alternative:* count the union. That changes ordering for existing workspaces, which the spec doesn't ask for.

**Validate warns `stale_claim` beside `stale_synthesis`.** It is a per-object finding, since a reader acts on that claim. Keeping the two codes separate means scripts that filter on `stale_synthesis` keep their meaning.

**Lineage: a `basis` child list, with each object expanded once.** `Node` gets `Basis []*Node` (JSON `basis`, always present like `inputs`). Basis children carry no `role` or `weight`, and the text tree labels them `basis`. A separate list was chosen over `inputs` entries with a `relation: basis` field so that consumers reading `inputs` see synthesis inputs only, which is what they see today. The spec says traversal visits each object at most once. So the path set becomes a tree-wide visited set. A later occurrence is emitted as a stub with `repeated: true` and is not expanded. `cycle: true` stays for a back-edge to an object still on the current path. Depth is consumed the same way along both edge kinds.

**Recall and lineage order.** Recall entries for a claim carry `basis` when the file does, in the same way they carry `inputs`. `LineageOrder` treats basis members as predecessors, so the evidence comes before the claim, as inputs come before a synthesis.

**The index entry field.** `IndexEntry.Basis []string` (`yaml:"basis,omitempty"`) goes after `inputs`, and `basis` joins `knownEntryFields`. A committed `basis` from a newer writer is already preserved as an unknown field. Once this lands, it is regenerated from the file instead.

**Scope checks generalise to the evidence an object cites.** `ScopeWiderThanInputs` becomes a function over any assertion with cited members: inputs for a synthesis, basis for an inferred claim. The message names "basis member(s)" for a claim. `ScopeFindingsForPromotion` considers every non-retracted assertion that is covered or cites something covered. `claim assert` runs the check after writing, as `synthesis create` does.

**Cycles are checked over the combined graph.** The validator's cycle check walks `Cited`, so a hand-edited loop through basis and inputs is reported as `cycle`. A self-citing basis gets its own `self_basis` error, because "cycle of length one" sends a reader to the wrong fix.

**Visual export.** A new `EdgeBasis` kind is drawn from the basis member to the claim and labelled `basis`. It is dashed in DOT and dotted (`-.->`) in Mermaid, so evidence reads differently from argument. Basis members outside the class are included and marked foreign, as cross-particular inputs are. `--depth` counts basis edges as levels.

## Risks / Trade-offs

- [The visit-once lineage changes output for diamonds] → Any consumer that walked the tree to collect ids still sees every id once. The stub keeps the id and type. This goes in the CHANGELOG as a behaviour change.
- [Stale inferred claims add `conflicts --fail-on-conflicts` failures in CI] → Only for new claims with a basis whose members are later retracted, which is the condition the change exists to surface.
- [Open overlap with `recall-query`] → This change avoids `mcp-server` *Spec tool surface* and modifies *Recall entries carry the warrant*, not *Recall by query*. Archive `recall-query` first anyway, so the syncs apply in order.
- [Older CLIs ignore `basis`] → They miss staleness through it but read every file, and they keep the index field. Their sets are a subset of the new ones.

## Migration Plan

No data migration. Ship in a minor release. The CHANGELOG notes the new flag, the new conditions, stale claims, and the lineage visit-once change.

## Open Questions

None blocking. Upstream's open question about marking basis members in `recall` stays open here too.
