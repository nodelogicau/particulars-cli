## 1. Terms and matching (`internal/query`)

- [x] 1.1 `isEdge` (category P), and `parseQuery` (split on `White_Space`, trim edges, drop empty terms, fold with `cases.Fold()` using a caser per call, dedupe after folding); table tests for `timeout?`, `PROJ-1234.`, `v1.2.3?`, `"Project X"`, `?!`, `Straße` versus `STRASSE`, and a test recording that ` Project X` keeps the leading space until nodelogicau/particulars#37 is settled.
- [x] 1.2 `exactIn(foldedQuery, foldedName)`: an occurrence test over every occurrence, overlapping ones included, with only `isEdge` runes outside it, skipping empty names; tests for `C#?` against `C#` (exact), `C` against `C#` (not exact), `Project X?` (exact), and `the Project X` (not exact).

## 2. Recall by query (`internal/query`)

- [x] 2.1 Confirm that `g.ClassOf` returns members sorted, with the lowest id first, or sort them; it determines the class root.
- [x] 2.2 Stage one, a resolver returning `[]candidate{root, resolution}`: scan particulars (labels and aliases) and all assertions (content), ignoring filters, map each to its class root, keep the best resolution, and return nothing for an empty query.
- [x] 2.3 Stage two, `RecallQuery(g, text, opts) (QueryResult, error)`: call `Recall` per class with `Subject` set to the root and `Limit: 0`; drop empty classes; set `Entry.Matched` (`json:"matched,omitempty"`); compute the `whole` and `terms` keys from the members' labels and aliases and the filtered entries; sort; apply `limit` (default 10, error on a negative value); and compute `omitted`.
- [x] 2.4 Tests translating every *Recall by query* scenario in the delta spec, including the superseded match, the retracted match not ranking, scope dropping a class without counting it in `omitted`, a merge class once under its lowest id, both ordering keys, the id tiebreak, and exact matches surviving the default limit.

## 3. CLI (`internal/cli/cmd_query.go`)

- [x] 3.1 The `--query` flag: a usage error with a positional particular; it satisfies the "pass something" check; it maps `--author`, `--topic`, `--scope`, `--include-retracted` and `--limit` into `RecallOptions`; and it emits the D7 JSON shape.
- [x] 3.2 Plain-text rendering: a header per particular with label, id, resolution and entry count; a `[matched]` flag on entries; and a `(+N more; raise --limit)` footer. Update `Use`, `Long`, and the `--limit` flag help for query mode.
- [x] 3.3 CLI tests: the usage error, the JSON shape, and an empty result exiting 0 for `?!`.

## 4. MCP (`internal/mcp/tools.go`)

- [x] 4.1 In `knowledgeRecall`, remove the `query` to `particular_id` fallback, return a usage error for both together, and route `query` to `RecallQuery` with the same option mapping as the CLI; the summary text is `N particulars (M omitted); top: <label> [<resolution>]`.
- [x] 4.2 Update the descriptions of the `query` and `limit` parameters and add one sentence about query mode to the tool description.
- [x] 4.3 Rewrite `server_test.go:161` (`query` as an alias) to assert the grouped result; add parity with `recall --query --json`, the usage error for both, and a name passed as `query` resolving as `label`.

## 5. Guidance

- [x] 5.1 In `skills/particulars/SKILL.md`, add a "Search by words" row to the query table and a line on when to use `--query` (the name is unknown) versus `recall "<name>"`.
- [x] 5.2 Regenerate `.claude/skills/particulars/SKILL.md` with `skill install --force` from a binary built from HEAD; run the skill test.

## 6. Benchmarks and docs

- [x] 6.1 Add `BenchmarkRecallQuery` to `internal/query/bench_test.go` at 100, 800, 5k and 20k claims, with a fixed query of two vocabulary terms plus a common word and a `?`; run `make bench-scale` and add the rows to `docs/benchmarks.md`.
- [x] 6.2 Add `CHANGELOG.md` entries: the `recall --query` feature, and a **BREAKING** note that MCP `knowledge_recall`'s `query` now returns grouped results.

## 7. Verify and follow up

- [x] 7.1 Run `go test ./...`, `golangci-lint run`, and `openspec validate recall-query --strict`.
- [x] 7.2 Check upstream nodelogicau/particulars#37: if `White_Space` is accepted as an edge, change `isEdge`, flip the test from 1.1, and add the whitespace scenarios to the delta spec; otherwise leave a note on #12.
- [ ] 7.3 Before tagging, run particulars-bench over one cached ingest with `f63ed46` (A: the last build before this change, which already repeats MCP results as text; the v0.14.0 release doesn't, so its agent sees only counts, per #15), with this change (B′ at the default limit), and at limits 5 and 20; confirm or change the default of 10 and report it on #12.
