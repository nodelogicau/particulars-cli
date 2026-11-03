# plugin-package Specification

## Purpose
TBD - created by archiving change chatgpt-plugin-package. Update Purpose after archive.
## Requirements
### Requirement: Plugin package contents
Each release SHALL attach `particulars-<version>-plugin.zip`, an Agent Plugins package whose root contains:
- `plugin.json`;
- `skills/particulars/SKILL.md`;
- `skills/setup/SKILL.md`;
- `assets/icon.png`.

The package SHALL NOT contain `mcp.json`, a `.codex-plugin/` directory, or any executable binary. `checksums.txt` SHALL list the zip.

#### Scenario: Release asset
- **WHEN** a tag is released
- **THEN** the release includes `particulars-<version>-plugin.zip` and `checksums.txt` lists it

#### Scenario: Skills-only
- **WHEN** the zip is listed
- **THEN** it contains the four files above and no `mcp.json`, `.codex-plugin/`, or binary

### Requirement: Plugin manifest
`plugin.json` SHALL be JSON with:
- `$schema` equal to `https://agent-plugins.org/schemas/1.0.0/plugin.schema.json`;
- `name` equal to `particulars`;
- `version` equal to the release version without a leading `v`;
- a non-empty `description`;
- `license` equal to `MIT`;
- `homepage` pointing at the repository.

It SHALL carry `extensions["com.openai"]` with:
- `requires_local_executor: true`;
- `onboardingSkill: "./skills/setup/SKILL.md"`;
- an `interface` object with `displayName`, `shortDescription`, and `logo` set to `./assets/icon.png`.

Every path in the manifest SHALL start with `./` and resolve to a file inside the package.

#### Scenario: Version and flags
- **WHEN** the package for release `v0.20.0` is built
- **THEN** `plugin.json` parses, `version` is `0.20.0`, and `extensions["com.openai"].requires_local_executor` is `true`

#### Scenario: Paths resolve
- **WHEN** each manifest path is resolved against the package root
- **THEN** every referenced file exists in the zip

### Requirement: Main skill matches the binary
`skills/particulars/SKILL.md` in the package SHALL be byte-identical to the output of `particulars skill show` from the binary built for the same release. That means it carries the release's version stamp and ownership marker, and its body is unchanged from `skills/particulars/SKILL.md` in the repository.

#### Scenario: Identical to skill show
- **WHEN** the packaged `skills/particulars/SKILL.md` is compared with `particulars skill show` from the same release
- **THEN** the two are identical

### Requirement: Onboarding skill
`skills/setup/SKILL.md` SHALL have frontmatter with `name: setup` and a `description`. Its instructions SHALL direct the agent to:
- check for the binary with `particulars version`. When it is missing, give install instructions for the user's OS: Homebrew or `install.sh` on macOS and Linux, the release archive on Windows. Do not install anything without the user's agreement.
- check for a workspace with `particulars workspace --json`. When none resolves, offer `particulars init <dir> --pointer`, `particulars workspace pointer <dir>`, or `DKF_WORKSPACE`.
- recommend recording the harness via `DKF_HARNESS` or `--harness`.

It SHALL NOT instruct the agent to edit workspace YAML files.

#### Scenario: Binary missing
- **WHEN** the setup skill runs and `particulars version` fails
- **THEN** the skill's instructions lead the agent to give install instructions for the user's OS and not to continue to workspace setup

#### Scenario: No workspace
- **WHEN** the binary is present but `particulars workspace --json` resolves no workspace
- **THEN** the skill's instructions lead the agent to offer `init --pointer`, `workspace pointer`, or `DKF_WORKSPACE`

### Requirement: Build target
`make plugin` SHALL produce `dist/particulars-<version>-plugin.zip` from `plugin/plugin.json.tmpl`, `plugin/skills/setup/SKILL.md`, `bundle/icon.png`, and the skill rendered by the binary. The build SHALL fail with a message, and SHALL NOT write a zip, when:
- the manifest does not parse;
- `version` differs from the build version;
- `requires_local_executor` is not `true`;
- a manifest path does not resolve;
- either skill lacks `name` or `description` frontmatter.

The release process SHALL run the same build.

#### Scenario: Local build
- **WHEN** `make plugin VERSION=v0.20.0` runs
- **THEN** `dist/particulars-0.20.0-plugin.zip` exists and satisfies the manifest and contents requirements

#### Scenario: Broken template
- **WHEN** `plugin/plugin.json.tmpl` references an onboarding skill path that does not exist
- **THEN** `make plugin` exits non-zero, names the missing path, and leaves no zip

### Requirement: Plugin documentation
`docs/plugin.md` SHALL explain:
- installing the package in ChatGPT desktop (upload and workspace publishing);
- installing it in Codex from a local path;
- that the binary must be installed separately;
- how the workspace is found in a local thread;
- that the plugin and `skill install` should not both be used for the same harness.

The README SHALL link to it from the section on connecting harnesses.

#### Scenario: README link
- **WHEN** a reader looks for ChatGPT in the README
- **THEN** they find a link to `docs/plugin.md`

