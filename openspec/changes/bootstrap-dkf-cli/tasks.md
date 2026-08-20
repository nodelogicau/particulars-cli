## 1. Project Setup

- [x] 1.1 Install Go toolchain (no `go` on PATH currently); pin version in `go.mod` and a `.tool-versions`/`mise.toml`
- [x] 1.2 `go mod init github.com/nodelogicau/particulars-cli`; add `spf13/cobra`, `gopkg.in/yaml.v3`, `github.com/google/uuid`
- [x] 1.3 Create package skeleton: `cmd/particulars`, `internal/cli`, `internal/dkf`, `internal/store`, `internal/query`
- [x] 1.4 Add `Makefile`/`Taskfile` targets: build, test, lint (`golangci-lint`), cross-compile matrix with `CGO_ENABLED=0`
- [x] 1.5 Add GitHub Actions workflow: test + lint on PR, GoReleaser config for darwin/linux/windows × amd64/arm64 on tag

## 2. Format Layer (`internal/dkf`)

- [x] 2.1 Define Go types for Particular, Claim, Synthesis, Source, Context, Input, Retracted, with spec field order documented on each struct
- [x] 2.2 Implement ID minting: `<prefix>_<uuidv7>` lowercase, monotonic counter; implement lenient parsing (`^(par|clm|syn)_[A-Za-z0-9-]+$`) and prefix→type mapping
- [x] 2.3 Implement deterministic YAML encoder using `yaml.v3` nodes: 2-space indent, spec key order, literal block scalars for multi-line strings, RFC 3339 `Z` timestamps, omit unset optionals, no document markers
- [x] 2.4 Implement decoder tolerant of unknown keys and foreign IDs
- [x] 2.5 Implement field-level validation rules (scope/role/weight enums, confidence range, required fields, timestamp format) shared by write paths and `validate`
- [x] 2.6 Implement slugify (lowercase, ASCII fold, hyphen-collapse, trim) and URI minting (`base-uri` or `urn:dkf:<ws>:`)
- [x] 2.7 Golden-file tests: encode→decode→encode byte stability for each object type, spec field order, burst-mint ordering, slug edge cases

## 3. Store Layer (`internal/store`)

- [x] 3.1 Implement `dkf.yaml` read/write (format, workspace.id, base-uri, defaults) preserving unknown keys
- [x] 3.2 Implement workspace discovery: `--workspace` flag → `DKF_WORKSPACE` → upward search for `dkf.yaml`; return a typed "no workspace" error
- [x] 3.3 Implement object file paths by type, create-exclusive writes, lazy creation of type directories
- [x] 3.4 Implement `Load()` that reads all objects into an in-memory graph (particulars by id/uri/label/alias; claims and syntheses by id; per-subject lists)
- [x] 3.5 Implement retraction append: append `retracted` block bytes, re-parse, restore original on failure
- [x] 3.6 Implement index model (spec fields + `scope`, `topics`, `timestamp`, `inputs`, `retracted`), deterministic write sorted by id, full rebuild from files, incremental upsert, and byte-for-byte check with diff of missing/extra/changed ids
- [x] 3.7 Implement particular update (label replace, alias union incl. previous label) for idempotent define
- [x] 3.8 Tests using temp directories: discovery precedence and nearest-wins, create-exclusive failure, retraction append/restore, index rebuild with conflict-marker garbage, index check drift

## 4. Query Layer (`internal/query`)

- [x] 4.1 Implement subject resolution (id/uri exact; label/alias case-insensitive) returning zero/one/many
- [x] 4.2 Implement `Recall`: per-subject and/or per-topic filtering, scope filter, retracted filter, topological lineage order with id tie-break, `current` marking, limit
- [x] 4.3 Implement `Lineage`: recursive input expansion with role/weight, depth limit, retracted marking, cycle guard
- [x] 4.4 Implement `Conflicts`: current / reconciled closure / unsynthesised / stale (direct inputs), reporting threshold, priority ordering
- [x] 4.5 Implement `Validate`: structural, referential (dangling refs, input-is-particular, cycles, duplicate URIs), index consistency, advisory warnings (`stale_synthesis`, `orphan_particular`, `non_canonical`)
- [x] 4.6 Table-driven tests for every scenario in `specs/knowledge-query`, `specs/conflict-detection`, `specs/validation`

## 5. CLI Layer (`internal/cli`, `cmd/particulars`)

- [x] 5.1 Root command with global flags `--json`, `--workspace`; exit-code mapping (0/1/2/3/4/5); JSON error envelope on stderr; guard against any TTY reads
- [x] 5.2 Provenance resolution helper: flag → `DKF_AUTHOR|DKF_HARNESS|DKF_MODEL` → `dkf.yaml` defaults
- [x] 5.3 `--content` / `--content-file <path|->` handling shared by assert and synthesis create
- [x] 5.4 `init` with `--base-uri`, `--author`, `--harness`, `--scope`; refuse when `dkf.yaml` exists
- [x] 5.5 `particular define` (`--label`, `--uri`, `--alias`, `created` flag in result) and `particular resolve` (`matches` list, exit 3 on none)
- [x] 5.6 `claim assert` with all flags, index upsert, full claim in result
- [x] 5.7 `claim retract` with `--reason` (required), provenance, `--superseded-by` existence check, double-retract refusal, index update
- [x] 5.8 `synthesis create` with `--input id:role[:weight]` parsing, `--unresolved` required, `--method` default, produced-by resolution, retracted-input warnings
- [x] 5.9 `recall`, `lineage` (JSON nested / text tree), `conflicts` (`--fail-on-conflicts`), `index` / `index --check`, `validate`, `version`
- [x] 5.10 Text renderers for each verb (compact, human-readable; JSON is the contract)
- [x] 5.11 End-to-end tests driving the built binary against temp workspaces covering every `specs/*` scenario not already covered at lower layers, including exit codes and JSON envelopes

## 6. Documentation and Release

- [x] 6.1 README: install, quick start (init → define → assert → synthesis → recall → conflicts), verb reference, exit codes, env vars, relationship to the DKF spec
- [x] 6.2 Document the PR-review workflow (agent branch → PR → merge; post-merge retraction) and a sample GitHub Action running `validate` and `index --check`
- [x] 6.3 Write `SPEC-FEEDBACK.md` (or issues on nodelogicau/particulars) from the design's Spec Feedback list
- [ ] 6.4 Tag `v0.1.0` and verify GoReleaser artifacts run on macOS arm64 and Linux amd64
