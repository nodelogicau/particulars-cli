## Context

The DKF spec's `recall-query` capability is now settled upstream, and it governs here. It is in nodelogicau/particulars `openspec/specs/recall-query/spec.md`, with changes `2026-10-16-recall-vector-cache` and `2026-10-17-recall-query-terms` archived.

- **Two stages.** Resolve the query to particulars. Then return, for each one, exactly what recall by particular returns under the same filters.
- **The baseline:**
  - A term is a maximal run of non-`White_Space` characters.
  - Category P is trimmed from the edges of the whole query and of each term, as written and before folding.
  - Terms are combined with OR.
  - Matching is by substring after case folding.
  - A query that is empty after trimming resolves nothing.
- **`resolution`:** `label` or `alias` when the folded name occurs in the folded query with only category P outside that occurrence, `lexical` otherwise, `semantic` for semantic-only matches.
- **A fixed order** within `label`, `alias` and `lexical`: whole-query match first, then distinct terms matched (more first), then lowest class id. Both are counted against the label, the aliases, and the objects in the particular's response after filtering. `limit` keeps the head of this order.

The spec leaves to implementations the response shape, how `limit` interacts with lineage, a default limit, how merge classes are presented, and the CLI surface. The #12 review raised each of these.

Current code:
- `query.Recall` (`internal/query/recall.go`) filters a candidate set and orders it by lineage. With `Subject` set it covers the whole merge class.
- `query.Resolve` (`internal/query/resolve.go`) is exact: `strings.EqualFold` on the label or an alias.
- `knowledgeRecall` (`internal/mcp/tools.go`) treats `query` as `particular_id`.
- The CLI has no query surface.
- The scale benchmarks put in-process recall at ~6 ms against ~2 s of load at 20,000 claims.

## Goals / Non-Goals

**Goals:**
- Conform to `recall-query` for the baseline, `resolution`, the order, and the matched marker.
- One function serves both the CLI and MCP, so `recall --query --json` equals the MCP structured result.
- A response bounded by default, so a question full of common words doesn't return the workspace.
- A seam where #13's semantic stage can add candidates.

**Non-Goals:**
- The semantic stage, embeddings, or any cache (#13).
- Stemming, stopwords, splitting on interior punctuation, or relevance scoring beyond the spec's order.
- Changing `particular_resolve`, `recall <particular>`, or how recall by topic or author behaves without `query`.
- Resolving ids or URIs through `query`. Those go in `particular_id`.

## Decisions

### D1. `query.RecallQuery` composes `Recall`, it doesn't re-implement it
`RecallQuery(g, text, opts) QueryResult` runs stage one, then calls `Recall` once per resolved class with `Subject` set to the class root, `Limit: 0`, and the caller's other filters. Stage two is therefore recall by particular by construction, which is what the first requirement asks for: lineage order, `current`, `unsynthesised`, `relations` under `author`, and effective scope.

*Alternative:* one filtered pass over all assertions, grouped afterwards. It is faster, but it re-derives `current` and lineage per class and would drift from `Recall`.

### D2. Term handling is one small, pure unit
`parseQuery(text) (whole string, terms []string)`:
- Split on `unicode.Is(unicode.White_Space, r)`.
- Trim each term and the whole query with `strings.TrimFunc(…, isEdge)`.
- Drop empty terms.
- Fold with `cases.Fold()`.
- Dedupe terms after folding, because the order counts *distinct* terms after case folding.

`isEdge(r) = unicode.IsPunct(r)`, which is category P. It is the only place edge characters are defined, so adopting nodelogicau/particulars#37 (also treat `White_Space` as an edge) is a one-line change plus scenarios.

Folding: `strings.EqualFold` and `strings.ToLower` are simple folding. The spec says Unicode case folding, and full folding is what makes `STRASSE` match `straße`. `golang.org/x/text/cases` is already in the module graph. A `cases.Caser` is stateful, so one is created per call and never shared.

### D3. Exact match is an occurrence test, not trim-and-compare
For each member's label and aliases: fold the name, then find every occurrence of it in the folded query, overlapping ones included. The name is exact if, for some occurrence, every rune before and after it satisfies `isEdge`. Empty names are skipped.

The check runs on the *untrimmed* folded query, because the rule is about every query character outside the occurrence. That is what makes `C#?` exact for `C#` and `C` not exact for `C#`. A trim-and-compare would get the second case wrong. The empty-query check runs first, so a label of `?` doesn't make `?!` resolve.

### D4. Stage one scans everything, ignoring filters
Stage one scans every particular (labels and aliases) and every claim and synthesis (`content`), including retracted and out-of-scope ones, because "filters never run in stage one". Each hit is mapped to its class root: the lowest id in `g.ClassOf`. Each class takes the best `resolution` of any member: `label`, then `alias`, then `lexical`.

Each assertion's content is folded once per call. Content that isn't loaded can't be scanned, so this is O(total content) per call, which `BenchmarkRecallQuery` measures.

The resolver returns `[]candidate{root, resolution}`. #13 appends `semantic` candidates for classes the baseline didn't find. It never removes or relabels any.

### D5. Stage two filters, marks, and ranks
For each candidate class:
1. `entries := Recall(…)`. A class with no entries is dropped, which covers the scope and retraction scenarios.
2. `matched = true` on each entry whose folded content contains the whole query or any term.
3. Rank keys: `whole` is true if the folded whole query is a substring of any member's label or alias, or of any entry's content. `terms` counts the distinct terms that are a substring of any of those.

Because the keys are computed from `entries`, not the stage-one scan, retracted or out-of-scope objects don't rank ("what the caller cannot see does not rank"). Labels and aliases count for every member of the class, since the class is the result.

### D6. Order, default limit, `omitted`
- Sort by resolution rank, then `whole` (true first), then `terms` (descending), then root id (ascending).
- In query mode, `limit` counts particulars. Omitted or `0` means **10**, and a negative value is a usage error.
- `omitted` is the number of non-empty classes cut.
- Each kept particular's entries stay whole, so no lineage is truncated.

The spec defines no unlimited value and agents can pass a larger `limit`, so a special "all" value isn't added.

*Why 10:* `is`, `the` and `my` resolve almost every particular as `lexical`, and each comes with its full lineage. Ten is a guess that particulars-bench tests (variant B′ against B, at limits 5, 10 and 20) before release.

### D7. Response shape
```json
{"query": "what is the auth timeout?",
 "particulars": [
   {"particular": "par_…", "label": "Auth", "resolution": "lexical",
    "class": ["par_…", "par_…"],
    "entries": [{"id": "clm_…", "…": "…", "matched": true}]}],
 "count": 1, "omitted": 0, "author": "par_…"}
```
- `class` appears only for a merge class with more than one member, as it does in recall by particular today.
- `author` appears only when that filter is given.
- `count` is the number of particulars returned.
- `Entry` gains `Matched bool` with `json:"matched,omitempty"`. Only query mode sets it, so the output of recall by particular is unchanged.
- An empty result is `particulars: []`, not an error. This replaces today's NotFound and ambiguity errors for `query`. An ambiguous name now yields both particulars as `label`.

### D8. Surfaces
**CLI.** `recall --query <text>`:
- It is a usage error together with a positional particular.
- It satisfies the "pass something" check, so it needs no `--topic` or `--author`.
- `--author` is resolved as today and passed as a filter, alongside `--topic`, `--scope`, `--include-retracted` and `--limit`.
- Plain text shows one header per particular, `Label (par_…) [lexical]: 3 entries`, then the existing entry lines with a `[matched]` flag, then `(+N more; raise --limit)` when `omitted > 0`.

**MCP.** `knowledgeRecall` returns a usage error when given both `particular_id` and `query`. Otherwise `query` calls `RecallQuery` with the same option mapping as the CLI. The summary text reads `N particulars (M omitted); top: <label> [<resolution>]`.

**Descriptions.** The `query` parameter: "free-text question or words; resolves particulars by label, alias, or content, then recalls each (exclusive with particular_id)". The `limit` parameter notes that it counts particulars when `query` is given, with a default of 10. The tool description gains one sentence on query mode.

### D9. Guidance
- `skills/particulars/SKILL.md` gains a row: "Search by words: `particulars recall --query "<words>" --json` → `{particulars: [{resolution, entries}]}`". It also gains one line on when to use it: when you don't know the name. Prefer `recall "<name>"` when you do.
- The installed copy is regenerated with `skill install --force` from a HEAD build, and CI checks the body.

## Risks / Trade-offs

- **The MCP `query` contract breaks** for clients that pass a name: they get a grouped response instead of a flat `entries`. → Names still resolve, as `label`. The change is noted as **BREAKING** in the CHANGELOG with a minor version bump, and nothing in this repository's own guidance used `query`.
- **The default limit hides a relevant particular** that matched only common terms. → `omitted` is always reported, and the descriptions say to raise `limit` or use more specific words. The ordering puts more distinct terms first.
- **Scan cost grows with the workspace.** → Folding is a single pass, and `BenchmarkRecallQuery` sits beside `BenchmarkRecall` at 100, 800, 5k and 20k claims. Process cost is still dominated by load.
- **Upstream #37 may change the edge rule.** → The rule is isolated in `isEdge`. Until #37 is settled the merged text governs, so ` Project X` resolves as `lexical`, and a test records that so the change is deliberate.
- **The stage-one scan includes retracted content,** so a particular can be resolved through a retracted claim and returned with its other, visible claims. → The spec requires exactly this ("`par_A` may be resolved"). Its rank counts only visible objects.

## Migration Plan

This is a single release. There is no data migration: nothing on disk changes, and no cache is involved. Rollback is the previous binary.

## Open Questions

- nodelogicau/particulars#37: should `White_Space` count as an edge? If it is accepted before this is applied, adopt it here. Otherwise follow up.
- Is 10 the right default? Settle it from particulars-bench (7.1/7.2 there) before tagging.
