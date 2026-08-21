## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Workspace verb
`particulars workspace` SHALL print the resolved workspace root and how it was resolved — `flag`, `env`, `dkf.yaml`, or `pointer` — together with `workspace.id` and `base-uri`; with `--json` as `{"root", "via", "pointer"?, "id", "base_uri"}`. When no workspace resolves it SHALL exit with code 5.

#### Scenario: Resolved via pointer
- **WHEN** `particulars workspace --json` is run from a directory whose ancestor holds a `.dkf` pointing at a valid workspace
- **THEN** the result has `via: "pointer"`, `pointer` set to the absolute `.dkf` path, and `root` set to the target

#### Scenario: Resolved via environment
- **WHEN** `DKF_WORKSPACE` is set to a valid workspace and `particulars workspace --json` is run
- **THEN** the result has `via: "env"`

#### Scenario: Nothing resolves
- **WHEN** `particulars workspace` is run with no flag, no env, and no `dkf.yaml` or `.dkf` in any ancestor
- **THEN** the command exits with code 5
