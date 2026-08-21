# Knowledge Query

## Purpose

Recalling claims and syntheses in lineage order with filters, and tracing the provenance tree of any assertion.

## Requirements

### Requirement: Recall by particular
`particulars recall <particular> [--include-retracted] [--limit <n>] [--scope <s>] [--topic <t>]...` SHALL return all claims and syntheses whose `subject` is any member of the resolved particular's merge equivalence class, in lineage order: a topological order in which every object precedes any synthesis that cites it, with ties broken by ascending id. Retracted objects SHALL be excluded unless `--include-retracted` is given. The `current` synthesis of the class (most recent by `timestamp`, then id, among non-retracted syntheses) SHALL be marked `current: true`; every non-retracted entry that is neither `current` nor in its transitive inputs SHALL be marked `unsynthesised: true`. Each entry SHALL include `id`, `type`, `subject`, `content`, `timestamp`, `confidence` (if set), `scope`, `topics`, `retracted` (boolean), `source`, and for syntheses `inputs` and `unresolved`. When the class has more than one member the JSON result SHALL include `class` listing the member particular ids.

#### Scenario: Lineage order
- **WHEN** claims `clm_A`, `clm_B` and synthesis `syn_C` (inputs A, B) exist about Project X and `recall "Project X"` is run
- **THEN** the result lists `clm_A`, `clm_B`, then `syn_C`; `syn_C` has `current: true` and neither claim has `unsynthesised: true`

#### Scenario: Later claim marked unsynthesised
- **WHEN** `clm_D` is asserted about Project X after `syn_C`
- **THEN** `recall "Project X"` marks `clm_D` with `unsynthesised: true` and `syn_C` remains `current`

#### Scenario: Recall across a merge
- **WHEN** particulars A and B are joined by a non-retracted merge and `recall A` is run
- **THEN** claims whose `subject` is B are included, each still carrying `subject: B`, and the JSON result has `class: [A, B]`

#### Scenario: Current by timestamp
- **WHEN** `syn_1` (timestamp 2026-03-01) and `syn_2` (timestamp 2026-01-01, minted later) both exist non-retracted about Project X
- **THEN** `syn_1` is `current`

#### Scenario: Retracted excluded by default
- **WHEN** `clm_A` is retracted and `recall "Project X"` is run
- **THEN** `clm_A` is absent; with `--include-retracted` it is present with `retracted: true`

#### Scenario: Empty recall
- **WHEN** `recall "Project X"` is run and no claims exist about it
- **THEN** the command exits 0 with an empty list

### Requirement: Recall by topic
`particulars recall --topic <t>` without a particular SHALL return all non-retracted claims and syntheses whose `context.topics` contains `t`, across all particulars, ordered by ascending id. When combined with a particular, results SHALL satisfy both filters.

#### Scenario: Topic across particulars
- **WHEN** claims about two different particulars both carry topic `architecture` and `recall --topic architecture` is run
- **THEN** both claims are returned

### Requirement: Scope filter
`--scope <s>` SHALL restrict results to objects whose `context.scope` equals `s`.

#### Scenario: Public only
- **WHEN** `recall "Project X" --scope public` is run
- **THEN** only objects with `context.scope: public` are returned

### Requirement: Lineage trace
`particulars lineage <id> [--depth <n>]` SHALL return the provenance tree rooted at the given claim or synthesis, recursively expanding each synthesis's `inputs` (with their `role` and `weight`) up to `depth` levels (default unlimited). Retracted ancestors SHALL be included and marked `retracted: true`; when a retracted node's block names a `superseded-by`, the node SHALL carry `superseded_by` with that id, which SHALL NOT be expanded as an input. In JSON mode the result SHALL be a nested object; in text mode an indented tree. An unknown id SHALL exit 3.

#### Scenario: Two-level tree
- **WHEN** `syn_D` cites `syn_C` (thesis) and `clm_E` (antithesis), and `syn_C` cites `clm_A` and `clm_B`, and `lineage syn_D` is run
- **THEN** the tree shows `syn_D` → `syn_C` (thesis) → `clm_A`, `clm_B`, and `syn_D` → `clm_E` (antithesis)

#### Scenario: Depth limited
- **WHEN** `lineage syn_D --depth 1` is run for the structure above
- **THEN** the tree shows `syn_D` with children `syn_C` and `clm_E` and no grandchildren

#### Scenario: Claim has no lineage
- **WHEN** `lineage clm_A` is run for a plain claim
- **THEN** the result is the claim alone with an empty `inputs` list

#### Scenario: Superseded node
- **WHEN** `clm_A` was retracted with `superseded-by: clm_Y` and `lineage syn_C` (which cites `clm_A`) is run
- **THEN** the `clm_A` node has `retracted: true` and `superseded_by: clm_Y`, and `clm_Y` does not appear as a child of `clm_A`

### Requirement: Topic listing
`particulars topics [<particular>] [--scope <s>] [--include-retracted]` SHALL list every topic carried by claims and syntheses in the workspace (or about the resolved particular), with the number of assertions carrying it and the number of distinct particulars among them, sorted by topic name. Retracted assertions SHALL be excluded unless `--include-retracted` is given.

#### Scenario: Topics across particulars
- **WHEN** a claim about Project X carries topics `architecture` and `db`, a claim about Project Y carries `architecture`, and `topics` is run
- **THEN** the result lists `architecture` (2 assertions, 2 particulars) and `db` (1 assertion, 1 particular)

#### Scenario: Retracted excluded
- **WHEN** the only claim carrying `db` is retracted and `topics` is run
- **THEN** `db` is absent; with `--include-retracted` it is present

#### Scenario: Per particular
- **WHEN** `topics "Project Y"` is run
- **THEN** only topics carried by assertions about Project Y are listed
