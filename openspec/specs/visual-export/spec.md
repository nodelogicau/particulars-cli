# Visual Export

## Purpose

Drawing a workspace so a reader can see its shape: the dialectic for one particular — claims and syntheses joined by typed input edges, with the current belief, unreconciled and stale assertions, retractions, and cross-particular inputs each rendered distinctly — and a map of every particular joined by merge records. Covers the two formats (Graphviz DOT and Mermaid), node identity and determinism, scope handling, and the flags that bound the lineage view.

## Requirements

### Requirement: Visual export formats
`particulars export --format dot|mermaid [--subject <particular>] [--depth <n>] [--include-retracted] [--scope <s>] [--out <path>]` SHALL emit the workspace as a graph description in the named format: `dot` as a Graphviz `digraph`, `mermaid` as a Mermaid `flowchart`. With `--out` it SHALL write to that file, otherwise to stdout. `--json` SHALL emit a summary object (`{format, view, nodes, edges, path?}`) rather than the diagram. The command SHALL NOT perform any network request and SHALL NOT require Graphviz or any renderer to be installed.

#### Scenario: DOT for a workspace
- **WHEN** `export --format dot` is run in a workspace with knowledge
- **THEN** stdout is a parseable Graphviz `digraph` and the exit code is 0

#### Scenario: Mermaid for a workspace
- **WHEN** `export --format mermaid` is run in the same workspace
- **THEN** stdout begins with a `flowchart` declaration and contains one node per exported object

#### Scenario: Summary instead of the diagram
- **WHEN** `export --format dot --json` is run
- **THEN** the result is a summary object carrying `format`, `view`, `nodes`, and `edges`, and no DOT text

#### Scenario: No renderer required
- **WHEN** the export runs on a machine with no Graphviz installation and no network
- **THEN** it succeeds

### Requirement: The lineage view
With `--subject <particular>`, the export SHALL render the dialectic for that particular's merge equivalence class: one node per claim and per synthesis, and one edge per synthesis input, directed from the input to the synthesis and labelled with the input's role. The subject SHALL be resolvable by id, URI, label, or alias, and an ambiguous subject SHALL exit 2 listing the candidates, as elsewhere. Cross-particular inputs SHALL be included as nodes, since they are part of the reasoning, and SHALL be marked as belonging to another particular.

#### Scenario: Roles are visible on the edges
- **WHEN** a synthesis cites one claim as `thesis` and another as `antithesis` and the lineage view is rendered
- **THEN** the two edges are labelled `thesis` and `antithesis` respectively

#### Scenario: Subject resolution
- **WHEN** `--subject` names a label that matches no particular
- **THEN** the command exits 3

#### Scenario: Merged particulars share one view
- **WHEN** two particulars are joined by a non-retracted merge record and either is named as the subject
- **THEN** the view contains the claims and syntheses of both

#### Scenario: Cross-particular input
- **WHEN** a synthesis in the view cites a claim whose subject is a different particular outside the class
- **THEN** that claim appears as a node marked as belonging to another particular

### Requirement: Conflict state is rendered, not merely present
The lineage view SHALL distinguish, by visual attribute rather than by text alone, the assertions the workspace treats differently: the `current` synthesis SHALL be emphasised, assertions in `unsynthesised` SHALL be marked as not yet reconciled, and syntheses in `stale` SHALL be marked as citing a retracted input. The attributes SHALL be legible in both formats without external styling.

#### Scenario: The current belief is distinguishable
- **WHEN** a particular has two non-retracted syntheses and the lineage view is rendered
- **THEN** only the one `conflict_detect` reports as `current` carries the emphasis attribute

#### Scenario: Unsynthesised claims are marked
- **WHEN** a claim is not reconciled into the current synthesis
- **THEN** its node carries the unsynthesised attribute

#### Scenario: Stale syntheses are marked
- **WHEN** a synthesis cites a retracted input, directly or transitively
- **THEN** its node carries the stale attribute

### Requirement: Retracted objects are excluded unless requested
Retracted claims, syntheses, and merge records SHALL be omitted from both views by default, matching `recall`. With `--include-retracted` they SHALL be rendered with a distinct attribute marking them retracted, and a `superseded-by` pointer SHALL be rendered as an edge to its successor, distinguishable from an input edge.

#### Scenario: Retracted by default
- **WHEN** a particular has one retracted claim and one live claim and the lineage view is rendered
- **THEN** only the live claim appears

#### Scenario: Retracted on request
- **WHEN** the same view is rendered with `--include-retracted`
- **THEN** both appear and the retracted one carries the retracted attribute

#### Scenario: Superseded-by is an edge, not an input
- **WHEN** a retracted claim names `superseded-by` and `--include-retracted` is given
- **THEN** an edge joins it to its successor, rendered distinctly from input edges

### Requirement: The workspace map
Without `--subject`, the export SHALL render one node per particular, sized or weighted by the number of non-retracted assertions about it, carrying its conflict priority, with non-retracted merge records rendered as edges joining the members of each equivalence class. A particular with no assertions SHALL still appear, since an orphan particular is a thing a reader needs to see.

#### Scenario: Particulars and merges
- **WHEN** a workspace holds three particulars, two of them joined by a merge record, and the map is rendered
- **THEN** three nodes appear and one edge joins the merged pair

#### Scenario: Orphan particular
- **WHEN** a particular has no claims
- **THEN** it appears in the map

#### Scenario: Priority is carried
- **WHEN** a particular has unsynthesised claims
- **THEN** its node carries the conflict priority reported by `conflict_detect`

### Requirement: Node identity and determinism
Generated node identifiers SHALL be unique and SHALL NOT be produced by truncating object ids, which share a long common prefix under UUIDv7. Each node SHALL carry enough of its object id to be found in the workspace by a reader of the diagram. Two runs against an unchanged workspace SHALL produce byte-identical output, with nodes and edges emitted in a stable order that does not depend on map iteration.

#### Scenario: Distinct nodes for ids sharing a prefix
- **WHEN** the view contains several objects minted in the same millisecond
- **THEN** each has a distinct node identifier and no node is silently merged with another

#### Scenario: Deterministic output
- **WHEN** the export is run twice against an unchanged workspace
- **THEN** both runs produce byte-identical output

#### Scenario: Labels are safe for the format
- **WHEN** an object's content contains quotes, newlines, or characters with meaning in DOT or Mermaid
- **THEN** the emitted diagram is still parseable by that format's parser

### Requirement: Scope filters but does not silently withhold
The visual formats SHALL include `personal` knowledge by default, because emitting a local file is not a transfer to a third party and a diagram that omitted part of the graph would misrepresent it. `--scope <s>` SHALL restrict the export to that scope, and unlike `--format graph` the value `personal` SHALL be accepted. Documentation SHALL state that a diagram discloses whatever it contains, so publishing one — in a pull request comment, for instance — carries the same judgement as publishing the objects themselves.

#### Scenario: Personal knowledge is drawn
- **WHEN** a particular's claims are all `personal` and `export --format mermaid --subject <it>` is run
- **THEN** those claims appear as nodes

#### Scenario: Scope narrows the diagram
- **WHEN** `--scope organisation` is given
- **THEN** only assertions whose scope is `organisation` or wider appear

### Requirement: Depth bounds the lineage view
`--depth <n>` SHALL limit the lineage view to `n` levels of inputs from the subject's current belief, as `lineage --depth` does. Omitted, the view SHALL cover the whole closure for the equivalence class.

#### Scenario: Bounded traversal
- **WHEN** a chain of syntheses four levels deep is rendered with `--depth 2`
- **THEN** only the first two levels of inputs appear
