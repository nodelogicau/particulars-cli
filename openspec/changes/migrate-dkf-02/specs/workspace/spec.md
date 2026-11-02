## MODIFIED Requirements

### Requirement: Workspace initialisation
`particulars init [dir] [--pointer]` SHALL create, in the target directory (default: current directory), the file `dkf.yaml`, the directories `particulars/`, `claims/`, `syntheses/`, `merges/`, and an empty `index.yaml`. It SHALL write `format: dkf/0.2` unless given `--format dkf/0.1`; `--format` with any other value SHALL be a usage error. It SHALL accept `--base-uri <uri>`, `--author <name>`, `--harness <name>`, and `--scope personal|organisation|public` (default `personal`) to populate `dkf.yaml`. A `--base-uri` lacking a trailing `/` SHALL be stored with one appended and the normalisation reported in the result. With `--pointer` and an explicit `dir` other than the current directory, it SHALL also write `./.dkf` containing the relative path to `dir`, refusing with exit code 1 if a `.dkf` with different content already exists; `--pointer` without such a `dir` SHALL be a usage error. It SHALL fail with exit code 1 if `dkf.yaml` already exists in the target directory.

#### Scenario: Fresh init
- **WHEN** `particulars init ./kb --author ben` is run in an empty directory
- **THEN** `kb/dkf.yaml`, `kb/particulars/`, `kb/claims/`, `kb/syntheses/`, `kb/merges/`, and `kb/index.yaml` exist, `dkf.yaml` contains `format: dkf/0.2` and `defaults.source.author: ben`, and `index.yaml` is a `dkf/0.2` root

#### Scenario: Init with pointer
- **WHEN** `particulars init ./knowledge --pointer` is run at a repository root
- **THEN** `./.dkf` contains `knowledge` and `particulars workspace` run from the root resolves to `./knowledge` via the pointer

#### Scenario: Pointer without a subdirectory
- **WHEN** `particulars init --pointer` is run with no `dir`
- **THEN** the command exits with code 2

#### Scenario: Base URI normalised
- **WHEN** `particulars init --base-uri https://example.com/particulars` is run
- **THEN** `dkf.yaml` contains `base-uri: https://example.com/particulars/` and the result reports the normalisation

#### Scenario: Re-init refused
- **WHEN** `particulars init` is run in a directory that already contains `dkf.yaml`
- **THEN** the command exits with code 1 and modifies nothing

#### Scenario: Init in the flat layout
- **WHEN** `particulars init ./kb --format dkf/0.1` is run
- **THEN** `kb/dkf.yaml` contains `format: dkf/0.1` and `kb/index.yaml` is a single `dkf/0.1` document

#### Scenario: Init with an unknown format
- **WHEN** `particulars init ./kb --format dkf/0.3` is run
- **THEN** the command exits with code 2 and creates nothing

### Requirement: dkf.yaml structure
`dkf.yaml` SHALL contain `format`, `workspace.id` (a UUIDv7 minted at init), optional `workspace.base-uri` (which, when present, SHALL end in `/`), `defaults.scope`, and `defaults.source` with optional `author`, `harness`, `model`. `format` SHALL be one the CLI reads — `dkf/0.1` (the flat layout) or `dkf/0.2` (the sharded layout) — and the CLI SHALL write `dkf/0.2` into a new workspace unless asked for `dkf/0.1`. Opening a workspace whose `format` is any other value SHALL fail with a message naming `dkf.yaml`, the value found, and the formats the CLI reads. Unknown keys SHALL be preserved on read and ignored. Opening a workspace whose `base-uri` lacks the trailing `/` SHALL fail with a message naming `dkf.yaml`.

#### Scenario: Workspace id minted
- **WHEN** `particulars init` completes
- **THEN** `dkf.yaml` contains a `workspace.id` that is a valid lowercase UUIDv7

#### Scenario: Base URI recorded
- **WHEN** `particulars init --base-uri https://example.com/particulars/` is run
- **THEN** `dkf.yaml` contains `workspace.base-uri: https://example.com/particulars/`

#### Scenario: Hand-edited base URI without slash
- **WHEN** `dkf.yaml` is edited to `base-uri: https://example.com/particulars` and `recall` is run
- **THEN** the command exits with code 1 and the error names `dkf.yaml`

#### Scenario: A format the CLI does not read
- **WHEN** `dkf.yaml` says `format: dkf/0.3` and `recall` is run
- **THEN** the command exits with code 1 and the error names `dkf.yaml`, `dkf/0.3`, and `dkf/0.1, dkf/0.2`

#### Scenario: A sharded workspace opens
- **WHEN** `dkf.yaml` says `format: dkf/0.2` and `recall` is run
- **THEN** the command reads the workspace and succeeds

## ADDED Requirements

### Requirement: Migration
`particulars migrate [--dry-run]` SHALL move a `dkf/0.1` workspace to `dkf/0.2` as `workspace-layout` defines: it SHALL rewrite `format` in `dkf.yaml` to `dkf/0.2` by editing that one line, preserving every other byte of the file; then rename every object and record file, byte-identical, to the path its id derives — `<type-dir>/<YYYY-MM>/<id>.yaml`, or `<type-dir>/legacy/<id>.yaml` for an id that is not a UUIDv7; then regenerate the index in the `dkf/0.2` shape, preserving the previous index's unknown entry types and unknown entry fields. No id, field, `source`, `retracted` block, or byte of an object file SHALL change. Before writing anything it SHALL refuse with exit code 1, naming every offending file, when a file in a type directory fails to load, when a type directory holds a subdirectory containing `.yaml` files other than month or `legacy` directories, when a target path exists with different content, or when `dkf.yaml` does not carry exactly one plain `format: dkf/0.1` line. Run on a `dkf/0.2` workspace it SHALL move any object file still at the top of a type directory and regenerate the index, so that an interrupted migration completes; with nothing to move and a current index it SHALL report the workspace already migrated and exit 0. `--dry-run` SHALL report the moves, the months that would be sealed, and the legacy ids, writing nothing. The result SHALL report the number of files moved, the sealed months, and the legacy ids.

#### Scenario: Migrating a flat workspace
- **WHEN** `particulars migrate` runs in a `dkf/0.1` workspace with claims minted in August and November 2026
- **THEN** every object file is under its month directory with identical bytes, `dkf.yaml` says `format: dkf/0.2` and is otherwise unchanged, `index --check` exits 0, and `validate` reports nothing it did not report before

#### Scenario: A legacy id
- **WHEN** the workspace holds `claims/clm_07m3zp9s2q1r4t8v.yaml`
- **THEN** it moves to `claims/legacy/clm_07m3zp9s2q1r4t8v.yaml` and its entry is in `index/legacy.yaml`

#### Scenario: An unreadable file
- **WHEN** a claim file fails to parse
- **THEN** `migrate` exits 1 naming it, and no file — `dkf.yaml` included — has changed

#### Scenario: Resuming
- **WHEN** a migration stopped after rewriting `dkf.yaml` and moving some files, and `migrate` is run again
- **THEN** the remaining files move, the index is regenerated, and the workspace validates

#### Scenario: Already migrated
- **WHEN** `migrate` runs in a `dkf/0.2` workspace with no file to move and a current index
- **THEN** it reports the workspace already migrated, exits 0, and writes nothing

#### Scenario: Dry run
- **WHEN** `particulars migrate --dry-run` is run in a `dkf/0.1` workspace
- **THEN** it reports each move, the months that would seal, and the legacy ids, and no file changes

#### Scenario: Comments and unknown keys survive
- **WHEN** `dkf.yaml` carries a comment and a key the CLI does not know
- **THEN** after migration both are present, byte for byte, and only the `format` line differs
