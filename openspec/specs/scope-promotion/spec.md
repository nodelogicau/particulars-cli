# Scope Promotion

## Purpose

Sharing immutable knowledge more widely without rewriting it: the promotion record and its file layout, effective scope and its ordering, the widen-only and no-cascade rules, the `publish` verb and its MCP tool, retraction and reversion, and the validation that keeps a promotion honest. A claim's scope is fixed when it is written; an object's *effective* scope is that scope widened by the non-retracted promotions covering it.

## Requirements

### Requirement: Promotion records
A promotion record SHALL be stored at `publishes/pub_<uuidv7>.yaml` with fields serialised in the order `id`, `type` (`publish`), `claims`, `scope`, `reason` (optional), `source`, `timestamp`, `retracted` (optional). `claims` SHALL list at least one claim or synthesis id and SHALL NOT contain a particular or merge id. `scope` SHALL be a valid scope. `source` SHALL carry at least one of `author` or `harness`, as every other record does. The `pub` prefix SHALL be accepted by both the lenient and the canonical identifier patterns, so that a minted promotion id is never reported as `legacy_id`.

#### Scenario: Written in canonical form
- **WHEN** a promotion is created
- **THEN** the file is `publishes/<id>.yaml`, its keys appear in the order above, and re-reading and re-serialising it produces identical bytes

#### Scenario: Claims must be assertions
- **WHEN** a promotion names a particular id or a merge id in `claims`
- **THEN** the operation is refused and no file is written

#### Scenario: At least one object
- **WHEN** a promotion is created with an empty `claims` list
- **THEN** the operation is refused

### Requirement: Effective scope
An object's **effective scope** SHALL be the widest scope named by a non-retracted promotion record covering it, or its asserted `context.scope` when none does, ordering `personal` < `organisation` < `public`. Effective scope SHALL be computed from the object together with the promotion records and never from `dkf.yaml`. In a workspace with no promotion records, every object's effective scope SHALL equal its asserted scope.

#### Scenario: Widest promotion wins
- **WHEN** a `personal` claim is covered by one promotion to `organisation` and another to `public`
- **THEN** its effective scope is `public`

#### Scenario: Retracted promotions do not count
- **WHEN** the only promotion covering a `personal` claim is retracted
- **THEN** the claim's effective scope is `personal` again

#### Scenario: No promotions at all
- **WHEN** a workspace contains no `publishes/` directory
- **THEN** every object's effective scope equals its asserted scope and every scope-filtered result is unchanged

### Requirement: Promotion may only widen
A promotion naming a scope narrower than the **asserted** scope of any object it covers SHALL be rejected as invalid, both when written and by `validate`. Comparison SHALL be against asserted scope, not effective scope, so that a record's validity does not depend on the order in which records were written. A promotion that names a scope already reached by another promotion SHALL be permitted and reported as a warning, not an error.

#### Scenario: Narrowing is refused
- **WHEN** a promotion names `organisation` for a claim asserted `public`
- **THEN** the operation is refused and `validate` reports an error for such a record

#### Scenario: Redundant promotion is allowed
- **WHEN** a claim already promoted to `public` is promoted again to `organisation`
- **THEN** the record is written, its effective scope remains `public`, and `validate` reports a warning rather than an error

### Requirement: Promotion does not cascade
Promoting a synthesis SHALL NOT promote the assertions it cites. Implementations SHALL NOT offer an option that promotes a synthesis together with its inputs in one act, because that is the silent lineage-widening the rule exists to prevent; promoting a chain SHALL require naming each object.

#### Scenario: Inputs stay where they were
- **WHEN** a synthesis citing two `personal` claims is promoted to `organisation`
- **THEN** the synthesis's effective scope is `organisation` and both claims remain `personal`

### Requirement: Publish verb
`particulars publish <id>... --scope <s> [--reason <text>] [provenance flags]` SHALL write one promotion record covering the named objects. It SHALL accept claim and synthesis ids only, and SHALL NOT resolve labels, URIs, or aliases — promotion is the one operation whose mistakes cannot be undone for a consumer that has already fetched the result, so the caller SHALL name exactly what is promoted. An id that does not exist SHALL exit 3. With `--json` the result SHALL carry the record and any `scope_wider_than_inputs` findings the promotion caused or cleared.

#### Scenario: Promoting two objects
- **WHEN** `publish clm_… syn_… --scope organisation` is run
- **THEN** one record is written naming both, and both report an effective scope of `organisation`

#### Scenario: Labels are not accepted
- **WHEN** `publish "Project X" --scope public` is run
- **THEN** the command exits with a usage error naming the id requirement

#### Scenario: Unknown id
- **WHEN** a named id does not exist in the workspace
- **THEN** the command exits 3 and writes no record

### Requirement: Promotions are retractable and reversible
A promotion SHALL be retractable through the same verb as any other record, and `--superseded-by` SHALL be refused for it as it is for merges: a promotion is undone, not superseded. After retraction the objects it covered SHALL revert to the effective scope they would have had without it.

#### Scenario: Retraction reverts exposure
- **WHEN** the promotion that made a claim `public` is retracted
- **THEN** the claim's effective scope returns to its asserted value and it no longer appears in scope-filtered results

#### Scenario: Superseding a promotion is refused
- **WHEN** `retract pub_… --superseded-by pub_…` is run
- **THEN** the command exits with a usage error

### Requirement: Scope filters use effective scope
Every operation that filters or reports scope SHALL use effective scope: `recall --scope`, `topics --scope`, `export --format graph`, and `export --format dot|mermaid --scope`. `export --format graph` SHALL continue to refuse `--scope personal` and SHALL continue to exclude objects whose effective scope is `personal`. A scope reported in output SHALL be the effective scope.

#### Scenario: A promoted workspace becomes exportable
- **WHEN** every claim about a particular is asserted `personal`, the particular's current synthesis and its claims are promoted to `organisation`, and `export --format graph` runs
- **THEN** an item is emitted carrying that belief and those claims, though no object file changed

#### Scenario: Recall filters on effective scope
- **WHEN** a `personal` claim has been promoted to `organisation` and `recall --scope organisation` runs
- **THEN** the claim is returned

#### Scenario: Personal is still never exported
- **WHEN** an object's effective scope is `personal`
- **THEN** it does not appear in `export --format graph` output under any flag

### Requirement: Promotions are not knowledge
`conflicts` SHALL ignore promotion records entirely: they form no equivalence class, contribute to no conflict set, and change no priority. `lineage` SHALL NOT treat a promotion as provenance.

#### Scenario: Conflict sets are unchanged by promotion
- **WHEN** a particular's claims and syntheses are promoted
- **THEN** `conflicts` reports exactly what it reported before

### Requirement: Promotion validation
`validate` SHALL report an error when a promotion's `claims` list is empty, names an id that does not resolve, names a particular or merge, or when the record narrows any covered object's asserted scope; and when its `source` carries neither `author` nor `harness`, or its id, prefix, and directory disagree. It SHALL report a warning when a non-retracted promotion covers a retracted object (`promotion_of_retracted`), and when two non-retracted promotions name the same object at the same scope (`duplicate_promotion`).

#### Scenario: Dangling promotion
- **WHEN** a promotion names an id with no file
- **THEN** `validate` reports an error and exits 4

#### Scenario: Promotion of a retracted claim
- **WHEN** a non-retracted promotion covers a retracted claim
- **THEN** `validate` reports a `promotion_of_retracted` warning and exits 0 if there are no errors

### Requirement: The MCP publish tool
The MCP server SHALL expose `knowledge_publish(claim_ids, scope, source, reason?)`, writing a promotion record with the same rules and returning the same result as the CLI verb, including any `scope_wider_than_inputs` findings.

#### Scenario: Promoting over MCP
- **WHEN** a client calls `knowledge_publish` with two claim ids and `scope: organisation`
- **THEN** one record is written and the result reports both objects at effective scope `organisation`

### Requirement: Promoting a quoted claim reports the disclosure
Promotion SHALL report, without refusing, when any object it covers carries a document `quote`. A quote reproduces its source verbatim, so widening a quoted claim's scope publishes that source text in full — unlike a synthesis, which summarises what it cites. The report SHALL name the objects concerned.

#### Scenario: Quoted claim promoted
- **WHEN** a claim carrying a quote is promoted to `public`
- **THEN** the result names it and states that verbatim source text is now published, and the promotion succeeds

#### Scenario: Unquoted promotion is silent
- **WHEN** no promoted object carries a quote
- **THEN** no disclosure is reported
