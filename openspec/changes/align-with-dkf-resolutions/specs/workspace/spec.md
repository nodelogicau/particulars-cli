## MODIFIED Requirements

### Requirement: Workspace initialisation
`particulars init [dir]` SHALL create, in the target directory (default: current directory), the file `dkf.yaml`, the directories `particulars/`, `claims/`, `syntheses/`, `merges/`, and an empty `index.yaml`. It SHALL accept `--base-uri <uri>`, `--author <name>`, `--harness <name>`, and `--scope personal|organisation|public` (default `personal`) to populate `dkf.yaml`. A `--base-uri` lacking a trailing `/` SHALL be stored with one appended and the normalisation reported in the result. It SHALL fail with exit code 1 if `dkf.yaml` already exists in the target directory.

#### Scenario: Fresh init
- **WHEN** `particulars init ./kb --author ben` is run in an empty directory
- **THEN** `kb/dkf.yaml`, `kb/particulars/`, `kb/claims/`, `kb/syntheses/`, `kb/merges/`, and `kb/index.yaml` exist, and `dkf.yaml` contains `defaults.source.author: ben`

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
