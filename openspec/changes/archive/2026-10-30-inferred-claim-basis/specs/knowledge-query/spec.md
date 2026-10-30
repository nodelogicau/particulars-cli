## MODIFIED Requirements

### Requirement: Lineage trace
`particulars lineage <id> [--depth <n>]` SHALL return the provenance tree rooted at the given claim or synthesis, recursively expanding each synthesis's `inputs` (with their `role` and `weight`) and each claim's `basis` up to `depth` levels (default unlimited), with a basis edge and an input edge each counting as one level. Basis children SHALL be listed under `basis`, separately from `inputs`, and SHALL carry no `role` or `weight`; in text mode they SHALL be labelled `basis`. Every node SHALL carry both lists, empty when there is nothing to list. Traversal SHALL expand each object at most once: a later occurrence of an object already expanded elsewhere in the tree SHALL appear as a node carrying its `id` and `type` and `repeated: true`, and SHALL NOT be expanded, while an occurrence of an object still on the current path SHALL carry `cycle: true`. Retracted ancestors SHALL be included and marked `retracted: true`; when a retracted node's block names a `superseded-by`, the node SHALL carry `superseded_by` with that id, which SHALL NOT be expanded as an input. In JSON mode the result SHALL be a nested object; in text mode an indented tree. An unknown id SHALL exit 3. The MCP `lineage_trace` tool SHALL return the same tree.

#### Scenario: Two-level tree
- **WHEN** `syn_D` cites `syn_C` (thesis) and `clm_E` (antithesis), and `syn_C` cites `clm_A` and `clm_B`, and `lineage syn_D` is run
- **THEN** the tree shows `syn_D` → `syn_C` (thesis) → `clm_A`, `clm_B`, and `syn_D` → `clm_E` (antithesis)

#### Scenario: Depth limited
- **WHEN** `lineage syn_D --depth 1` is run for the structure above
- **THEN** the tree shows `syn_D` with children `syn_C` and `clm_E` and no grandchildren

#### Scenario: Claim has no lineage
- **WHEN** `lineage clm_A` is run for a claim with no basis
- **THEN** the result is the claim alone with empty `inputs` and `basis` lists

#### Scenario: Walking from a rule to its instances
- **WHEN** `lineage clm_rule` is run on a claim whose basis lists `clm_a`, `clm_b` and `clm_c`
- **THEN** the tree shows the three claims under `basis`, and its `inputs` list is empty

#### Scenario: Through a synthesis to a rule's instances
- **WHEN** `syn_1` cites `clm_rule` as `thesis`, `clm_rule`'s basis lists `clm_a`, and `lineage syn_1` is run
- **THEN** the tree shows `syn_1` → `clm_rule` (thesis) → `clm_a` (basis)

#### Scenario: An object reached twice
- **WHEN** `syn_D` cites `syn_C` and `clm_A`, and `syn_C` also cites `clm_A`
- **THEN** `clm_A` is expanded once, and its other occurrence carries `repeated: true` with no children

#### Scenario: Superseded node
- **WHEN** `clm_A` was retracted with `superseded-by: clm_Y` and `lineage syn_C` (which cites `clm_A`) is run
- **THEN** the `clm_A` node has `retracted: true` and `superseded_by: clm_Y`, and `clm_Y` does not appear as a child of `clm_A`

### Requirement: Recall entries carry the warrant
Each recall entry for a claim SHALL include `evidential` when the file declares one, and SHALL omit it when the file predates the field — so a consumer sees exactly the three-values-or-absent that the file says, and `undeclared` remains a validator's report rather than a value that travels. An entry for a claim that carries `basis` SHALL include `basis` as written, and SHALL omit it otherwise. Lineage order SHALL treat basis members as predecessors of the claim citing them, as inputs are of a synthesis, so a recalled rule follows the instances it rests on.

#### Scenario: Declared and undeclared side by side
- **WHEN** `recall` returns one claim carrying `evidential: held` and one written before the field existed
- **THEN** the first entry carries `"evidential": "held"` and the second carries no `evidential` key

#### Scenario: A rule and its instances
- **WHEN** `recall` returns `clm_rule` with `basis: [clm_a]` and also returns `clm_a`
- **THEN** the `clm_rule` entry carries `"basis": ["clm_a"]` and comes after the `clm_a` entry
