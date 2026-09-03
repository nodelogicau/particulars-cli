## MODIFIED Requirements

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
