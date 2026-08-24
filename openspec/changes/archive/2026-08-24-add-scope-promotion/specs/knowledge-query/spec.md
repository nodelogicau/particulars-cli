## MODIFIED Requirements

### Requirement: Scope filter
`--scope <s>` SHALL restrict results to objects whose **effective** scope equals `s` — the widest scope named by a non-retracted promotion covering the object, or its asserted `context.scope` when none does. The `scope` reported on each entry SHALL be the effective scope. In a workspace with no promotion records this SHALL be identical to filtering on `context.scope`.

#### Scenario: Filtering by scope
- **WHEN** `recall "Project X" --scope public` is run
- **THEN** only objects whose effective scope is `public` are returned

#### Scenario: A promoted claim becomes visible to a wider filter
- **WHEN** a claim asserted `personal` is promoted to `organisation` and `recall --scope organisation` is run
- **THEN** the claim is returned and its reported `scope` is `organisation`

#### Scenario: Topics honour effective scope
- **WHEN** `topics --scope organisation` is run in a workspace where a `personal` claim carrying topic `billing` has been promoted to `organisation`
- **THEN** `billing` is counted
