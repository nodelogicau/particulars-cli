## Context

Three things already exist that this builds on. `dkf.ScopeRank`/`ScopeAtLeast` give the ordering. Merge records are a working precedent for a non-knowledge record type: a file under its own directory, a `mrg_` prefix, retractable, an index entry, and a validation pass — promotions are the same shape with different fields, and should look like them wherever there is a choice. And `scope_wider_than_inputs` already exists in `validate`, comparing asserted scopes; this change moves it rather than writing it.

The constraint that shapes the design is that **effective scope is workspace state, not a property of a file**. Every existing scope decision reads one object: `claim.Context.Scope`, and that is all. After this change no correct scope decision can be made from an object alone — it needs the promotion set too. That is a change in kind, not degree, and it is where the bugs will be: any surviving `a.GetContext().Scope` in a filter is a latent false answer that only appears once someone promotes something.

## Goals / Non-Goals

**Goals:**

- One place computes effective scope, and every filter calls it.
- Widen-only enforced at write time and at validate, so the invariant that makes partial implementations safe cannot be violated by our own tools.
- `scope_wider_than_inputs` evaluated at all three points the spec names, from one shared function, so the three cannot disagree.
- A workspace with no `publishes/` behaves byte-for-byte as it does today.

**Non-Goals:**

- An acknowledgement or suppression mechanism — upstream deferred it deliberately.
- Cascading promotion, or a `--with-inputs` convenience. The spec makes non-cascading the default *because* cascading widens a lineage silently; a flag that does it in one step is the same hazard with a nicer name. The `publish` verb takes several ids, so promoting a chain is one command with the ids written out — deliberate, and visible in the record.
- Promoting particulars. The spec's `claims` list is claims and syntheses; a particular carries no scope.
- Narrowing. Retracting the promotion is how exposure is reduced, and that is the spec's answer.

## Decisions

### Effective scope is computed once, on the loaded graph, and cached

`store.Graph` gains `effective map[string]dkf.Scope`, built during `Load` in the same pass that builds merge classes: for each non-retracted promotion, for each covered id, keep the widest scope seen. `g.EffectiveScope(id)` returns it, falling back to the object's asserted scope. Filters call that and never read `Context.Scope` directly.

*Alternative considered:* computing it per query from the promotion list. Rejected — `recall` and the exports would each rebuild it, and the temptation to inline "just this once" is exactly how the two would diverge. A map built at load is O(promotions) and removes the choice.

*Consequence to accept:* a `Graph` is a snapshot. A promotion written after a load is not visible to that load — true of every other object here, so it is consistent rather than surprising.

### Widen-only is checked against **asserted** scope, deliberately

The spec says a record naming a scope narrower than an object's *asserted* scope is invalid — asserted, not effective. So promoting a `personal` claim to `organisation` twice is fine, and a second promotion to `organisation` after one to `public` is also valid: it is redundant, not narrowing, because effective scope is the widest and stays `public`. Checking against effective scope instead would reject that second record and would make validity depend on the order records were written, which is not a property a file should have.

Redundancy is reported as a warning (`duplicate_promotion`), never an error.

### The three evaluation points share one function

`query.ScopeWiderThanInputs(g, s) []Finding` returns the warning for one synthesis. `validate` calls it for every non-retracted synthesis; `synthesis create` calls it for the synthesis it just wrote; `publish` calls it for each promoted synthesis **and** for every non-retracted synthesis citing a promoted object — because promoting an input can *clear* the condition on a synthesis nobody named, and promoting a synthesis can create it.

At `synthesis create` the warning is attached to the result and **never blocks the write**. The spec is explicit that implementations must not reject; a create that failed on it would push authors toward asserting standalone claims with no inputs, destroying the lineage the format exists to keep.

### The message names why a scope is what it is

`scope_wider_than_inputs` currently says "is organisation but reasons from narrower input(s) X (personal)". With promotions, the same sentence can be true for four different reasons, and a reviewer needs to know which. The message states each side's effective scope and, when it differs from the asserted one, that a promotion is responsible and which. A warning that says "public vs personal" without saying "because pub_… promoted it" sends the reader to the wrong file.

### `publish` takes ids only — no labels, no `--all`

`particular resolve` accepts labels because particulars are things people name. Claims are not: promoting by anything other than an explicit id invites promoting the wrong one, and promotion is the one operation here whose mistakes are not recoverable by retraction — the record can be retracted, but whoever fetched the feed already has the content. Ids are cheap to obtain from `recall --json`.

### Retraction reuses the existing verb, and refuses `--superseded-by`

Exactly as merges do: a promotion is undone, not superseded. `IsRetractableID` gains `pub_`, and the usage message that lists retractable kinds is updated with it.

## Risks / Trade-offs

- **A missed filter silently under- or over-shares.** → The risk is asymmetric and the format is built around that: a filter still reading asserted scope *under*-shares, which is a bug report rather than a breach. A test asserts that no filter path returns a promoted object as still-narrow, and the audit is mechanical: grep for `Context.Scope` outside `internal/dkf` and the effective-scope builder.
- **`export --format graph` changes what it emits for an existing workspace** — that is the point of the change, but it means a sync job's next run can publish a great deal at once. → The export already reports `exported`/`skipped`, `--manifest` diffs deletions, and promotion is an explicit act with a record naming who did it and when. Nothing is promoted without someone writing a record.
- **Validate gets slower**: effective scope is another map, and the warning now runs over promotions too. → Both are linear in objects and built in the existing load pass.
- **The warning may fire on a workspace where it did not before, or stop firing without either file changing.** That will look like a bug. → Documented in `docs/` and in the message itself, which names the promotion responsible.
- **Two ways to widen exposure with very different ceremony** — a promotion is sourced, timestamped and retractable; asserting a synthesis wider than its inputs is a field value. Upstream noticed this and deferred acting on it. → Not resolved here either; the warning is the only guard, and that is the spec's decision, recorded so it is not re-litigated by accident.

## Open Questions

- Whether `workspace_status` should report a promotion count. It is the natural place to notice "this workspace has published things", but the tool's output is already dense; deferred until the verb has been used enough to know.
- Whether `lineage` should show promotions on a node. They are not provenance — nothing about the reasoning changes — but they do change who can see it. Left out for now on the grounds that lineage answers "why is this believed", not "who can read it".
