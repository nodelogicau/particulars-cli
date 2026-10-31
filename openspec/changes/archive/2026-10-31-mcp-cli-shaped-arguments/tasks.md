## 1. Equal values set twice

- [x] 1.1 In `withShorthand` (`internal/mcp/server.go`), refuse a field set both ways only when the values differ. Equal values are written once (design D1)
- [x] 1.2 Tests: the same quote, and the same hash, given both ways succeed and write the same file as the nested form. Differing values are still refused with "pass quote once"

## 2. Document-author shorthand

- [x] 2.1 Add `DocumentAuthor` (`document_author`) and `DocumentAuthorHyphen` (`document-author`) to `sourceIn`, with descriptions naming `source.document`'s author
- [x] 2.2 Collapse the two spellings, refusing them if they differ. Add an `author` row to the shorthand table, with messages naming `source.document_author` (design D2)
- [x] 2.3 Tests:
  - flat `document_author` and `document-author` each write the same file as the nested form, including the resolved `uri`;
  - an author without a document is refused;
  - differing values set both ways are refused;
  - an unknown author id gets the nested form's error;
  - the five tools' schemas list `source.document_author`

## 3. Top-level topics

- [x] 3.1 Add `Topics` to `assertIn`. In `claimAssert`, write `context.topics` followed by each shorthand topic not already present (design D3)
- [x] 3.2 Tests:
  - top-level `topics` alone writes `context.topics`;
  - top-level `topics` with `context.topics` writes their ordered union;
  - an invalid topic is refused as it would be in `context.topics`

## 4. Preamble, docs and release

- [x] 4.1 Update the preamble's `--topic` and `--document-author` lines (design D4). Extend `TestInstructionsMapCommandsToTools` or a sibling test to check that they mention the shorthand
- [x] 4.2 `docs/mcp.md`: document `topics` and `document_author` as shorthand, keeping the nested form canonical. Add a `CHANGELOG.md` Unreleased entry referencing #18
- [x] 4.3 Run `go test ./...` and `golangci-lint run`
- [x] 4.4 After release, re-run particulars-bench at the #16 settings, record the `claim_assert` errors against the v0.16.0 baseline (157 of 1,075), and comment the result on #18
