## 1. The model

- [x] 1.1 Create `internal/viz/model.go`: `Model{Nodes []Node, Edges []Edge}`, `Node{ID, ObjectID, Kind, Label, State, Foreign, Weight, Priority}`, `Edge{From, To, Role, Kind}`, with `Kind`/`State` as named constants. Attributes are semantic only — no colours, shapes, or line styles in the model.
- [x] 1.2 Assign node ids sequentially (`n1`, `n2`, …) after sorting nodes by object id, and carry `prefix…last6` of the object id in every label. Never derive an id by truncating an object id.
- [x] 1.3 Sort edges by `(from, to, role)` before emitting, so nothing depends on Go map iteration order.
- [x] 1.4 Unit-test the model: distinct ids for objects minted in the same millisecond, stable ordering across repeated builds from the same workspace.

## 2. The lineage view

- [x] 2.1 `internal/viz/lineage.go`: build a `Model` for one particular's merge equivalence class from `query.Closure` and `query.Analyse` — a node per claim and synthesis, an edge per input directed input→synthesis carrying its role.
- [x] 2.2 Map conflict state onto `Node.State`: the `current` synthesis, members of `unsynthesised`, members of `stale` (transitive, as `Analyse` already computes).
- [x] 2.3 Include cross-particular inputs as nodes marked `Foreign`, without pulling in the rest of their particular's claims.
- [x] 2.4 Exclude retracted objects by default; with `include-retracted`, add them in state `retracted` and emit `superseded-by` as an edge of a distinct kind.
- [x] 2.5 Honour a depth bound measured from the current belief, matching `lineage --depth`.
- [x] 2.6 Tests: roles land on the right edges; merged particulars yield one combined view from either member; only the analysed `current` carries that state; depth truncates.

## 3. The workspace map

- [x] 3.1 `internal/viz/map.go`: a node per particular from `query.AnalyseSet`, weighted by non-retracted assertion count and carrying conflict priority; non-retracted merge records as edges joining class members.
- [x] 3.2 Include particulars with no claims, so an orphan is visible.
- [x] 3.3 Tests: three particulars with one merge give three nodes and one merge edge; an orphan appears; priority is carried through.

## 4. Renderers

- [x] 4.1 `internal/viz/dot.go`: emit a Graphviz `digraph`, mapping `State` to shape/style/penwidth and `Kind` to node shape (claim rectangular, synthesis rounded, particular distinct). Escape labels for DOT.
- [x] 4.2 `internal/viz/mermaid.go`: emit a Mermaid `flowchart` using `classDef` for each state. Escape labels for Mermaid — quote every label, convert newlines to `<br>`, drop what cannot be encoded safely. Emit a conservative syntax subset rather than the newest.
- [x] 4.3 Table-test both renderers against a fixture whose content contains quotes, angle brackets, backticks, pipes, and Unicode; assert the output parses (DOT via a syntax check, Mermaid via its documented grammar constraints) and that no label breaks out of its delimiter.
- [x] 4.4 Cross-renderer test: for one model, assert both formats mark the same node as current, the same nodes as unsynthesised, and the same nodes as retracted — so the two cannot drift apart.
- [x] 4.5 Byte-identical output on repeated runs, for both formats and both views.

## 5. CLI surface

- [x] 5.1 Extend `internal/cli/cmd_export.go`: accept `dot` and `mermaid` as `--format` values alongside `graph`, dispatching to `internal/viz`. Keep the existing `graph` path untouched.
- [x] 5.2 Add `--subject`, `--depth`, and `--include-retracted`, rejecting them with a usage error when the format is `graph`, so the flags cannot appear to do something they do not.
- [x] 5.3 Accept `--scope personal` for the visual formats (unlike `graph`), and state the difference in the `--scope` help text rather than leaving it to be discovered.
- [x] 5.4 Resolve `--subject` by id, URI, label, or alias with the existing helper: exit 3 when nothing matches, exit 2 listing candidates when ambiguous.
- [x] 5.5 `--json` emits `{format, view, nodes, edges, path?}`; `--out` writes to a file. Update the command's `Long` help.
- [x] 5.6 In-process CLI tests covering: both formats, both views, `--out`, `--json`, subject resolution failures, flag rejection against `--format graph`, and `--scope personal` being accepted here and still refused for `graph`.

## 6. Documentation and CI

- [x] 6.1 Write `docs/visualise.md`: the two views with a worked example of each, rendering DOT to SVG, embedding Mermaid in a pull request or markdown file, and the disclosure note — a diagram discloses whatever it contains, so publishing one carries the same judgement as publishing the objects.
- [x] 6.2 Add the `export` visual formats to the README verb table and point at `docs/visualise.md`.
- [x] 6.3 Add an **opt-in, commented-out** step to `docs/examples/dkf-check.yml` that renders a Mermaid diagram for the particulars a pull request touches and folds it into the existing comment, reusing the `<!-- dkf-check -->` marker so it updates in place rather than accumulating.
- [x] 6.4 CHANGELOG entry under Unreleased.

## 7. Verification

- [x] 7.1 `go build ./... && go vet ./... && go test ./...` clean.
- [x] 7.2 Run both views of both formats against the real knowledge workspace; render the DOT to SVG and check the Mermaid renders on GitHub, confirming the current belief, an unsynthesised claim, and a retracted object are each visually distinguishable in practice and not merely attributed in the source.
