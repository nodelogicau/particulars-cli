## 1. Top-level source fields

- [x] 1.1 Add `Document` (`documentIn`), `Quote`, `Hash`, `DocumentAuthor` and `DocumentAuthorHyphen` to `assertIn`, with descriptions naming the `source` field each stands for
- [x] 1.2 Add `foldSource(in *assertIn) (*sourceIn, error)`. It copies or creates the source, fills the empty fields, accepts equal values, and refuses differing values with "pass <field> once: at the top level or in source, not both". `document` is compared with `reflect.DeepEqual`. Call it first in `claimAssert` and pass its result to `s.source` (design D1)
- [x] 1.3 Tests:
  - the flat CLI shape (`document`, `quote`, `document_author`, `scope`, `topics`, `timestamp`) writes the same file as the nested form;
  - a mapping `document` at the top level works;
  - the same value at the top level and in `source` is accepted;
  - a differing `document` or `quote` is refused and writes nothing;
  - a top-level `quote` with no document gets `source.quote needs source.document`;
  - the `claim_assert` schema states the top-level `document` as string | object

## 2. Preamble

- [x] 2.1 Change the preamble's `--document-author` / `--quote` / `--document-hash` and `--topic` lines back to the nested form only (design D3). Update `TestInstructionsMapCommandsToTools` so it checks that the preamble no longer says "may also be given" or "top-level"

## 3. Docs and release

- [x] 3.1 `docs/mcp.md`: add `document?`, `quote?`, `hash?` and `document_author?` to the `claim_assert` row and describe them as shorthand. Add a `CHANGELOG.md` Unreleased entry referencing #18
- [x] 3.2 Run `go test ./...` and `golangci-lint run`
- [ ] 3.3 After release, re-run particulars-bench at the #16 settings. Record the `claim_assert` errors against v0.17.0 (85 of 962) and the number of sessions sending top-level source fields (9 of 70), comment the result on #18, and close it if the flat shape no longer fails
