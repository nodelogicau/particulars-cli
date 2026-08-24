## MODIFIED Requirements

### Requirement: Scope governs what leaves the workspace
Assertions whose **effective** scope is `personal` SHALL NOT be exported, and SHALL NOT contribute to any item's brief, properties, or counts. Effective scope is the widest scope named by a non-retracted promotion covering the object, or its asserted `context.scope` when none does. A particular with no exportable assertions SHALL produce no item. Exported items SHALL carry `acl: [{"type": "everyone", "value": "everyone", "accessType": "grant"}]`. `--scope <s>` SHALL narrow the export further and SHALL NOT be able to widen it to include `personal`. An item's reported `scope` property SHALL be the widest effective scope contributing to it.

#### Scenario: Personal claims are withheld
- **WHEN** a particular has one `personal` claim and one `organisation` claim and the export runs
- **THEN** the item's brief contains the organisation claim and no text from the personal claim, and `claimCount` is 1

#### Scenario: Promotion makes a personal workspace exportable
- **WHEN** every assertion about a particular is asserted `personal`, and its current synthesis and claims are promoted to `organisation`
- **THEN** the export emits an item carrying that belief and those claims, though no object file changed

#### Scenario: Retracting a promotion withdraws the item
- **WHEN** the promotion that made a particular's only assertions exportable is retracted
- **THEN** the next export emits no item for it and its id is absent from the manifest
