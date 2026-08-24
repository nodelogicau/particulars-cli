# Graph Export

## Purpose

Emitting a workspace as Microsoft Graph connector payloads so that merged, organisation-scoped knowledge is searchable and citable in Microsoft 365 Copilot: item shape and identity, the rendered belief brief, scope filtering and ACLs, exclusion of retracted objects, properties and semantic labels, the schema payload, and the manifest that drives deletion.

## Requirements

### Requirement: Export Graph items
`particulars export --format graph [--source-url <base>] [--out <path>] [--manifest <path>] [--scope <s>]` SHALL write one JSON object per line (NDJSON), each of the form `{"id": <particular id>, "item": {"acl": [...], "properties": {...}, "content": {"value": <brief>, "type": "text"}}}`, one per exported particular, ordered by particular id. With `--out` it SHALL write to that file, otherwise to stdout; with `--manifest` it SHALL additionally write the exported ids, one per line, to that path. `--json` SHALL emit a summary object (`{exported, skipped, path?}`) rather than the items. The command SHALL NOT perform any network request.

#### Scenario: Items for a workspace
- **WHEN** `export --format graph` runs in a workspace with two particulars carrying organisation-scoped claims
- **THEN** stdout contains exactly two NDJSON lines, ordered by particular id, each with `id`, `acl`, `properties`, and `content`

#### Scenario: Deterministic output
- **WHEN** the export is run twice against an unchanged workspace
- **THEN** both runs produce byte-identical output

#### Scenario: Offline
- **WHEN** the export runs with no network available
- **THEN** it succeeds

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

### Requirement: Retracted knowledge is never exported
Retracted claims, syntheses, and merge records SHALL be excluded from every item. A retracted synthesis SHALL NOT be treated as the current belief. Content that was retracted after a previous export SHALL be absent from the next export of that particular.

#### Scenario: Retracted claim disappears
- **WHEN** a claim is retracted and the export is re-run
- **THEN** its content no longer appears in the particular's brief and `claimCount` decreases

#### Scenario: Retracted synthesis is not the belief
- **WHEN** the most recent synthesis about a particular is retracted and an older non-retracted one exists
- **THEN** the brief presents the older synthesis as the current belief

### Requirement: The brief carries belief and caveats
Each item's `content.value` SHALL begin with the particular's label and URI, then present the current synthesis's content under a belief heading with its id, timestamp, and confidence when set; the synthesis's `unresolved` under a heading naming what was not reconciled; and the exported claims with their content, confidence when set, and `source.document` when set, marking those not reconciled into the current synthesis. When there is no current synthesis the belief section SHALL be replaced by a statement that none exists and how many claims are unreconciled. Empty sections SHALL be omitted.

#### Scenario: Belief with open questions
- **WHEN** a particular has a current synthesis with `unresolved: "Compliance basis unsourced."` and two supporting claims
- **THEN** the brief contains the synthesis content, the text `Compliance basis unsourced.`, and both claims with their evidence

#### Scenario: Unsynthesised claim marked
- **WHEN** a claim was asserted after the current synthesis and is not among its transitive inputs
- **THEN** that claim's line in the brief is marked as unsynthesised

#### Scenario: No synthesis yet
- **WHEN** a particular has three claims and no synthesis
- **THEN** the brief states that no synthesis exists and that three claims are unreconciled

#### Scenario: Conventional empty unresolved
- **WHEN** the current synthesis has `unresolved: "None identified"`
- **THEN** the brief still shows it, so the reader can tell it was considered

### Requirement: Item properties and labels
Each item's `properties` SHALL contain `title` (the particular's label), `particularUri`, `scope` (the highest scope among its exported assertions, `public` over `organisation`), `topics` (the distinct topics of its exported assertions, with an `@odata.type` of `Collection(String)`), `authors` (distinct non-empty `source.author` values, likewise typed), `lastModifiedDateTime` (the greatest exported timestamp, ISO 8601), `claimCount`, `openQuestions` (the number of unsynthesised assertions), and `currentSynthesis` (the id, when one exists). With `--source-url <base>` it SHALL also contain `url`, formed by joining the base with the workspace-relative path of the current synthesis's file, or of the newest exported claim when no synthesis exists.

#### Scenario: Properties present
- **WHEN** an item is exported for a particular with topics `architecture` and `db`
- **THEN** `properties.topics` is `["architecture", "db"]` with `topics@odata.type` of `Collection(String)`, and `lastModifiedDateTime` is the greatest assertion timestamp in ISO 8601

#### Scenario: URL points at the reviewed file
- **WHEN** `--source-url https://github.com/o/r/blob/main/` is given and the particular has a current synthesis
- **THEN** `properties.url` is that base joined with `syntheses/syn_….yaml`

#### Scenario: No source URL
- **WHEN** `--source-url` is omitted
- **THEN** no `url` property is emitted

### Requirement: Schema and connection payloads
`particulars export --format graph --schema --connection <id> [--name <n>] [--description <d>]` SHALL emit a JSON object containing a `connection` body (`id`, `name`, `description`) and a `schema` body whose `baseType` is `microsoft.graph.externalItem` and whose `properties` declare every property the item export emits, with flags that satisfy Microsoft's constraints: no property both searchable and refinable, `isExactMatchRequired` only on non-searchable properties, and every labelled property retrievable. It SHALL NOT perform any network request.

#### Scenario: Schema is emitted
- **WHEN** `export --format graph --schema --connection particulars` is run
- **THEN** the output contains a connection body with `id: particulars` and a schema declaring `title`, `url`, `particularUri`, `topics`, `scope`, `lastModifiedDateTime`, `authors`, `claimCount`, `openQuestions`, and `currentSynthesis`

#### Scenario: Flags respect Microsoft's constraints
- **WHEN** the emitted schema is inspected
- **THEN** no property has both `isSearchable` and `isRefinable` true, every property with a label has `isRetrievable` true, and any property with `isExactMatchRequired` true has `isSearchable` false

### Requirement: Manifest supports deletion
With `--manifest <path>`, the export SHALL write the exported item ids, one per line, sorted. A caller comparing a previous manifest with the current one SHALL be able to determine exactly which items to delete.

#### Scenario: Particular drops out of scope
- **WHEN** a particular's only organisation claim is retracted and the export is re-run with a manifest
- **THEN** the new manifest omits that particular's id while the previous manifest contains it
