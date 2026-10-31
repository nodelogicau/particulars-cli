## 1. Shared shorthand struct

- [x] 1.1 Add the embedded struct `flatIn` (`internal/mcp/tools.go`). It holds `Scope`, `Topics`, `Document`, `Quote`, `Hash`, `DocumentAuthor`, `DocumentAuthorHyphen`, `Harness` and `Model`, keeping today's JSON names and descriptions. Move `assertIn`'s shorthand fields into it, embed it in `assertIn` and `synthesisIn`, and add a comment on each embedding that names the folds (design D2)
- [x] 1.2 Add `foldContext` with today's `claim_assert` rules (`context.scope` wins; topics are joined, context first, without duplicates). Use it in `claimAssert` and `synthesisCreate`
- [x] 1.3 Make `foldSource` a method on `flatIn` that takes the nested source. Add `harness` and `model` rows that follow the equal-or-refuse rule, and call it from `synthesisCreate` as well

## 2. Field-by-field document merge

- [x] 2.1 In `foldSource`, when both documents are present, decode each with `documentFrom`, merge `ref`, `author`, `hash` and `quote` one field at a time, refuse a field that differs with "pass document.<field> once", and write the merged document back as a mapping (design D1)
- [x] 2.2 Tests:
  - a string document plus a mapping with the same `ref` writes the mapping;
  - two partial mappings merge;
  - different `ref`s are refused;
  - a decode error still names the unknown key

## 3. Tests for the shared shorthand

- [x] 3.1 `claim_assert`: a top-level `harness` (and `model`) without `source` writes `source.harness`, and a different value in `source` is refused
- [x] 3.2 `synthesis_create`: top-level `harness`, `scope` and `topics` write the same synthesis as the nested form; the flat source fields work; the `synthesis_create` schema lists the top-level fields and states `document` as string | object
- [x] 3.3 The existing `TestTopLevelSource`, `TestSourceShorthand` and topics tests pass. Two `TestTopLevelSource` cases change as D1 intends: a string beside a mapping with the same `ref` now merges rather than being refused, and the differing-document message names the field (`pass document.ref once`)

## 4. Docs and release

- [x] 4.1 `docs/mcp.md`: add the shorthand to the `synthesis_create` row and describe it once for both tools, including the document merge. Add a `CHANGELOG.md` Unreleased entry that references #18
- [x] 4.2 Run `go test ./...` and `golangci-lint run`
- [ ] 4.3 After release, re-run particulars-bench at the #16 settings, record the `claim_assert` errors (16 of 891 on v0.18.0) and the `synthesis_create` errors (8 of 22), and open or update an issue with the result
