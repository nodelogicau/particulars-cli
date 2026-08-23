## Why

A DKF workspace is a graph — claims and syntheses joined by typed input edges, particulars joined by merge records — but every way of reading it today is linear. `recall` returns a list, `lineage` returns a JSON tree, `conflicts` returns three sets of ids. A reviewer approving a pull request has to reconstruct the shape in their head from ids that share a 12-character prefix.

The shape is exactly what carries the meaning in this format. Which claim was the antithesis, which synthesis is current, what a retraction orphaned: all of it is structure, and structure is what a diagram shows and a list does not. The data needed is already computed and already indexed — `index.yaml` carries `inputs`, `query.Analyse` computes `current`/`unsynthesised`/`stale`, merge records already define equivalence classes. This change is assembly, not new machinery.

## What Changes

- `particulars export --format dot` emits Graphviz DOT; `--format mermaid` emits Mermaid `flowchart`. Both are new values of the existing `--format` flag, alongside `graph` (Microsoft Graph), reusing `--out`, `--scope`, and `--json`.
- **With `--subject <particular>`**: the lineage view. Claims and syntheses are nodes, input edges are labelled with their role (`thesis`, `antithesis`, `thesis:qualifying`), and the styling carries the semantics — the current belief emphasised, retracted objects dashed and muted, `unsynthesised` and `stale` marked. Computed across the merge equivalence class, like every other query verb.
- **Without `--subject`**: the workspace map. Particulars are nodes sized by claim count and coloured by conflict priority, joined by merge edges into their equivalence classes — the "where is the unreconciled knowledge" view.
- `--depth <n>` bounds the lineage view; `--include-retracted` opts retracted objects into it (excluded by default, as in `recall`).
- Output is deterministic: stable node ordering and stable generated node ids, so a diagram committed to a repository diffs cleanly.
- Mermaid is the format that renders without tooling — in a GitHub pull request comment, in a markdown file, in a chat client. `docs/examples/dkf-check.yml` gains an optional step posting a diagram for the particulars a pull request touches, so a reviewer sees the shape of what changed.
- **Personal knowledge is included by default**, unlike `--format graph`. Writing a local file is not a transfer to a third party, and a visualisation that silently omits half the graph misrepresents it. `--scope` filters, and the documentation states plainly that pasting a diagram into a public pull request discloses whatever the diagram contains.

## Capabilities

### New Capabilities
- `visual-export`: emitting a workspace as Graphviz DOT and Mermaid — the two views, what each node and edge represents, how conflict state and retraction are rendered, node identity and determinism, scope handling, and the flags that bound the lineage view.

### Modified Capabilities

None. Every requirement in `graph-export` is scoped to `--format graph` and stays true verbatim; `cli-interface` specifies the cross-cutting contract (exit codes, `--json`, workspace selection) and does not enumerate verbs or their flags. The new format values, and the flags that only apply to them, belong entirely to `visual-export`.

## Impact

- **New**: `internal/viz/` (node and edge model derived from `internal/query`, one renderer per format). No new module dependencies — both formats are text.
- **Modified**: `internal/cli/cmd_export.go` (format dispatch and the new flags), `README.md` (verb table), `docs/examples/dkf-check.yml` (optional diagram step).
- **New**: `docs/visualise.md` — the two views, rendering DOT to SVG, embedding Mermaid in a pull request, and the disclosure note about scope.
- **Reused unchanged**: `query.Closure`, `query.Analyse`, `query.AnalyseSet`, `store.Graph` merge classes. No format or on-disk change; nothing about how objects are written is affected.
- **Not in scope**: rendering images (no Graphviz dependency is introduced — the command emits DOT and the user renders it), interactive or HTML output, and any layout tuning beyond what the two formats express natively.
