## ADDED Requirements

### Requirement: Non-interactive operation
The CLI SHALL never prompt for input or read from a terminal. Multi-line content SHALL be accepted via `--content-file <path>` or `--content-file -` (piped stdin). Any command that lacks required input SHALL fail with a usage error rather than prompting.

#### Scenario: Content supplied via stdin pipe
- **WHEN** `particulars claim assert --subject par_x --content-file -` is run with content piped to stdin
- **THEN** the piped bytes are used as the claim `content` and the command completes without prompting

#### Scenario: Missing required flag
- **WHEN** `particulars claim assert --subject par_x` is run with neither `--content` nor `--content-file`
- **THEN** the command exits with code 2 and prints a usage error to stderr

### Requirement: JSON output mode
Every verb SHALL accept a `--json` flag. In JSON mode the command SHALL write exactly one JSON object to stdout on success, and on failure SHALL write a JSON object of the form `{"error": {"code": <string>, "message": <string>}}` to stderr and nothing to stdout. The JSON shape of each verb is the stable machine contract; text output is informational.

#### Scenario: Successful JSON output
- **WHEN** any verb is run with `--json` and succeeds
- **THEN** stdout contains a single parseable JSON object and stderr is empty

#### Scenario: Failure in JSON mode
- **WHEN** any verb is run with `--json` and fails
- **THEN** stdout is empty and stderr contains a single JSON object with an `error.code` and `error.message`

### Requirement: Exit codes
The CLI SHALL use the following exit codes: `0` success; `1` runtime error; `2` usage error; `3` not found (an ID, subject, or query that resolves to nothing); `4` check failed (`validate`, `index --check`, `conflicts --fail-on-conflicts`); `5` no workspace found.

#### Scenario: Unknown ID
- **WHEN** `particulars lineage clm_does-not-exist` is run
- **THEN** the command exits with code 3

#### Scenario: No workspace
- **WHEN** any workspace-requiring verb is run from a directory with no `dkf.yaml` in it or any ancestor and no `--workspace`/`DKF_WORKSPACE` set
- **THEN** the command exits with code 5

### Requirement: Workspace selection
Every workspace-requiring verb SHALL accept `--workspace <dir>`. Selection precedence SHALL be: `--workspace` flag, then `DKF_WORKSPACE` environment variable, then upward discovery from the current directory.

#### Scenario: Flag overrides environment
- **WHEN** `DKF_WORKSPACE=/a` is set and `--workspace /b` is passed
- **THEN** the command operates on `/b`

### Requirement: Provenance defaults from environment
The CLI SHALL resolve `author`, `harness`, and `model` for provenance fields with precedence: explicit flag, then `DKF_AUTHOR` / `DKF_HARNESS` / `DKF_MODEL` environment variables, then `dkf.yaml` `defaults.source`.

#### Scenario: Harness from environment
- **WHEN** `DKF_HARNESS=claude` is set and `claim assert` is run without `--harness`
- **THEN** the written claim has `source.harness: claude`

### Requirement: Version verb
The CLI SHALL provide `particulars version` reporting the binary version and the supported DKF format version (`dkf/0.1`).

#### Scenario: Version output
- **WHEN** `particulars version --json` is run
- **THEN** stdout contains an object with `version` and `format: "dkf/0.1"`
