## 1. Implementation

- [x] 1.1 Make `isEdge` category P or `White_Space` (`unicode.IsPunct || unicode.IsSpace`).
- [x] 1.2 Record each class's longest exact label or alias (code points, folded) in stage one, checking aliases even when the label is exact.
- [x] 1.3 Order `label` and `alias` particulars by that length first, then whole query, then distinct terms, then id.
- [x] 1.4 Update `recall --help`.

## 2. Tests

- [x] 2.1 Flip the parse and exact-match cases that pinned untrimmed edge whitespace.
- [x] 2.2 Add scenarios: `Project X ?`, ` Project X\n`, `Project  X`, three spaces, a trailing space against a phrase that ends a claim, `C`/`C#` under `limit 1`, and two exact aliases of different lengths.
- [x] 2.3 Confirm that each new rule's tests fail with the rule reverted.

## 3. Wrap-up

- [x] 3.1 CHANGELOG entry under Unreleased.
- [x] 3.2 Archive the change and sync `knowledge-query`.
- [x] 3.3 Comment on #12 that #37 is implemented. #12 stays open for #38 and the LongMemEval_S run.
