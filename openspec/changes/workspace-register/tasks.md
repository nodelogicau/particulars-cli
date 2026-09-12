## 1. The register

- [x] 1.1 Add a query in `internal/query` that builds the register from a loaded graph: per particular the label, aliases, whether the URI is global or `urn:dkf:`-minted, claim and synthesis counts, and the unsynthesised count (reuse what `conflict_detect` already computes rather than recomputing it); per topic the assertion count and a singleton flag.
- [x] 1.2 Apply the orderings from the spec (claims desc then label asc; count desc then topic asc) in the query, so text and JSON agree and two runs are byte-identical.
- [x] 1.3 Add the text renderer with the size cap, dropping from the tail of each ordering and emitting the omitted counts. Pick the cap in the design's spirit — large enough that neither dogfood workspace truncates.
- [x] 1.4 Tests: determinism across two runs, tie ordering, truncation keeping the head and naming the omitted counts, singleton marking, and that a conventions document does not affect the output.

## 2. The verb

- [x] 2.1 Add `internal/cli/cmd_register.go` for `particulars register [--json]`, following the shape of `topics` and `unresolved`.
- [x] 2.2 Tests: text and `--json` output on a seeded workspace; exit 5 with no workspace, matching the other verbs.

## 3. The MCP instructions

- [x] 3.1 In `internal/mcp/server.go` `instructions()`, insert the register under its own heading after `skill.Body()` and before the conventions block.
- [x] 3.2 Tests: the register appears in `initialize` instructions; it appears when no conventions document exists; the conventions document still follows it.

## 4. The skill body

- [x] 4.1 Add the register step to `skills/particulars/SKILL.md` — read once before choosing a subject or a topic, with the note that an MCP client already has it. Keep the existing loop unchanged. Add `register` to the verbs table.
- [x] 4.2 Regenerate the installed copy with `skill install --force` from a binary built from HEAD, never by copying and never from a release binary, since CI compares the body.

## 5. Docs, changelog, dogfood

- [x] 5.1 `docs/mcp.md`: note that instructions carry the register, and the order of the three sections.
- [x] 5.2 `CHANGELOG.md` entry under Unreleased, carrying the two-workspace evidence in a sentence.
- [x] 5.3 Run `register` against both dogfood workspaces and check the output is what the proposal's table claims; confirm neither truncates at the chosen cap.
- [x] 5.4 Consider trimming the registry sections of `particulars-knowledge/dkf.md` in a follow-up, leaving the prose that says what the tool cannot know. Not part of this change — that workspace is a separate repository.

## 6. Verify

- [x] 6.1 `go test ./...` and `golangci-lint run ./...`.
- [x] 6.2 `particulars validate` and `index --check` clean on both dogfood workspaces.
