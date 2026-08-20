## ADDED Requirements

### Requirement: Workspace initialisation
`particulars init [dir]` SHALL create, in the target directory (default: current directory), the file `dkf.yaml`, the directories `particulars/`, `claims/`, `syntheses/`, and an empty `index.yaml`. It SHALL accept `--base-uri <uri>`, `--author <name>`, `--harness <name>`, and `--scope personal|organisation|public` (default `personal`) to populate `dkf.yaml`. It SHALL fail with exit code 1 if `dkf.yaml` already exists in the target directory.

#### Scenario: Fresh init
- **WHEN** `particulars init ./kb --author ben` is run in an empty directory
- **THEN** `kb/dkf.yaml`, `kb/particulars/`, `kb/claims/`, `kb/syntheses/`, and `kb/index.yaml` exist, and `dkf.yaml` contains `defaults.source.author: ben`

#### Scenario: Re-init refused
- **WHEN** `particulars init` is run in a directory that already contains `dkf.yaml`
- **THEN** the command exits with code 1 and modifies nothing

### Requirement: dkf.yaml structure
`dkf.yaml` SHALL contain `format: dkf/0.1`, `workspace.id` (a UUIDv7 minted at init), optional `workspace.base-uri`, `defaults.scope`, and `defaults.source` with optional `author`, `harness`, `model`. Unknown keys SHALL be preserved on read and ignored.

#### Scenario: Workspace id minted
- **WHEN** `particulars init` completes
- **THEN** `dkf.yaml` contains a `workspace.id` that is a valid lowercase UUIDv7

#### Scenario: Base URI recorded
- **WHEN** `particulars init --base-uri https://example.com/particulars/` is run
- **THEN** `dkf.yaml` contains `workspace.base-uri: https://example.com/particulars/`

### Requirement: Upward workspace discovery
When no explicit workspace is given, the CLI SHALL search the current directory and each ancestor in turn for a file named `dkf.yaml` and use the first directory containing it as the workspace root.

#### Scenario: Run from subdirectory
- **WHEN** a workspace exists at `/kb` and a verb is run from `/kb/claims`
- **THEN** the verb operates on the workspace rooted at `/kb`

#### Scenario: Nearest wins
- **WHEN** workspaces exist at `/outer` and `/outer/inner` and a verb is run from `/outer/inner/sub`
- **THEN** the verb operates on `/outer/inner`

### Requirement: Directory layout
Objects SHALL be stored one per file at `<root>/particulars/<id>.yaml`, `<root>/claims/<id>.yaml`, and `<root>/syntheses/<id>.yaml` according to their type. The CLI SHALL create a missing type directory on first write rather than failing.

#### Scenario: Synthesis file location
- **WHEN** a synthesis with id `syn_X` is created
- **THEN** the file `<root>/syntheses/syn_X.yaml` exists
