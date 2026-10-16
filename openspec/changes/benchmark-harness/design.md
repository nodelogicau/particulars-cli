## Context

There are no Go benchmarks. The dkf/0.2 sharding trigger ("~800 files in `claims/`") is a guess recorded when the layout was deferred. The read paths that would degrade as a workspace grows are graph load from files, index rebuild, `recall` (lineage and merge-class traversal), `conflicts`, and `validate`.

Memory efficacy — agent, guidance, and spec tools on LongMemEval — is benchmarked in `nodelogicau/particulars-bench`, against the DKF specification and any implementation started as a stdio MCP command. It is out of scope here.

## Goals / Non-Goals

**Goals:**
- Time the core read operations across sizes that bracket the sharding trigger.
- Keep `go test ./...` unchanged and CI unblocked.
- Produce output that `benchstat` can compare by hand.

**Non-Goals:**
- Gating CI on performance.
- Memory efficacy (particulars-bench).
- Write-path benchmarks; writes create one file each and are not the suspected bottleneck.

## Decisions

### 1. In-process benchmarks over generated workspaces
A test-only generator writes a workspace through the store package from a fixed seed, with seeded ids and timestamps so two generations are byte-identical. Shape follows the two dogfood workspaces: particulars ≈ claims/20, a Zipf-shaped topic vocabulary, a synthesis every few claims, a small fraction of retractions and merges. Sizes 100, 800, 5,000, 20,000 bracket the trigger by an order of magnitude each side.

*Alternative:* benchmark a copy of a real workspace. Rejected: private content, a single size, and no control over shape.

### 2. Generation outside the timer, cached per run
Each size is generated once per process, before `b.ResetTimer()`. 20,000 claims is too slow to regenerate per benchmark function.

### 3. One end-to-end benchmark
In-process timings miss process start and the index read a user actually waits for. One benchmark execs the built binary's `recall` per size.

### 4. Opt-in by construction, published by CI
`Benchmark*` functions never run under `go test ./...`. A separate CI job on pushes to `main` runs them with a fixed `-benchtime` and uploads the output; it cannot fail the workflow on timings. Shared runners are noisy, so CI output shows trends; decisions use local runs.

## Risks / Trade-offs

- **Synthetic shape differs from real workspaces** → shape parameters taken from the dogfood workspaces and recorded with the generator.
- **CI noise** → non-gating; `benchstat` across several local runs for any decision.
- **Generator drift changes results** → the generator's version is part of its output directory name and noted in benchmark names.

## Open Questions

- Where do the curves bend, and does that move the dkf/0.2 trigger? Answered by the first local run (task 1.7).
