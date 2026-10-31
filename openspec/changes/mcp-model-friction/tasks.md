## 1. Source shorthand

- [x] 1.1 Add `Quote` and `Hash` to `sourceIn` (`internal/mcp/server.go`), with descriptions saying each is shorthand for the field on `source.document`
- [x] 1.2 In `source()`, merge the shorthand into the document returned by `documentFrom`. Refuse shorthand with no per-call document, and a field set both ways, with usage errors that name where the field belongs. Apply the blank-quote refusal after the merge (design D1)
- [x] 1.3 Tests: a flat quote beside a string document writes the same file as the nested form; the shorthand fills a mapping document, including a resolved `author`; quote set twice is refused; quote with no document is refused; a blank shorthand quote is refused; the shorthand works on `synthesis_create`

## 2. Document schema

- [x] 2.1 Change `Document any` to a named `documentIn` empty interface, and check that `documentFrom` and the JSON decoding behave as before
- [x] 2.2 Build the `anyOf` (string | object with `ref`, `uri`, `author`, `hash`, `quote`) schema, and a `withSource[In]()` helper that infers each input schema with `TypeSchemas` (design D2). Register `claim_assert`, `synthesis_create`, `claim_retract`, `knowledge_publish` and `particular_merge` with it. Make `github.com/google/jsonschema-go` a direct dependency (`go mod tidy`)
- [x] 2.3 Test: through `ListTools`, each of the five tools' `source.document` has both branches and a description, and `source.quote` is present. A document with an unknown key still gets `documentFrom`'s message naming the key, not an SDK schema error

## 3. MCP preamble

- [x] 3.1 Add the `cliTools` table to `server.go`, covering every command the skill names (design D3), including the rows with no equivalent (`register`, `export`, `index`, `init`, `skill install`, `--hash-document`)
- [x] 3.2 Write the preamble in `instructions()` between the header and the skill body. Move the existing line naming the tool families into it
- [x] 3.3 Tests: the preamble comes before "Recall **before** you assert", and contains "no `particulars` tool", `particular_resolve` and `knowledge_recall`. The set of tools named in `cliTools` equals the set from `ListTools`. The prompt text still equals the instructions

## 4. Docs and release

- [x] 4.1 `docs/mcp.md`: document the `source.quote` / `source.hash` shorthand, naming the nested form as canonical, and the preamble
- [x] 4.2 `CHANGELOG.md`: add an Unreleased entry referencing #16
- [x] 4.3 Run `go test ./...` and `golangci-lint run`
- [ ] 4.4 After release, re-run particulars-bench at the #16 settings (bench `4b5d19f`, oracle with 70 questions, Haiku 5.5 ingest). Record the `claim_assert` error count and the unknown-tool count against the baseline (834 of 1,686, and 58), and comment the result on #16
