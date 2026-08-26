## 1. Preserve

- [x] 1.1 `Index.Unknown []yaml.Node`: decode splits entries by known `type` (the five values); unknown rows kept as raw nodes and out of `Entries`.
- [x] 1.2 Encode merges `Entries` and `Unknown` in ascending-id order; an index with no unknown entries round-trips byte-identically (guard test).
- [x] 1.3 `RebuildIndex` and `UpsertIndex` read-and-carry the unknown set; `BuildIndex` from a graph alone stays as is, with the merge at the workspace layer where the committed file is in hand.

## 2. Check

- [x] 2.1 `CheckIndex` expectation = files + committed unknowns, so unknown types can never be `extra` and real drift still fails.
- [x] 2.2 Tests: a synthetic future entry (`type: signature`) survives rebuild unchanged and in order, `index --check` exits 0, `validate` reports no `index_stale`, `claim assert` upsert carries it, and a genuinely missing claim still fails the check.

## 3. Verification

- [x] 3.1 Full suite + lint; real-workspace round-trip guard; hand-run: inject a fake future entry into a copy of the real workspace's index, rebuild, confirm byte-identical presence and a clean check.
