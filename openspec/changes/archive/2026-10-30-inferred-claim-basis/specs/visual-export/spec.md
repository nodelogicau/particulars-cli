## MODIFIED Requirements

### Requirement: The lineage view
With `--subject <particular>`, the export SHALL render the dialectic for that particular's merge equivalence class: one node per claim and per synthesis, one edge per synthesis input, directed from the input to the synthesis and labelled with the input's role, and one edge per basis member of an inferred claim, directed from the member to the claim, labelled `basis`, and drawn in a line style distinct from input edges (dotted in DOT, `-.->` in Mermaid, where the `basis` label tells it from a superseded-by edge). The subject SHALL be resolvable by id, URI, label, or alias, and an ambiguous subject SHALL exit 2 listing the candidates, as elsewhere. Cross-particular inputs and basis members SHALL be included as nodes, since they are part of the reasoning, and SHALL be marked as belonging to another particular.

#### Scenario: Roles are visible on the edges
- **WHEN** a synthesis cites one claim as `thesis` and another as `antithesis` and the lineage view is rendered
- **THEN** the two edges are labelled `thesis` and `antithesis` respectively

#### Scenario: Basis edges are distinguishable
- **WHEN** an inferred claim in the view lists two claims in its basis
- **THEN** two edges run from those claims to it, each labelled `basis` and drawn in the basis line style

#### Scenario: Subject resolution
- **WHEN** `--subject` names a label that matches no particular
- **THEN** the command exits 3

#### Scenario: Merged particulars share one view
- **WHEN** two particulars are joined by a non-retracted merge record and either is named as the subject
- **THEN** the view contains the claims and syntheses of both

#### Scenario: Cross-particular input
- **WHEN** a synthesis in the view cites a claim whose subject is a different particular outside the class
- **THEN** that claim appears as a node marked as belonging to another particular

#### Scenario: Cross-particular basis member
- **WHEN** an inferred claim in the view lists in its basis a claim about a particular outside the class
- **THEN** that claim appears as a node marked as belonging to another particular

### Requirement: Conflict state is rendered, not merely present
The lineage view SHALL distinguish, by visual attribute rather than by text alone, the assertions the workspace treats differently: the `current` synthesis SHALL be emphasised, assertions in `unsynthesised` SHALL be marked as not yet reconciled, and claims and syntheses in `stale` SHALL be marked as citing a retracted object. The attributes SHALL be legible in both formats without external styling.

#### Scenario: The current belief is distinguishable
- **WHEN** a particular has two non-retracted syntheses and the lineage view is rendered
- **THEN** only the one `conflict_detect` reports as `current` carries the emphasis attribute

#### Scenario: Unsynthesised claims are marked
- **WHEN** a claim is not reconciled into the current synthesis
- **THEN** its node carries the unsynthesised attribute

#### Scenario: Stale syntheses are marked
- **WHEN** a synthesis cites a retracted input, directly or transitively
- **THEN** its node carries the stale attribute

#### Scenario: Stale inferred claims are marked
- **WHEN** an inferred claim's basis lists a retracted claim
- **THEN** its node carries the stale attribute

### Requirement: Depth bounds the lineage view
`--depth <n>` SHALL limit the lineage view to `n` levels of inputs and basis members from the subject's current belief, as `lineage --depth` does, with each kind of edge counting as one level. Omitted, the view SHALL cover the whole closure for the equivalence class.

#### Scenario: Bounded traversal
- **WHEN** a chain of syntheses four levels deep is rendered with `--depth 2`
- **THEN** only the first two levels of inputs appear

#### Scenario: A basis counts as a level
- **WHEN** the current synthesis cites an inferred claim whose basis lists one claim about a particular outside the class, and the view is rendered with `--depth 1`
- **THEN** the inferred claim appears and its basis member does not
