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

### Requirement: Recall entries carry the warrant
Each recall entry for a claim SHALL include `evidential` when the file declares one, and SHALL omit it when the file predates the field — so a consumer sees exactly the three-values-or-absent that the file says, and `undeclared` remains a validator's report rather than a value that travels.

#### Scenario: Declared and undeclared side by side
- **WHEN** `recall` returns one claim carrying `evidential: held` and one written before the field existed
- **THEN** the first entry carries `"evidential": "held"` and the second carries no `evidential` key

### Requirement: Recall by author
`particulars recall [--author <id|uri|label|alias>]` SHALL return the non-retracted claims and syntheses asserted by or reported from the resolved particular's merge equivalence class — an object is asserted by the particular its `source.author` resolves to and reported from the particular its `source.document.author` resolves to, per `source-attribution`. Each returned entry SHALL carry `relations`, a list containing `asserted`, `reported`, or both, never empty. `--author` SHALL be a sufficient selector on its own and SHALL be combinable with a subject, `--topic`, `--scope`, and `--include-retracted`, results satisfying every filter given. An author query resolving to no particular SHALL exit 3; one matching several SHALL exit 2 listing the candidate ids. Entries returned without `--author` SHALL NOT carry `relations`.

#### Scenario: Everything Jane said
- **WHEN** `recall --author jane` is run where one claim's `source.author` and another claim's `source.document.author` resolve to Jane's class
- **THEN** both are returned, the first with `relations: [asserted]`, the second with `relations: [reported]`

#### Scenario: Both relations on one object
- **WHEN** a claim's `source.author` and `source.document.author` both resolve to Ben's class and `recall --author ben` is run
- **THEN** that entry carries `relations: [asserted, reported]`

#### Scenario: Author with a subject
- **WHEN** `recall "Project X" --author jane` is run
- **THEN** only objects whose subject is in Project X's class and which are asserted by or reported from Jane are returned

#### Scenario: Attribution across a merge
- **WHEN** a person's URN and ORCID are merged, a claim carries the URN as `source.author`, and `recall --author <ORCID>` is run
- **THEN** the claim is returned, its file unchanged

### Requirement: Recall by query
`particulars recall --query <text> [--author <who>] [--topic <t>]... [--scope <s>] [--include-retracted] [--limit <n>]` SHALL implement DKF `recall-query`'s lexical baseline:
- **Particulars resolved.** It SHALL resolve the particulars whose label, an alias, or the `content` of a claim or synthesis about them matches the query, ignoring every filter.
- **Response per particular.** It SHALL then return, for each resolved merge class, exactly what `recall <lowest member id>` would return with the same `--author`, `--topic`, `--scope`, and `--include-retracted`. A class with nothing left after filtering SHALL be omitted.
- **Terms.** A term SHALL be a maximal run of characters without the Unicode `White_Space` property. Leading and trailing characters of Unicode general category P SHALL be removed from the whole query and from each term, as written and before case folding, and interior characters SHALL be kept. A term empty after removal SHALL be ignored.
- **Matching.** Matching SHALL be under full Unicode case folding. A text SHALL match when the trimmed whole query or any one trimmed term is a substring of it.
- **Empty query.** A query empty after removal SHALL produce an empty result and exit 0.

`--query` SHALL be a usage error together with a positional particular.

The JSON result SHALL carry:
- `query`;
- `particulars`, a list of `{particular, label, resolution, entries}`, plus `class` when the class has more than one member, where `particular` is the lowest id in the class;
- `count`, the number of particulars returned;
- `omitted`, the number of non-empty resolved classes not returned because of the limit;
- `author` when `--author` is given.

Entries SHALL be as in recall by particular. An entry whose `content` matches SHALL carry `matched: true`.

`resolution` SHALL be:
- `label` when a member's label, case-folded, occurs in the case-folded query with every query character outside that occurrence of category P;
- otherwise `alias`, on the same test against aliases;
- otherwise `lexical`.

Particulars SHALL be ordered by `resolution` in that order. Within each value they SHALL be ordered by:
1. whether the trimmed whole query matches a member's label, an alias, or an entry's `content`, matches first;
2. the number of distinct case-folded trimmed terms that match any of those, more first;
3. ascending `particular`.

`--limit` SHALL count particulars, keeping the first in this order with their entries whole. It SHALL default to 10 when omitted or `0`, and SHALL be a usage error when negative.

#### Scenario: A query matches a superseded claim
- **WHEN** `clm_1` about Project T contains "auth timeout", `syn_1` about Project T synthesises `clm_1` without the phrase, and `recall --query "auth timeout" --json` is run
- **THEN** Project T's entry lists `clm_1` with `matched: true` and `syn_1` with `current: true`

#### Scenario: A question mark does not hide a term
- **WHEN** a claim reads "The session timeout is 30 minutes." and `recall --query "timeout?"` is run
- **THEN** its subject is returned with `resolution: lexical` and the claim is `matched`

#### Scenario: An identifier at the end of a sentence
- **WHEN** a claim contains `PROJ-1234` and `recall --query "the ticket was PROJ-1234."` is run
- **THEN** the claim's subject is returned

#### Scenario: Interior punctuation is kept
- **WHEN** a claim's only matching text is "v1.2" and `recall --query "v1.2.3?"` is run
- **THEN** its subject is not returned

#### Scenario: Full case folding
- **WHEN** a claim contains "Straße" and `recall --query STRASSE` is run
- **THEN** the claim's subject is returned

#### Scenario: A name asked about is exact
- **WHEN** particulars are labelled `Project X` and `C#`, and the queries are `Project X?`, `"Project X"`, and `C#?`
- **THEN** each named particular has `resolution: label`

#### Scenario: A prefix is not exact
- **WHEN** a particular is labelled `C#`, a claim about it contains "C", and the query is `C`
- **THEN** its `resolution` is `lexical`

#### Scenario: Exact alias
- **WHEN** a particular has alias `DKF` and the query is `dkf`
- **THEN** its `resolution` is `alias`

#### Scenario: Punctuation alone
- **WHEN** the query is `?!`, even with a particular labelled `?`
- **THEN** `particulars` is empty and the command exits 0

#### Scenario: Retracted match
- **WHEN** the only match for the query is a retracted claim about Project A, which has other claims, and `--include-retracted` is not given
- **THEN** Project A is returned, but the retracted claim is not among its entries and does not count toward its terms

#### Scenario: Nothing in scope
- **WHEN** the query resolves Project P, all of whose objects are `personal`, and `--scope public` is given
- **THEN** Project P is not returned and is not counted in `omitted`

#### Scenario: A merge class appears once
- **WHEN** Project A and Project B are merged and the query matches a claim about each
- **THEN** one particular is returned, under the lower id, with `class` listing both and entries about both

#### Scenario: More terms matched ranks first
- **WHEN** the query is `auth timeout`, Project A's entries contain "auth" and "timeout" in separate claims, Project B's contain only "auth", and `--limit 1` is given
- **THEN** Project A is returned and `omitted` is 1

#### Scenario: The whole query ranks first
- **WHEN** the query is `auth timeout`, Project A has a claim containing "auth timeout", and Project B has "auth" and "timeout" in separate claims
- **THEN** Project A is ordered before Project B

#### Scenario: Exact before lexical under a limit
- **WHEN** one particular is labelled `Billing` and has a claim, twelve others have claims containing "billing", and `recall --query billing` is run
- **THEN** ten particulars are returned, the first with `resolution: label`, and `omitted` is 3

#### Scenario: Query and particular together
- **WHEN** `recall "Project X" --query timeout` is run
- **THEN** the command exits with the usage code

#### Scenario: Filters combine with a query
- **WHEN** `recall --query timeout --topic auth` is run
- **THEN** every returned entry carries topic `auth`

