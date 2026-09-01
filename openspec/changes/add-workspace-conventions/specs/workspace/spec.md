## ADDED Requirements

### Requirement: Workspace conventions file
A workspace MAY carry a conventions document for agents: `CONVENTIONS.md` at the workspace root by default, or the file named by `workspace.conventions` in `dkf.yaml`. The key SHALL be a relative path that resolves inside the workspace; an absolute path or one escaping the root SHALL fail config validation. `particulars workspace` SHALL report the resolved file when one applies — in JSON as `conventions` — and SHALL report a configured file that does not exist as `conventions` with `conventions_missing: true`. A workspace with neither key nor default file SHALL report nothing.

#### Scenario: Default conventions file
- **WHEN** `CONVENTIONS.md` exists at the workspace root and `particulars workspace --json` is run
- **THEN** the result has `conventions: "CONVENTIONS.md"`

#### Scenario: Configured conventions file
- **WHEN** `dkf.yaml` has `workspace.conventions: TOPICS.md` and that file exists
- **THEN** the result has `conventions: "TOPICS.md"`

#### Scenario: Configured but missing
- **WHEN** `workspace.conventions` names a file that does not exist
- **THEN** the result has `conventions` and `conventions_missing: true`, and the command exits 0

#### Scenario: Escaping path refused
- **WHEN** `dkf.yaml` has `workspace.conventions: ../secrets.md`
- **THEN** the workspace fails to open with a config error
