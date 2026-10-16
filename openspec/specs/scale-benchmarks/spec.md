# Scale Benchmarks

## Purpose

Timing the read paths that degrade as a workspace grows — load, index, recall, conflicts, validate, and the built binary — over seeded, generated workspaces of fixed sizes, without affecting the default test run or gating CI.

## Requirements

### Requirement: Generated workspaces at fixed sizes
The scale benchmarks SHALL generate DKF workspaces of 100, 800, 5,000, and 20,000 claims from a fixed seed, containing particulars, topics, syntheses, retractions, and merges in fixed proportions. Two generations with the same seed and size SHALL produce the same objects with the same identifiers, timestamps, and content. Every generated workspace SHALL pass `particulars validate`.

#### Scenario: Deterministic generation
- **WHEN** the 800-claim workspace is generated twice with the same seed
- **THEN** both contain identical files

#### Scenario: Generated workspaces are valid
- **WHEN** any generated workspace is checked with `particulars validate`
- **THEN** it exits 0

### Requirement: Timed operations
For each size the benchmarks SHALL time, outside the generation step: loading the graph from files, rebuilding the index, `recall` of the particular with the most claims, `recall --topic` of the most-used topic, `conflicts` over the whole workspace, and `validate`. One benchmark SHALL exec the built binary for `recall` so that process start and index read are included. Each benchmark SHALL report time, bytes, and allocations per operation.

#### Scenario: All operations at all sizes
- **WHEN** `make bench-scale` is run
- **THEN** output contains a result for each timed operation at each of the four sizes

#### Scenario: Generation is not timed
- **WHEN** a benchmark at 20,000 claims runs
- **THEN** workspace generation happens before the timer starts and is not included in `ns/op`

### Requirement: Never run by the default test command
The scale benchmarks SHALL run only when benchmarks are requested. `go test ./...` without `-bench` SHALL NOT generate a workspace or run a timed operation.

#### Scenario: Plain test run
- **WHEN** `go test ./... -count=1` is run
- **THEN** no benchmark workspace is generated and its duration is not affected by the benchmarks' existence

### Requirement: Published but not gating in CI
CI SHALL run the scale benchmarks in a separate job on pushes to `main` with a fixed `-benchtime`, and SHALL upload the output as a build artifact. The job's result SHALL NOT block merging or releases.

#### Scenario: A slow run
- **WHEN** a push to `main` produces benchmark times twice the previous run's
- **THEN** the benchmark job still succeeds and the artifact records the times
