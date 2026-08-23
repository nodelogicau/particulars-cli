## Context

The workspace is already a graph; nothing in the CLI draws it. `internal/query` computes everything a drawing needs — `Closure` walks a particular's reasoning, `Analyse`/`AnalyseSet` return `current`/`unsynthesised`/`stale` per equivalence class, and `store.Graph` resolves merges into classes. `internal/graph` (the Microsoft Graph export) is the existing precedent for a read-only projection of the workspace into somebody else's format, and it is the shape to copy: pure functions over a loaded `store.Graph`, no I/O of its own, deterministic output, tested by comparing bytes.

The constraint that shapes this design is that two formats must render the same semantics. DOT and Mermaid have different syntax, different escaping rules, and different styling vocabularies, but the *meaning* — this synthesis is current, that claim is unreconciled — must survive in both. If each renderer decides for itself what "current" looks like, the two drift and the second one rots.

A prototype run against the real knowledge workspace during design produced a legible dialectic (two theses and an antithesis resolving into a synthesis) and, on the first attempt, silently collapsed every node into one because it derived node ids by truncating object ids. UUIDv7 ids share a 12-character prefix. That bug is why node identity is a normative requirement rather than an implementation detail.

## Goals / Non-Goals

**Goals:**

- One node/edge model, two renderers, so DOT and Mermaid cannot disagree about what the workspace means.
- Conflict state visible as a visual attribute, not a text annotation a reader has to decode.
- Deterministic bytes, so a diagram can be committed and diffed.
- No new module dependencies, no renderer required at runtime.

**Non-Goals:**

- Rendering images. The command emits text; `dot -Tsvg` and GitHub's Mermaid renderer do the drawing. Shelling out to Graphviz would put a system dependency in the path of a verb that otherwise has none.
- Interactive or HTML output, clickable navigation, layout tuning beyond what the formats express natively.
- A third format. Two is enough to prove the model is renderer-independent; a third can be added against the same interface if a need appears.

## Decisions

### An intermediate model, not two independent renderers

`internal/viz` builds a `Model{Nodes, Edges}` from the query layer, and each renderer consumes it. Node attributes are *semantic* (`Kind: claim|synthesis|particular`, `State: current|unsynthesised|stale|retracted|plain`, `Foreign bool`), never presentational — no colours or line styles in the model. Each renderer maps state to the vocabulary of its format.

*Alternative considered:* a single renderer with format-specific string templates. Rejected — escaping rules differ enough (DOT wants `\"` inside quoted labels and supports HTML-like labels; Mermaid breaks on unescaped `"` and needs `<br>` for line breaks) that a shared template would end up as a pile of conditionals, and that is exactly where the two formats would drift apart.

### Node ids are sequential, assigned in traversal order

`n1`, `n2`, … assigned as nodes are appended, with the object id carried in the label as `clm…025dde` (prefix plus last six characters). Sequential ids are guaranteed unique, valid in both formats without escaping, and stable given a stable traversal order.

*Alternatives considered:* the full object id sanitised — valid, but 40 characters of visual noise per node in the source, and Mermaid ids cannot contain `-` without quoting; a hash of the id — stable and short, but meaningless to a reader diffing the file, and collisions would be silent. Truncation is excluded by the requirement, having already failed once in the prototype.

### Determinism comes from sorting at the boundary, not from Go map order

Nodes are collected into a slice sorted by object id before ids are assigned; edges are sorted by `(from, to, role)`. Go's map iteration is deliberately randomised, so any traversal that touches a map must sort before emitting. This is the same discipline the Graph export already follows, and the "run twice, compare bytes" test is the guard.

### The two views share the model, not the query

The lineage view calls `query.Closure` + `Analyse` for one equivalence class; the map calls `AnalyseSet` across the workspace. They produce different node kinds from different queries, but both emit a `Model`, so both renderers work on both views without knowing which is which.

### Scope defaults to including everything

The Graph export refuses `personal` because it transfers knowledge to a third party's index, where a mistake is unrecoverable. Emitting a local file is not that, and a diagram missing half its nodes would be a misleading picture of the reasoning — the failure mode here is a *wrong drawing*, which is worse than a drawing the user chooses not to share. `--scope` filters for the cases where the output is going somewhere public, and the documentation carries the disclosure note.

*This is a deliberate asymmetry between two subcommands of the same verb*, and it will look inconsistent to someone reading the flag table. The help text for `--scope` states the difference explicitly rather than leaving it to be discovered.

### Cross-particular inputs are drawn and marked

A synthesis may cite a claim about another particular. Omitting it would break the visible chain of reasoning at exactly the point a reader is asking "where did that come from?", so it is drawn, marked `Foreign`, and rendered with a distinct outline. It does not pull that particular's other claims into the view.

## Risks / Trade-offs

- **Large workspaces produce unreadable diagrams.** → `--depth` bounds the lineage view and the map is one node per particular rather than per claim, which keeps it proportional to the number of things rather than the amount known about them. Neither is a real fix for a workspace of thousands; if that arrives, filtering by topic is the next lever, and the model supports it without renderer changes.
- **Mermaid's parser is stricter and less forgiving than Graphviz's, and its accepted syntax has changed between versions.** → Escape aggressively (quote every label, strip newlines to `<br>`, drop characters with no safe encoding), and test against a fixture containing quotes, angle brackets, and Unicode. Emit the most conservative subset of Mermaid that expresses the semantics, not the newest.
- **A diagram pasted into a public pull request leaks whatever it contains** — the same derivation-crossing-a-boundary problem `scope_wider_than_inputs` warns about, one level up. → Default is a local file; the documentation and the `--scope` help say plainly what publishing one means. Not enforced in code, because the command cannot know where its stdout is going.
- **The optional CI step could post a large diagram into a pull request comment.** → It renders only the particulars the pull request touches, and it is opt-in in the example workflow rather than on by default.
- **Two renderers is twice the surface to keep correct.** → The shared model keeps the semantics in one place; renderer tests assert that both formats mark the same node as current, so a drift shows up as a failing test rather than a subtly wrong picture.

## Open Questions

- Should the map's node weight be claim count, or unreconciled count? Claim count answers "where is the knowledge", unreconciled answers "where is the work". Starting with claim count and carrying priority as a separate attribute, so both are legible; revisit once there is a workspace big enough to judge.
- Whether `--topic` filtering belongs in this change or the next. The model supports it trivially; the flag is left out for now to keep the surface small.
