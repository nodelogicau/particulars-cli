## ADDED Requirements

### Requirement: The skill is embedded in the binary
The binary SHALL embed `skills/particulars/SKILL.md` at build time. The embedded text, rendered for output, SHALL have its frontmatter `metadata.version` set to the binary version (without a leading `v`) and SHALL carry, immediately after the frontmatter's closing `---`, a marker line of the form `<!-- installed by particulars <version>; regenerate with: particulars skill install -->`. The rendered body SHALL otherwise be byte-identical to the repository file.

#### Scenario: Version stamped
- **WHEN** a binary built as version `0.2.1` renders the skill
- **THEN** the frontmatter contains `version: "0.2.1"` and the line after the frontmatter is `<!-- installed by particulars 0.2.1; regenerate with: particulars skill install -->`

#### Scenario: Body unchanged
- **WHEN** the rendered skill is compared with `skills/particulars/SKILL.md` after removing the marker and masking the version line
- **THEN** the two are identical

### Requirement: Show the skill
`particulars skill show` SHALL print the rendered skill to stdout. With `--json` it SHALL emit `{"version": <string>, "content": <string>}`. It SHALL NOT require a workspace.

#### Scenario: Plain show
- **WHEN** `particulars skill show` is run outside any workspace
- **THEN** stdout begins with `---` and contains `name: particulars`, and the exit code is 0

#### Scenario: JSON show
- **WHEN** `particulars skill show --json` is run
- **THEN** the object's `content` equals the plain output and `version` equals the binary version

### Requirement: Install the skill
`particulars skill install [--user | --dir <path>] [--force] [--check]` SHALL write the rendered skill to `SKILL.md` in `./.claude/skills/particulars/` by default, `$HOME/.claude/skills/particulars/` with `--user`, or `<path>/` with `--dir`, creating directories as needed and writing atomically. `--user` and `--dir` together SHALL be a usage error. The result SHALL include the absolute `path`, the `version`, and exactly one of `created`, `updated`, or `unchanged` set to true. It SHALL NOT require a workspace.

#### Scenario: Fresh project install
- **WHEN** `skill install --json` is run in a directory with no `.claude/skills/particulars/SKILL.md`
- **THEN** the file is created, its content equals `skill show`, and the result has `created: true`

#### Scenario: User install
- **WHEN** `skill install --user` is run
- **THEN** `$HOME/.claude/skills/particulars/SKILL.md` is written

#### Scenario: Explicit directory
- **WHEN** `skill install --dir ./tmp/skills` is run
- **THEN** `./tmp/skills/SKILL.md` is written

#### Scenario: Conflicting flags
- **WHEN** `skill install --user --dir x` is run
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Reinstall is idempotent
- **WHEN** `skill install` is run twice with the same binary
- **THEN** the second run reports `unchanged: true` and does not modify the file

### Requirement: Never overwrite a foreign skill file
If the target exists and does not contain the marker line, `skill install` SHALL refuse with exit code 1 and an error naming the path, unless `--force` is given, in which case it SHALL overwrite and report `updated: true`. A target that contains the marker SHALL be overwritten without `--force` when its content differs.

#### Scenario: Hand-written file protected
- **WHEN** `.claude/skills/particulars/SKILL.md` exists with content that lacks the marker and `skill install` is run
- **THEN** the command exits with code 1, the file is unchanged, and the message mentions `--force`

#### Scenario: Force overwrite
- **WHEN** the same situation occurs and `skill install --force` is run
- **THEN** the file is replaced and the result has `updated: true`

#### Scenario: Own file updated
- **WHEN** the target carries the marker from an older version and `skill install` is run with a newer binary
- **THEN** the file is replaced and the result has `updated: true`

### Requirement: Check mode for CI
With `--check`, `skill install` SHALL NOT write. It SHALL compare the target with the rendered skill, ignoring the version in the frontmatter and marker, and exit 0 when they match, or exit 4 when the target is missing, differs in body, or lacks the marker. With `--json` the result SHALL be `{"path": <abs>, "status": "ok" | "missing" | "differs" | "foreign"}`.

#### Scenario: Up to date
- **WHEN** the target was written by `skill install` from a binary whose body matches and `skill install --check` is run
- **THEN** the command exits 0 with `status: ok`, even if the stamped versions differ

#### Scenario: Drift
- **WHEN** the target's body differs from the embedded skill and `skill install --check` is run
- **THEN** the command exits with code 4 and `status: differs`, and the target is unchanged

#### Scenario: Missing
- **WHEN** no target exists and `skill install --check` is run
- **THEN** the command exits with code 4 and `status: missing`
