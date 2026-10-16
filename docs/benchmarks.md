# Benchmarks

Two kinds, in two places:

| | Scale benchmarks | Memory benchmark |
|---|---|---|
| question | how do the read paths cost as a workspace grows? | does an agent remember better with DKF? |
| measures | this implementation | any DKF implementation, through MCP |
| lives in | here, `internal/query/bench_test.go` | [nodelogicau/particulars-bench](https://github.com/nodelogicau/particulars-bench) |
| needs | Go | an Anthropic API key |

## Scale benchmarks

```sh
make bench-scale              # 5 iterations per benchmark; a few minutes
make bench-scale BENCHTIME=1x # a quick look
```

Each benchmark runs over generated workspaces of 100, 800, 5,000, and 20,000
claims. The generator (`internal/query/benchgen_test.go`) writes through the
store from a fixed seed, so files are exactly what the CLI writes and two
generations are byte-identical. Its proportions come from the two dogfood
workspaces: one particular per 10 claims, a synthesis after every 3–9 pending
claims of a particular, 2% of claims retracted, one merge per 50 particulars,
1–3 topics per claim from a Zipf-shaped vocabulary of a third of the claim
count. Every generated workspace is validated, with a clean index check,
before anything is timed. Change the shape and you must bump `genVersion`.

| Benchmark | Times |
|---|---|
| `Load` | parsing every object file into the graph |
| `RebuildIndex` | `index` |
| `Recall` | `recall` of the particular with the most claims, graph already loaded |
| `RecallTopic` | `recall --topic` of the most-used topic, graph already loaded |
| `Conflicts` | `conflicts` over the whole workspace, graph already loaded |
| `Validate` | `validate`, including its load |
| `ExecRecall` | the built binary running `recall --json`: process start, load, and output, as a user waits for them |

Plain `go test ./...` never generates a benchmark workspace; it runs only a
120-claim determinism and validity check of the generator. CI runs the
benchmarks on every push to `main` in a job that cannot fail the workflow and
uploads the output as the `bench-scale-<sha>` artifact. Shared runners are
noisy: use CI output for trends, and compare local runs with
[`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) for decisions.

### First results (2026-10-16, Apple M2 Pro, generator `g1`)

| claims | files | `Load` | `ExecRecall` | `RebuildIndex` | `Validate` | `Recall` | `Conflicts` |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 100 | 123 | 7 ms | 70 ms | 18 ms | 34 ms | 0.02 ms | 0.02 ms |
| 800 | ~1,000 | 60 ms | 85 ms | 139 ms | 0.29 s | 0.16 ms | 0.15 ms |
| 5,000 | ~6,200 | 0.40 s | 0.66 s | 0.81 s | 1.7 s | 1.2 ms | 1.0 ms |
| 20,000 | ~25,000 | 1.9 s | 2.2 s | 3.3 s | 7.9 s | 6.1 ms | 5.7 ms |

What they say:

- **Every curve is linear; nothing bends at 800.** Loading costs ~75–95 µs
  per object file at every size.
- **The cost a user waits for is the load, not the query.** In-process
  `recall` and `conflicts` stay under 10 ms at 20,000 claims. Every invocation
  parses every file, so `ExecRecall` tracks `Load`.
- **The sharded layout would not change this.** dkf/0.2 moves files into
  month directories, but each verb would still read all of them. What would
  change it is answering `recall` from `index.yaml` without opening object
  files, which the spec already permits. That is an implementation change,
  not a format change.
- **`validate` allocates heavily:** ~370 KB per claim, 7.3 GB at 20,000.
  It runs in CI rather than interactively, but it is the first thing to look
  at if those runs slow down.
- At the dogfood workspace's growth rate (~240 claims a month), `recall`
  passes half a second (~4,000 claims) in about a year and a half.

The dkf/0.2 trigger was never about CLI speed. It is the GitHub
1,000-file directory listing, which these numbers neither move nor measure.

## Memory benchmark

[particulars-bench](https://github.com/nodelogicau/particulars-bench) runs
LongMemEval against any implementation that can be started as a stdio MCP
server bound to a workspace. To benchmark a local build:

```sh
make build
particulars-bench … --impl dev="$PWD/dist/particulars serve --mcp --workspace {workspace}"
```

Variants of recall-by-query (#12, #13) are compared by running the same cached
ingest against different builds.
