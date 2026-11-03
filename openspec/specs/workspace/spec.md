# Workspace

## Purpose

Creation and discovery of a DKF workspace: `init`, the `dkf.yaml` marker/config file, and the on-disk directory layout.
## Requirements
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

### Requirement: Explicit workspace path
When `--workspace <dir>` or `$DKF_WORKSPACE` is given, the CLI SHALL use that directory as the workspace root if it contains `dkf.yaml`; otherwise, if it contains a `.dkf` pointer file, it SHALL resolve the pointer exactly as discovery does — one hop, target MUST contain `dkf.yaml` — and report the resolution as `pointer`. A directory containing neither SHALL fail with exit code 5 naming the directory and both file names.

#### Scenario: Explicit path holding a pointer
- **WHEN** `/repo/.dkf` contains `knowledge`, `/repo/knowledge/dkf.yaml` exists, and a verb is run with `--workspace /repo` (or `DKF_WORKSPACE=/repo`)
- **THEN** the verb operates on `/repo/knowledge` and `particulars workspace --json` reports `via: "pointer"` with `pointer` set to `/repo/.dkf`

#### Scenario: Explicit path with neither file
- **WHEN** `--workspace /empty` names a directory with no `dkf.yaml` and no `.dkf`
- **THEN** the command exits with code 5 and the message names `/empty`, `dkf.yaml`, and `.dkf`

### Requirement: Upward workspace discovery
When no explicit workspace is given, the CLI SHALL search the current directory and each ancestor in turn. At each directory it SHALL use that directory as the workspace root if it contains `dkf.yaml`; otherwise, if it contains a `.dkf` pointer file, it SHALL resolve the pointer's first non-blank, non-comment line as a path (relative to the pointer's directory, or absolute) and use that directory, which MUST contain `dkf.yaml`. A pointer whose target lacks `dkf.yaml` SHALL fail with exit code 5 naming both the pointer and the target. Pointers SHALL NOT chain.

#### Scenario: Run from subdirectory
- **WHEN** a workspace exists at `/kb` and a verb is run from `/kb/claims`
- **THEN** the verb operates on the workspace rooted at `/kb`

#### Scenario: Nearest wins
- **WHEN** workspaces exist at `/outer` and `/outer/inner` and a verb is run from `/outer/inner/sub`
- **THEN** the verb operates on `/outer/inner`

#### Scenario: Pointer from a repository root
- **WHEN** `/repo/.dkf` contains `knowledge`, `/repo/knowledge/dkf.yaml` exists, and a verb is run from `/repo/src`
- **THEN** the verb operates on `/repo/knowledge`

#### Scenario: dkf.yaml beats a pointer in the same directory
- **WHEN** a directory contains both `dkf.yaml` and `.dkf`
- **THEN** that directory is the workspace and the pointer is ignored

#### Scenario: Dangling pointer
- **WHEN** `/repo/.dkf` contains `knowledge` and `/repo/knowledge/dkf.yaml` does not exist
- **THEN** the command exits with code 5 and the message names `/repo/.dkf` and `/repo/knowledge`

### Requirement: Directory layout
Objects SHALL be stored one per file in the type directory for their id's prefix — `particulars`, `claims`, `syntheses`, `merges`, or `publishes` — laid out as `dkf.yaml`'s `format` declares, never inferred from the directory contents. In a `dkf/0.1` workspace the file SHALL be `<root>/<type-dir>/<id>.yaml`. In a `dkf/0.2` workspace it SHALL be `<root>/<type-dir>/<YYYY-MM>/<id>.yaml`, where `<YYYY-MM>` is the UTC calendar month of the UUIDv7 timestamp in the id, or `<root>/<type-dir>/legacy/<id>.yaml` when the id's identifier part is not a UUID of version 7. The path SHALL be derived from the id alone. The CLI SHALL create a missing type directory on first write rather than failing.

#### Scenario: Synthesis file location
- **WHEN** a synthesis with id `syn_X` is created in a `dkf/0.1` workspace
- **THEN** the file `<root>/syntheses/syn_X.yaml` exists

#### Scenario: Reading a sharded claim
- **WHEN** a `dkf/0.2` workspace holds `claims/2026-09/clm_01a0f3c1-4d20-7b8e-9a11-6c2f4e7d9b03.yaml`, whose id's UUIDv7 timestamp falls in September 2026, and `recall` names its subject
- **THEN** the claim is returned

#### Scenario: Reading a legacy id
- **WHEN** a `dkf/0.2` workspace holds `claims/legacy/clm_07m3zp9s2q1r4t8v.yaml`
- **THEN** the claim is read like any other

#### Scenario: Assertion time does not move a file
- **WHEN** a claim carrying `timestamp: 2019-03-01T00:00:00Z` has an id minted in September 2026
- **THEN** its derived path is under `claims/2026-09/`

### Requirement: Workspace verb
`particulars workspace` SHALL print the resolved workspace root and how it was resolved — `flag`, `env`, `dkf.yaml`, or `pointer` — together with `workspace.id` and `base-uri`; with `--json` as `{"root", "via", "pointer"?, "id", "base_uri"}`. When no workspace resolves it SHALL exit with code 5.

#### Scenario: Resolved via pointer
- **WHEN** `particulars workspace --json` is run from a directory whose ancestor holds a `.dkf` pointing at a valid workspace
- **THEN** the result has `via: "pointer"`, `pointer` set to the absolute `.dkf` path, and `root` set to the target

#### Scenario: Nothing resolves
- **WHEN** `particulars workspace` is run with no flag, no env, and no `dkf.yaml` or `.dkf` in any ancestor
- **THEN** the command exits with code 5

### Requirement: Pointer verb
`particulars workspace pointer [workspace-dir] [--at <dir>] [--force]` SHALL write a `.dkf` pointer in `--at` (default: the current directory) naming a workspace that already exists, so that the workspace resolves from that directory and below. With no argument it SHALL name the workspace that would be used now, resolved by the ordinary precedence. The target SHALL be written relative when the workspace lies within the pointer's directory and absolute otherwise, and the result SHALL report which; with `--json` as `{"pointer", "root", "target", "relative"}`.

It SHALL be a usage error (exit code 2) when the pointer directory is the workspace itself, when it already contains `dkf.yaml` — which wins over a pointer at the same level — or when `--at` is not a directory. Writing a pointer that already names the same target SHALL succeed unchanged; one naming a different target SHALL fail with exit code 1 and leave the file untouched unless `--force` is given.

#### Scenario: Pointer for an existing workspace
- **WHEN** `particulars workspace pointer ./knowledge` is run at a repository root
- **THEN** `./.dkf` contains `knowledge`, the result reports `relative: true`, and a verb run from `src/pkg/` resolves via the pointer

#### Scenario: Workspace outside the pointer's tree
- **WHEN** the named workspace is not under the pointer directory
- **THEN** the target is written absolute, `relative` is `false`, and the result warns that the pointer is machine-specific and should not be committed

#### Scenario: Replacing a pointer
- **WHEN** a `.dkf` naming a different workspace already exists
- **THEN** the command exits 1 mentioning `--force`, the file is unchanged, and repeating with `--force` rewrites it

#### Scenario: Shadowed pointer refused
- **WHEN** `--at` names a directory that already contains `dkf.yaml`
- **THEN** the command exits 2 explaining that `dkf.yaml` wins over a pointer at the same level

#### Scenario: Resolved via environment
- **WHEN** `DKF_WORKSPACE` is set to a valid workspace and `particulars workspace --json` is run
- **THEN** the result has `via: "env"`

### Requirement: Workspace conventions file
A workspace MAY carry a conventions document for agents: `dkf.md` at the workspace root by default, or the file named by `workspace.conventions` in `dkf.yaml`. The key SHALL be checked lexically on the cleaned, slash-normalised path: an absolute path, a leading slash, or a first segment of `..` is invalid. An invalid key SHALL NOT fail config validation; it SHALL be treated as if the key were absent and reported as a warning — by `serve --mcp` on stderr, and by `particulars workspace` as a warning (stderr in text mode; the `warnings` list in `--json`) alongside `conventions_invalid` carrying the value. `particulars workspace` SHALL report the resolved file when one applies — in JSON as `conventions` — and SHALL report a configured file that does not exist as `conventions` with `conventions_missing: true`. A workspace with neither key nor default file SHALL report nothing.

#### Scenario: Default conventions file
- **WHEN** `dkf.md` exists at the workspace root and `particulars workspace --json` is run
- **THEN** the result has `conventions: "dkf.md"`

#### Scenario: Configured conventions file
- **WHEN** `dkf.yaml` has `workspace.conventions: TOPICS.md` and that file exists
- **THEN** the result has `conventions: "TOPICS.md"`

#### Scenario: Configured but missing
- **WHEN** `workspace.conventions` names a file that does not exist
- **THEN** the result has `conventions` and `conventions_missing: true`, and the command exits 0

#### Scenario: Escaping path warns and is ignored
- **WHEN** `dkf.yaml` has `workspace.conventions: ../secrets.md` and `dkf.md` exists at the root
- **THEN** every verb opens the workspace, `workspace --json` has `conventions_invalid: "../secrets.md"`, `conventions: "dkf.md"`, and a `warnings` entry naming the value, and in text mode stderr carries that warning

#### Scenario: Absolute path is invalid on every platform
- **WHEN** `dkf.yaml` has `workspace.conventions: /etc/motd`
- **THEN** the key is treated as unset and reported as `conventions_invalid`

### Requirement: Sealed months gain nothing new
In a `dkf/0.2` workspace, a month SHALL be sealed once the workspace holds an id minted two or more months after it, the current month being that of the newest id found in the month directories. Creating an object whose new UUIDv7 id falls in a sealed month SHALL be refused, and writing a retraction whose `timestamp` falls in a sealed month SHALL be refused, each before any file is written, with code `sealed_month`, a message naming the sealed month and the workspace's current month and saying to check the machine's clock, and exit code 1. The CLI and MCP server SHALL mint ids and date retractions at the machine's clock, so that only a clock more than a month behind the workspace can reach either refusal.

#### Scenario: A clock months behind
- **WHEN** the newest id in a `dkf/0.2` workspace is from December 2026 and `claim assert` runs on a machine whose clock reads September 2026
- **THEN** the command exits 1 with `sealed_month`, naming September and December, and writes nothing

#### Scenario: A clock days behind across a boundary
- **WHEN** the newest id is from 2 December 2026 and a machine two days slow mints on what it reads as 30 November
- **THEN** the claim is written under `claims/2026-11/`, which is open, and nothing is refused

#### Scenario: Retracting with a clock months behind
- **WHEN** the newest id is from December 2026 and `retract` runs on a machine whose clock reads September 2026
- **THEN** the command exits 1 with `sealed_month` and the file is unchanged

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

