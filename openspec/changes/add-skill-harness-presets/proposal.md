## Why

`skill install` knows one layout — Claude Code's `.claude/skills/` — plus a raw `--dir`. Other harnesses read different shapes: Cursor wants a `.cursor/rules/*.mdc` file with its own frontmatter, the `AGENTS.md` convention (Codex, Jules, Gemini CLI, Cursor, Copilot and ~20 others) wants a *section* in one shared file, and the Agent Skills format has a vendor-neutral home at `.agents/skills/`. Today a multi-harness repository has to hand-convert the skill for each, and can accidentally install duplicates (GitHub Copilot reads `.github/skills`, `.claude/skills`, *and* `.agents/skills`, so installing to two of them loads the skill twice). Presets make "install for my harness" one flag, with the same version stamp, ownership marker, and `--check` everywhere.

## What Changes

- `skill install --harness <preset>` (repeatable) with presets:
  - `claude` (default) → `./.claude/skills/particulars/SKILL.md`; `--user` → `~/.claude/skills/particulars/SKILL.md`. Also read by GitHub Copilot.
  - `copilot` → `./.github/skills/particulars/SKILL.md`; `--user` → `~/.copilot/skills/particulars/SKILL.md`.
  - `agents` → `./.agents/skills/particulars/SKILL.md`; `--user` → `~/.agents/skills/particulars/SKILL.md` — the vendor-neutral Agent Skills location.
  - `cursor` → `./.cursor/rules/particulars.mdc`: the same body under Cursor frontmatter (`description`, `alwaysApply: false`); no `--user` variant.
  - `agents-md` → a marker-bounded section in `./AGENTS.md` (created if absent; replaced in place if present; everything outside the markers untouched); `--file <path>` retargets it (e.g. `GEMINI.md`); no `--user` variant.
- `skill show --harness <preset>` prints the rendered variant, so any harness can be fed by hand.
- `--check` works per preset; for `agents-md` it checks only the bounded section.
- Documentation warns that Copilot reads three skills directories and recommends installing to exactly one.
- README harness table; CHANGELOG v0.3.1.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `skill-distribution`: `install`/`show` gain `--harness` presets with per-harness rendering (SKILL.md, Cursor `.mdc`, AGENTS.md section) and per-preset check semantics.

## Impact

- `skills/particulars`: `Body()`, `RenderCursorRule(version)`, `RenderAgentsSection(version)`, section markers and splice helpers.
- `internal/cli/cmd_skill.go`: preset table, `--harness`, `--file`, multi-target loop, section-aware install/check.
- README, CHANGELOG, the skill's own "Setup" paragraph (so an agent knows the presets exist).
- No new dependencies; no workspace or format impact.
- Not in scope: `--all` (would duplicate for Copilot); harness-specific prompt files (`.github/prompts`), Windsurf/Zed-specific paths beyond `--dir`; a short "pointer" variant for AGENTS.md (see design open question).
