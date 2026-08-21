## 1. Embed and Render

- [ ] 1.1 Add `skills/particulars/skill.go` (package `particularsskill`) with `//go:embed SKILL.md`, `Raw() []byte`, and `Render(version string) []byte` that stamps `metadata.version` and inserts the marker after the frontmatter; set the committed frontmatter version to `"dev"`
- [ ] 1.2 Add `Marker(version)`, `HasMarker(data) bool`, and `BodyEqual(a, b []byte) bool` (compares with version lines and marker masked)
- [ ] 1.3 Unit tests: stamping, marker placement, body equality with the repository file, mask-compare tolerates version-only differences

## 2. CLI Verb

- [ ] 2.1 `internal/cli/cmd_skill.go`: parent `skill` command; `show` (text + `--json {version, content}`); neither subcommand opens a workspace
- [ ] 2.2 `install`: target resolution (`default` → `./.claude/skills/particulars/SKILL.md`, `--user` → `$HOME/…`, `--dir`), mutual-exclusion usage error, atomic write with directory creation, `created|updated|unchanged` result with absolute `path`
- [ ] 2.3 Overwrite rules: foreign file → exit 1 naming the path and `--force`; marker present → overwrite when different; identical → no write
- [ ] 2.4 `--check`: no write; `ok|missing|differs|foreign` with exit 0/4; text one-liner
- [ ] 2.5 Register under root; root help lists `skill`
- [ ] 2.6 In-process CLI tests covering every scenario in `specs/skill-distribution` (use `t.Setenv("HOME", …)` and a temp cwd)

## 3. Dogfood and CI

- [ ] 3.1 `Makefile`: `make skill` builds then runs `dist/particulars skill install` at the repo root; regenerate `.claude/skills/particulars/SKILL.md` and commit the marker-bearing copy
- [ ] 3.2 `.github/workflows/ci.yml`: after build, run `dist/particulars skill install --check`
- [ ] 3.3 Remove the hand-run flag audit from the workflow notes; the check replaces it

## 4. Documentation and Release

- [ ] 4.1 README "Teaching an agent the verbs": `particulars skill install` (project/user/dir), `--force` for previously hand-copied files, `skill show` for other harnesses
- [ ] 4.2 `docs/examples/claude-settings.json` with the SessionStart hook shape; README note that the installer it references is a follow-up and the supported path today is download + `skill install`
- [ ] 4.3 `docs/review-workflow.md` agent-side setup line: `particulars skill install`
- [ ] 4.4 `CHANGELOG.md` v0.2.1 entry; tag `v0.2.1`; verify the published binary's `skill show` carries `version: "0.2.1"`
