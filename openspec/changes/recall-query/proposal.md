## Why

`knowledge_recall(query)` is an alias of `particular_id` (`internal/mcp/tools.go:521-524`). It matches labels and aliases exactly, so a question an agent asks in its own words fails to resolve, and object `content` is never searched. The DKF `recall-query` capability now defines recall by query: two stages, a model-free lexical baseline that every implementation runs, a `resolution` marker, and a fixed order within exact and lexical matches (nodelogicau/particulars `recall-vector-cache`, then `recall-query-terms` via #33/#36). The reference implementation should implement it, and particulars-bench needs it as the B variant to measure (#12).

## What Changes

- **BREAKING (MCP):** `knowledge_recall`'s `query` is no longer an alias of `particular_id`. It runs recall by query: the lexical baseline resolves particulars, then each is recalled as recall by particular would, under the same filters. Passing both `query` and `particular_id` is a usage error.
- New CLI flag `particulars recall --query <text>`, mutually exclusive with the positional particular and combinable with `--author`, `--topic`, `--scope`, `--include-retracted`, and `--limit`. Its `--json` equals the MCP structured result.
- The baseline follows the upstream spec. A term is a run of non-whitespace characters. Edge punctuation is trimmed from each term and from the whole query, and interior characters are kept. Terms combine by OR. Matching uses full Unicode case folding. A query that is empty after trimming returns an empty result and is not an error.
- Each particular carries `resolution`: `label` or `alias` when only punctuation surrounds the name in the query, otherwise `lexical`. Particulars are ordered by resolution value, then by whether they match the whole query, then by the number of distinct terms matched, then by id. Matches are counted only against what survives the filters.
- The query-mode response is grouped by particular. Each merge class appears once, under its lowest id, with `class` listed when it has more than one member. Objects whose content matched are marked `matched: true`. The response also carries `query`, `count` (particulars returned), and `omitted` (resolved particulars cut by `limit`).
- In query mode, `limit` counts particulars and keeps each one's lineage whole. It defaults to 10, because short terms such as `is` and `the` resolve almost the whole workspace.
- The tool description, the `query` parameter description, and the skill body teach the new mode. The installed skill copy is regenerated.
- A `BenchmarkRecallQuery` joins the scale benchmarks.
- No semantic stage. That is #13. The resolver keeps a seam for it.

## Capabilities

### New Capabilities

None. The upstream `recall-query` capability is implemented through `knowledge-query` and `mcp-server`.

### Modified Capabilities

- `knowledge-query`: adds *Recall by query*: the `--query` flag, the baseline, `resolution`, ordering, the grouped response, the matched marker, and query-mode `limit`.
- `mcp-server`: modifies *Spec tool surface* so that `knowledge_recall`'s `query` means recall by query, is exclusive with `particular_id`, and equals `recall --query --json`.

## Impact

- `internal/query`: a new `RecallQuery` (stage one resolution, stage two by `Recall` per class), term handling, and folding with `golang.org/x/text/cases`, which is already a dependency through `x/text`. `Entry` gains `matched`.
- `internal/cli/cmd_query.go`: the `--query` flag, plain-text rendering grouped by particular, and an `omitted` footer.
- `internal/mcp/tools.go`: `knowledgeRecall` branches on `query`, and the tool and parameter descriptions change.
- `skills/particulars/SKILL.md`, plus the regenerated `.claude/skills/particulars/SKILL.md`.
- Tests in `internal/query`, `internal/cli`, and `internal/mcp`. A benchmark in `internal/query/bench_test.go`. Rows in `docs/benchmarks.md`.
- `CHANGELOG.md` gets a breaking-change note for MCP clients that pass a name as `query`. Names still resolve, as `resolution: label`, but the response shape is grouped.
- Pending upstream: nodelogicau/particulars#37 (edge whitespace). The edge-character predicate is a single function, so adopting #37 changes one line and adds scenarios.
