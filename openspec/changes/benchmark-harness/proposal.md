## Why

The dkf/0.2 sharded layout was deferred on 2026-09-12 with a trigger of "claims/ passes ~800 files". That number has never been measured against the operations that would degrade — loading the graph, rebuilding the index, `recall`, `conflicts`, `validate`. A cheap, repeatable measurement replaces the guess with a curve.

Whether `particulars` works as *memory* is a separate question, measured as a system (agent + spec tool surface + guidance) and against the DKF specification rather than this implementation. That benchmark lives in `nodelogicau/particulars-bench`; this change covers only what is specific to this codebase.

## What Changes

- **Scale benchmarks.** Go `testing.B` benchmarks of graph load, index rebuild, `recall`, `recall --topic`, `conflicts`, and `validate` over seeded, generated workspaces of 100, 800, 5,000, and 20,000 claims, plus one benchmark that execs the built binary so process start and index read are counted.
- They run only under `-bench`, so `go test ./...` is unchanged. `make bench-scale` runs them; a non-gating CI job on pushes to `main` publishes the output as an artifact.
- **`docs/benchmarks.md`** describes the scale benchmarks and how to point particulars-bench at a local build (`--impl cli='dist/particulars serve --mcp --workspace {workspace}'`).

Nothing is **BREAKING**. No verb, object file, index, or MCP tool changes.

## Capabilities

### New Capabilities
- `scale-benchmarks`: what workspaces are generated, which operations are timed at which sizes, that they never run under plain `go test`, and how results are published.

### Modified Capabilities
(none)

## Impact

- New `_test.go` benchmark files and a test-only seeded workspace generator.
- `Makefile` target `bench-scale`; a non-gating job in `.github/workflows/ci.yml`; `docs/benchmarks.md`; `CHANGELOG.md`.
- No new module dependencies.
