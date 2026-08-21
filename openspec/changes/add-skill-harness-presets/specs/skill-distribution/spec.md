## MODIFIED Requirements

### Requirement: Install the skill
`particulars skill install [--harness <preset>]... [--user | --dir <path>] [--file <path>] [--force] [--check]` SHALL write the rendered skill for each selected preset (default `claude`), creating directories as needed and writing atomically. Preset targets are: `claude` → `./.claude/skills/particulars/SKILL.md` (`--user`: `$HOME/.claude/skills/particulars/SKILL.md`); `copilot` → `./.github/skills/particulars/SKILL.md` (`--user`: `$HOME/.copilot/skills/particulars/SKILL.md`); `agents` → `./.agents/skills/particulars/SKILL.md` (`--user`: `$HOME/.agents/skills/particulars/SKILL.md`); `cursor` → `./.cursor/rules/particulars.mdc`; `agents-md` → a bounded section in `./AGENTS.md` or the file named by `--file`. `--dir` writes `<path>/SKILL.md` and SHALL NOT be combined with `--harness` or `--user`; `--user` with `cursor` or `agents-md`, and `--file` with any preset other than `agents-md`, SHALL be usage errors. The result SHALL list, per target, the absolute `path`, the `harness`, the `version`, and exactly one of `created`, `updated`, or `unchanged` set to true. It SHALL NOT require a workspace.

#### Scenario: Fresh project install
- **WHEN** `skill install --json` is run in a directory with no `.claude/skills/particulars/SKILL.md`
- **THEN** the file is created, its content equals `skill show`, and the result has `created: true`

#### Scenario: User install
- **WHEN** `skill install --user` is run
- **THEN** `$HOME/.claude/skills/particulars/SKILL.md` is written

#### Scenario: Copilot preset
- **WHEN** `skill install --harness copilot` is run
- **THEN** `./.github/skills/particulars/SKILL.md` is written with content identical to the `claude` preset's

#### Scenario: Neutral skills location, user scope
- **WHEN** `skill install --harness agents --user` is run
- **THEN** `$HOME/.agents/skills/particulars/SKILL.md` is written

#### Scenario: Several presets at once
- **WHEN** `skill install --harness claude --harness cursor --json` is run
- **THEN** both targets are written and the result lists two entries

#### Scenario: Explicit directory
- **WHEN** `skill install --dir ./tmp/skills` is run
- **THEN** `./tmp/skills/SKILL.md` is written

#### Scenario: Conflicting flags
- **WHEN** `skill install --user --dir x` is run, or `skill install --harness cursor --user`, or `skill install --file AGENTS.md` without `--harness agents-md`
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Reinstall is idempotent
- **WHEN** `skill install` is run twice with the same binary
- **THEN** the second run reports `unchanged: true` and does not modify the file

### Requirement: Check mode for CI
With `--check`, `skill install` SHALL NOT write. For each selected target it SHALL compare what it owns — the whole file for SKILL.md and `.mdc` targets, the marker-bounded section for `agents-md` — with the rendered variant, ignoring the stamped version, and exit 0 when every target matches, or exit 4 when any target is missing, differs, or (for file targets) lacks the marker. With `--json` the result SHALL list per target `{"path", "harness", "status": "ok" | "missing" | "differs" | "foreign"}`. For `agents-md`, a file without the section SHALL report `missing`, not `foreign`.

#### Scenario: Up to date
- **WHEN** the target was written by `skill install` from a binary whose body matches and `skill install --check` is run
- **THEN** the command exits 0 with `status: ok`, even if the stamped versions differ

#### Scenario: Drift
- **WHEN** the target's body differs from the embedded skill and `skill install --check` is run
- **THEN** the command exits with code 4 and `status: differs`, and the target is unchanged

#### Scenario: Missing
- **WHEN** no target exists and `skill install --check` is run
- **THEN** the command exits with code 4 and `status: missing`

#### Scenario: AGENTS.md without a section
- **WHEN** `AGENTS.md` exists with user content but no particulars section and `skill install --harness agents-md --check` is run
- **THEN** the result has `status: missing` and the exit code is 4

## ADDED Requirements

### Requirement: Cursor rule rendering
For the `cursor` preset the CLI SHALL render the skill as a Cursor rule: frontmatter containing `description` (the skill's description, verbatim) and `alwaysApply: false`, then the ownership marker line naming the preset, then the skill body unchanged. The file extension SHALL be `.mdc`.

#### Scenario: Cursor rule shape
- **WHEN** `skill show --harness cursor` is run
- **THEN** the output starts with a frontmatter block whose keys are exactly `description` and `alwaysApply`, followed by a line matching `<!-- installed by particulars …; regenerate with: particulars skill install --harness cursor -->`, followed by the body that begins `You are the author of a knowledge base`

### Requirement: AGENTS.md section
For the `agents-md` preset the CLI SHALL own only a region bounded by `<!-- particulars:skill:start … -->` and `<!-- particulars:skill:end -->` containing a `## particulars — capturing knowledge` heading and the skill body with every Markdown heading (outside fenced code) demoted one level. Install SHALL create the file with only the section when absent, append the section when the file exists without markers, and replace exactly the bounded region when markers exist. Content outside the markers SHALL be byte-identical before and after. A start marker without an end marker SHALL be refused with exit code 1.

#### Scenario: Fresh AGENTS.md
- **WHEN** `skill install --harness agents-md` is run with no `AGENTS.md`
- **THEN** `AGENTS.md` is created containing only the bounded section

#### Scenario: Existing content preserved
- **WHEN** `AGENTS.md` has user text above and below a previously installed section and `skill install --harness agents-md` is run with a newer binary
- **THEN** only the bounded region changes and the text above and below is byte-identical

#### Scenario: Appended to a file without markers
- **WHEN** `AGENTS.md` exists with user content and no markers
- **THEN** the section is appended after a blank line and the existing content is unchanged

#### Scenario: Broken markers refused
- **WHEN** `AGENTS.md` contains a start marker but no end marker
- **THEN** the command exits with code 1 and the file is unchanged

#### Scenario: Retarget with --file
- **WHEN** `skill install --harness agents-md --file GEMINI.md` is run
- **THEN** the section is written to `GEMINI.md`

### Requirement: Copilot duplicate-location warning
When installing the `copilot` or `agents` preset (or `claude` when another Copilot-readable location already holds a particulars skill), the CLI SHALL still install but SHALL emit a warning naming the other location(s) among `.claude/skills/particulars/SKILL.md`, `.github/skills/particulars/SKILL.md`, and `.agents/skills/particulars/SKILL.md` that already exist, because GitHub Copilot loads all three and would load the skill more than once.

#### Scenario: Second Copilot-readable location
- **WHEN** `.claude/skills/particulars/SKILL.md` exists and `skill install --harness copilot --json` is run
- **THEN** the skill is written to `.github/skills/particulars/SKILL.md` and the result's `warnings` names `.claude/skills/particulars/SKILL.md`

### Requirement: Show a harness variant
`particulars skill show --harness <preset>` SHALL print exactly what `install` would write for that preset (for `agents-md`, the section only); with `--json`, `{"version", "harness", "content"}`.

#### Scenario: Show the AGENTS.md section
- **WHEN** `skill show --harness agents-md` is run
- **THEN** stdout begins with the start marker and ends with the end marker
