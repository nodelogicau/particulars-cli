## 1. Rendering (`skills/particulars`)

- [ ] 1.1 `Body()` (skill minus frontmatter), `Description()` (frontmatter description), `RenderCursorRule(version)`, `RenderAgentsSection(version)` with heading demotion outside fenced code
- [ ] 1.2 Section markers: `SectionStart(version)`, `SectionEnd`, `FindSection(data) (start, end int, ok, broken bool)`, `SpliceSection(data, section) ([]byte, error)`
- [ ] 1.3 Extend `HasMarker`/`Mask`/`BodyEqual` to both marker shapes and the `.mdc` preset hint
- [ ] 1.4 Unit tests: cursor frontmatter shape; demotion leaves fenced `#` untouched and demotes every heading level; splice create/append/replace with byte-identical surroundings; broken markers; mask tolerates version-only differences across all three renders

## 2. CLI

- [ ] 2.1 Preset table (`claude`, `copilot`, `agents`, `cursor`, `agents-md`) with project/user paths and kind; `--harness` (repeatable), `--file`; validation of flag combinations per spec
- [ ] 2.2 `install`: loop over targets; file-kind write/overwrite rules unchanged; section-kind splice; per-target result entries; `targets` array in JSON (single-target output stays backward compatible by also including the top-level fields)
- [ ] 2.3 `--check` per target, section-aware; overall exit 4 if any target is not `ok`
- [ ] 2.4 Copilot duplicate-location warning (text + `warnings` in JSON)
- [ ] 2.5 `show --harness`
- [ ] 2.6 In-process CLI tests for every scenario in the delta spec, including multi-preset install and the duplicate warning

## 3. Docs and Release

- [ ] 3.1 README: harness table (which preset for which tool, which locations Copilot reads, "pick one"), `show --harness` for unlisted harnesses, note that `--dir` into `.cursor/rules` does not work
- [ ] 3.2 Skill "Setup" paragraph mentions presets so an agent can install for its own harness; regenerate the repo's installed copy (`make skill`) and keep `skill install --check` green
- [ ] 3.3 `CHANGELOG.md` v0.3.1; tag `v0.3.1`; verify `brew upgrade` picks it up and `skill show --harness cursor` / `agents-md` render from the published binary
- [ ] 3.4 Record v0.3.1 in the knowledge workspace (branch), including the verified harness conventions as claims with the docs URLs as evidence
