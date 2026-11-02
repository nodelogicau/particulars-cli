## 1. Format and month derivation

- [x] 1.1 In `internal/dkf`, replace `FormatVersion` with `ReadFormats` (`dkf/0.1`, `dkf/0.2`) and `WriteFormat` (`dkf/0.1`), and update every user: `config.go`, `index.go`, `workspace.go`, `cmd_maint.go` (design D2)
- [x] 1.2 Add `dkf.MonthOf(id) (string, bool)`. A month derives only from the 36-character hyphenated form (either case) with version 7 and an RFC 4122 variant, from the first 48 bits read as UTC milliseconds (design D1). Test the upstream example (`clm_01a0f3c1-…` → `2026-09`), a draft-era id, a v4 UUID, the 32-hex form, upper case, and a month boundary at the last millisecond of a month
- [x] 1.3 `Config.Validate` accepts any format in `ReadFormats` and refuses others with `ErrUnsupportedFormat`, naming the value found and the supported set. Add `Workspace.Sharded()`

## 2. Paths and loading

- [x] 2.1 `Workspace.Path(id)`: in a sharded workspace, return `<type-dir>/<month or legacy>/<id>.yaml` (design D3)
- [x] 2.2 `Workspace.Load`: the flat branch is unchanged. The sharded branch reads `YYYY-MM` and `legacy` subdirectories, checks each file's directory against its id, and reports `misplaced_file` (with the derived path) for stray top-level files, wrong months, legacy ids in month directories, UUIDv7 ids in `legacy/`, and files in other subdirectories. Misplaced files are never loaded
- [x] 2.3 Tests: build a sharded fixture (several months, `legacy/`, each misplaced case). Check that `recall`, `lineage`, `register` and `workspace` read it the same as the flat equivalent, and that every misplaced case is reported once and not loaded

## 3. Read-only guard and the index

- [x] 3.1 Add `ErrReadOnlyFormat` and `w.writable()`. Call it first in `Create`, `WriteParticular`, `Retract`, `WriteIndex` and `RebuildIndex` (design D4). `export` is a read verb and is not guarded
- [x] 3.2 `CheckIndex` returns `ErrReadOnlyFormat` on a sharded workspace. `validate` skips the comparison there and warns `index_unchecked` (design D5)
- [x] 3.3 Tests:
  - every CLI write verb (`particular define`, `claim assert`, `synthesis create`, `retract`, `merge`, `publish`, `index`) and every MCP write tool refuses in a sharded workspace;
  - each refusal leaves the tree byte-identical;
  - `index --check` exits 1 there

## 4. Validation codes and time checks

- [x] 4.1 `validate`: map `ErrUnsupportedFormat` to `unsupported_format`. Add `future_time` (warning: id instant, object `timestamp`, `retracted.timestamp` after now; one finding per object) and `far_future_id` (error: id month two or more months after now's month), using a clock that tests can override (design D6)
- [x] 4.2 Tests:
  - one day ahead warns, one month ahead warns, two months ahead fails;
  - backdated timestamps are silent;
  - both layouts are covered.

  Then build from HEAD, run `validate` on `particulars-knowledge`, `particulars-feeds` and `particulars-personal`, and confirm that no new error appears

## 5. Version, docs, release

- [x] 5.1 `version` prints the formats read and the one written. `--json` gains `reads` and keeps `format` as the format written
- [x] 5.2 Update the docs that describe paths (`README.md`, `docs/review-workflow.md`, `docs/provenance.md`) with the sharded layout and the read-only note. Add a `CHANGELOG.md` Unreleased entry referencing #11 that says plainly that `dkf/0.2` is read-only in this release
- [x] 5.3 Run `go test ./...` and `golangci-lint run`
