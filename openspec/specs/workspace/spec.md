# Workspace

## Purpose

Creation and discovery of a DKF workspace: `init`, the `dkf.yaml` marker/config file, and the on-disk directory layout.

## Requirements

### Requirement: Workspace initialisation
`particulars init [dir] [--pointer]` SHALL create, in the target directory (default: current directory), the file `dkf.yaml`, the directories `particulars/`, `claims/`, `syntheses/`, `merges/`, and an empty `index.yaml`. It SHALL accept `--base-uri <uri>`, `--author <name>`, `--harness <name>`, and `--scope personal|organisation|public` (default `personal`) to populate `dkf.yaml`. A `--base-uri` lacking a trailing `/` SHALL be stored with one appended and the normalisation reported in the result. With `--pointer` and an explicit `dir` other than the current directory, it SHALL also write `./.dkf` containing the relative path to `dir`, refusing with exit code 1 if a `.dkf` with different content already exists; `--pointer` without such a `dir` SHALL be a usage error. It SHALL fail with exit code 1 if `dkf.yaml` already exists in the target directory.

#### Scenario: Fresh init
- **WHEN** `particulars init ./kb --author ben` is run in an empty directory
- **THEN** `kb/dkf.yaml`, `kb/particulars/`, `kb/claims/`, `kb/syntheses/`, `kb/merges/`, and `kb/index.yaml` exist, and `dkf.yaml` contains `defaults.source.author: ben`

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

### Requirement: dkf.yaml structure
`dkf.yaml` SHALL contain `format: dkf/0.1`, `workspace.id` (a UUIDv7 minted at init), optional `workspace.base-uri` (which, when present, SHALL end in `/`), `defaults.scope`, and `defaults.source` with optional `author`, `harness`, `model`. Unknown keys SHALL be preserved on read and ignored. Opening a workspace whose `base-uri` lacks the trailing `/` SHALL fail with a message naming `dkf.yaml`.

#### Scenario: Workspace id minted
- **WHEN** `particulars init` completes
- **THEN** `dkf.yaml` contains a `workspace.id` that is a valid lowercase UUIDv7

#### Scenario: Base URI recorded
- **WHEN** `particulars init --base-uri https://example.com/particulars/` is run
- **THEN** `dkf.yaml` contains `workspace.base-uri: https://example.com/particulars/`

#### Scenario: Hand-edited base URI without slash
- **WHEN** `dkf.yaml` is edited to `base-uri: https://example.com/particulars` and `recall` is run
- **THEN** the command exits with code 1 and the error names `dkf.yaml`

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
Objects SHALL be stored one per file at `<root>/particulars/<id>.yaml`, `<root>/claims/<id>.yaml`, and `<root>/syntheses/<id>.yaml` according to their type. The CLI SHALL create a missing type directory on first write rather than failing.

#### Scenario: Synthesis file location
- **WHEN** a synthesis with id `syn_X` is created
- **THEN** the file `<root>/syntheses/syn_X.yaml` exists

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
A workspace MAY carry a conventions document for agents: `dkf.md` at the workspace root by default, or the file named by `workspace.conventions` in `dkf.yaml`. The key SHALL be checked lexically on the cleaned, slash-normalised path: an absolute path, a leading slash, or a first segment of `..` is invalid. An invalid key SHALL NOT fail config validation; it SHALL be treated as if the key were absent and reported as a warning — by `serve --mcp` on stderr, and by `particulars workspace` as a warning (stderr in text mode; the `warnings` list in `--json`) alongside `conventions_invalid` carrying the value. `particulars workspace` SHALL report the resolved file when one applies — in JSON as `conventions` — and SHALL report a configured file that does not exist as `conventions` with `conventions_missing: true`. A workspace with neither key nor default file SHALL report nothing. When the key is unset, `dkf.md` is absent, and `CONVENTIONS.md` exists at the root, `serve --mcp` SHALL print a stderr notice that the file is no longer read and how to migrate, `particulars workspace` SHALL carry the same notice as a warning, and `workspace --json` SHALL carry `conventions_legacy: "CONVENTIONS.md"`; the file SHALL NOT be delivered.

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

#### Scenario: Legacy file is noticed, not delivered
- **WHEN** the root holds `CONVENTIONS.md`, no `dkf.md`, and no `workspace.conventions`, and `particulars workspace --json` is run
- **THEN** the result has `conventions_legacy: "CONVENTIONS.md"`, no `conventions`, and a `warnings` entry saying the file is no longer read; in text mode that notice is on stderr

#### Scenario: Legacy file beside a configured one is silent
- **WHEN** the root holds `CONVENTIONS.md` and `dkf.yaml` has `workspace.conventions: TOPICS.md`
- **THEN** the result has `conventions: "TOPICS.md"` and no `conventions_legacy`
