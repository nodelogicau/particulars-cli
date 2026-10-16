## 1. Scale benchmarks

- [x] 1.1 Add a test-only, seeded workspace generator (seeded ids and timestamps; particulars ≈ claims/10, Zipf topics, syntheses, retractions, merges) that writes through the store package; check its proportions against the two dogfood workspaces.
- [x] 1.2 Test that two generations with one seed are byte-identical and that every size passes `validate`.
- [x] 1.3 Add `Benchmark*` functions for graph load, index rebuild, `recall` (most-claimed particular), `recall --topic` (most-used topic), `conflicts`, and `validate` at 100/800/5k/20k, with generation outside the timer and `b.ReportAllocs()`.
- [x] 1.4 Add the exec benchmark that runs the built binary's `recall` against each size.
- [x] 1.5 Confirm `go test ./... -count=1` time is unchanged; add `make bench-scale`.
- [x] 1.6 Add the non-gating CI job on pushes to `main` with fixed `-benchtime`, uploading the output as an artifact.
- [x] 1.7 Run locally; record where (if anywhere) the curves bend, and update the dkf/0.2 deferral trigger if the evidence moves it.

## 2. Docs and verify

- [x] 2.1 `docs/benchmarks.md`: the scale benchmarks, how to run them, and how to point particulars-bench at a local build.
- [x] 2.2 `CHANGELOG.md` entry under Unreleased.
- [x] 2.3 `go test ./...` and `golangci-lint run ./...`.
