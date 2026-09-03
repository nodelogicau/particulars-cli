## 1. Query

- [x] 1.1 `query.Unresolved(g, subject, opts)` in `internal/query/unresolved.go`: class enumeration as in `Conflicts`, `Analyse` per class for `current` and the unsynthesised count, entry shape per spec, `None identified` filter, effective-scope filter, ascending `(timestamp, id)` order
- [x] 1.2 Tests in `internal/query`: current-only, no-synthesis omitted, retracted current falls back, merged class once with `members`, ordering, `None identified` default and `--include-none`, exact-match filter, scope on a promoted synthesis, unsynthesised count

## 2. CLI

- [x] 2.1 `unresolved [<particular>] [--include-none] [--scope <s>]` in `internal/cli/cmd_query.go`, registered in `root.go`; JSON `{entries, count}`; text block per D5; empty result prints `Nothing unresolved.` and exits 0; unknown particular exits 3
- [x] 2.2 CLI tests: JSON shape, text shape with and without `unsynthesised`, empty workspace, unknown particular

## 3. MCP

- [x] 3.1 `unresolved_list` tool in `internal/mcp/tools.go`: `unresolvedIn{particular_id, scope, include_none}`, read-only annotation, extension-prefixed description, structured result equal to the verb's JSON
- [x] 3.2 MCP test: count with and without `include_none`, `not_found` on an unknown particular

## 4. Skill and docs

- [x] 4.1 `skills/particulars/SKILL.md`: review-flow line under `conflicts` and a "What is still open" row in the query table
- [x] 4.2 Regenerate the installed copy from a HEAD build with `skill install --force`; `skill install --check` green
- [x] 4.3 `README.md` verb table, `docs/mcp.md` tool table, `docs/review-workflow.md` CI summary step prints `unresolved` beside `conflicts`
- [x] 4.4 CHANGELOG entry under Unreleased

## 5. Verification

- [x] 5.1 `go test ./...`, `golangci-lint run`, `openspec validate add-unresolved-verb`
- [x] 5.2 Run `particulars unresolved` against the dogfood workspace: seven entries, DKF showing `unsynthesised: 1`, none saying `None identified`
