## Context

v0.2.1 embedded the skill and added `skill install` with Claude Code's layout and a raw `--dir`. Harness conventions as verified on 2026-08-21:

| Harness | Project location | Personal location | Format |
|---|---|---|---|
| Claude Code | `.claude/skills/<name>/SKILL.md` | `~/.claude/skills/…` | SKILL.md (frontmatter `name`, `description`) |
| GitHub Copilot (cloud agent, code review, CLI, VS Code/JetBrains agent mode) | `.github/skills/`, `.claude/skills/`, or `.agents/skills/` | `~/.copilot/skills/` or `~/.agents/skills/` | SKILL.md |
| Cursor | `.cursor/rules/*.mdc`; also `AGENTS.md` | — | `.mdc` frontmatter: `description`, `globs`, `alwaysApply`; plain `.md` in that dir is ignored |
| AGENTS.md convention (Codex, Jules, Gemini CLI, Windsurf, Zed, Copilot, Cursor, …) | `AGENTS.md` at repo root, nested allowed | — | plain Markdown, no required headings |

Two facts shape the design: Copilot already reads our default location, and Copilot reads *three* locations, so "install everywhere" is actively harmful.

## Goals / Non-Goals

**Goals:**
- One flag per harness; identical skill body everywhere; version stamp, ownership marker, `--force` protection and `--check` preserved for every variant.
- Never corrupt a user's `AGENTS.md`: own only a bounded section.
- Make the duplicate-skill trap for Copilot hard to fall into.

**Non-Goals:**
- Harness-specific prompt/command files; per-harness tailoring of the skill's *content*; auto-detecting which harness is in use; an `--all` preset.

## Decisions

### D1. Presets are a table, not code paths

```go
type preset struct {
    name     string
    project  string // relative path from cwd
    user     string // relative to $HOME; "" = no user variant
    kind     kind   // skillFile | cursorRule | agentsSection
}
```

`claude`, `copilot`, `agents` are all `skillFile` with different paths; `cursor` is `cursorRule`; `agents-md` is `agentsSection`. `--harness` is repeatable; `--dir` remains the escape hatch (implies `skillFile`). `--user` with a preset that has no user variant is a usage error naming the preset. `--file` is only valid with `agents-md`.

**Why:** every future harness that adopts SKILL.md is a one-line table entry; only genuinely different formats need code.

### D2. Cursor rule rendering

`RenderCursorRule(version)` emits:

```
---
description: <the skill's frontmatter description, verbatim>
alwaysApply: false
---
<!-- installed by particulars <v>; regenerate with: particulars skill install --harness cursor -->
<skill body>
```

No `globs`: the skill is about when to capture knowledge, not which files are open; `alwaysApply: false` + `description` lets Cursor attach it intelligently, mirroring how Claude Code triggers on the description. The marker's "regenerate with" hint names the preset.

### D3. AGENTS.md section

`RenderAgentsSection(version)` emits:

```
<!-- particulars:skill:start — installed by particulars <v>; regenerate with: particulars skill install --harness agents-md -->
## particulars — capturing knowledge

<skill body with every heading demoted one level>

<!-- particulars:skill:end -->
```

Install splices: if the file is absent, create it with just the section; if present without markers, append the section after a blank line; if present with markers, replace only the bounded region. Anything outside the markers is never touched, and a file with a start marker but no end marker is refused (exit 1) rather than guessed at. `--check` compares the bounded region (version-masked); `foreign` means the file exists but has no markers — which for this preset is *not* an error state, so `--check` reports `missing` instead (the section is what we own, and it is absent).

**Why full text rather than a pointer:** `AGENTS.md` is loaded into every session; so is the skill when triggered. The whole point is that the discipline is in context *before* the agent acts, and a pointer ("run `particulars skill show`") defers that to a tool call the agent may not make. ~4 KB is acceptable. A `--brief` variant is left as an open question.

### D4. Ownership marker per kind

SKILL.md and `.mdc`: the existing single marker line (the `.mdc` one names its preset). AGENTS.md: start/end markers. `HasMarker`/`Mask` gain awareness of both marker shapes; `BodyEqual` continues to mask versions.

### D5. Copilot duplication guard

`install --harness copilot` (and `agents`) warns — text and a `warnings` array in JSON — when another Copilot-readable location already holds a particulars skill, naming it. It still installs (the user may be migrating); the warning is what prevents the silent double-load. README states the rule plainly: *pick one of `.claude/skills`, `.github/skills`, `.agents/skills`; `agents` is the neutral choice for multi-harness repos.*

### D6. `show --harness`

`skill show --harness cursor|agents-md|…` prints exactly what `install` would write (for `agents-md`, the section only), so users of unlisted harnesses can pipe it wherever they need.

## Risks / Trade-offs

- [Harness conventions move] → the table is data; verify paths in the README against docs at each release. The `.agents/skills` neutral location is newest and the one most likely to shift.
- [Heading demotion could mangle code blocks containing `#`] → demote only lines that start a Markdown heading outside fenced code; unit-test with the real skill.
- [Replacing a bounded section in a user-edited AGENTS.md] → only the region between our markers is replaced; a missing end marker aborts. Tests cover "user text above and below survives".
- [Cursor ignores `.md` in rules dir] → we write `.mdc`; `--dir` into `.cursor/rules` is documented as *not* working for Cursor.

## Migration Plan

Additive. Existing installs are the `claude` preset. v0.3.1.

## Open Questions

- `--brief` for `agents-md` (a two-paragraph summary plus "run `particulars skill show` for the full discipline") if the context cost proves to matter.
- Whether to detect the harness from the cwd (`.cursor/` present → suggest `cursor`). Easy, but guessing is the thing this tool avoids; a hint in the text output is probably enough.
