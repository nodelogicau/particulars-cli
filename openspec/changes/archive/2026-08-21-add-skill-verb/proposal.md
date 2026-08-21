## Why

The agent-facing skill (`skills/particulars/SKILL.md`) is the only way an agent learns the verbs, yet the binary knows nothing about it: every fresh session — and every remote sandbox — has to fetch the binary from one place and copy the skill from another, and the two can drift (the skill's flag list is only checked against the binary by hand). Shipping the skill inside the binary and adding a `skill` verb makes installation one command, keeps skill and verbs in lockstep by construction, and removes the duplicated copy this repo keeps under `.claude/skills/`.

## What Changes

- Embed `skills/particulars/SKILL.md` in the binary at build time (a Go package co-located with the file, so the human-browsable path stays canonical).
- New verb `particulars skill` with subcommands:
  - `show` — print the embedded skill (with `--json`: `{version, content}`), for any harness that can pipe a file.
  - `install [--user | --dir <path>] [--force] [--check]` — write `SKILL.md` to `./.claude/skills/particulars/` (project, default), `~/.claude/skills/particulars/` (`--user`), or an explicit directory; refuse to overwrite a file it did not write unless `--force`; `--check` verifies without writing (exit 4 on drift) for CI.
- The installed/shown text is stamped with the binary version (frontmatter `metadata.version` and an HTML-comment marker after the frontmatter) so `install` can recognise its own output and reviewers can see which CLI a skill describes.
- This repo's `.claude/skills/particulars/SKILL.md` becomes the output of `particulars skill install`, verified in CI with `skill install --check`, instead of a hand-maintained second copy.
- Docs: README install flow (`particulars skill install`), a sample Claude Code `SessionStart` hook that installs the binary if missing and runs `skill install`, so remote sessions need zero manual setup.
- Release as v0.2.1.

## Capabilities

### New Capabilities
- `skill-distribution`: Embedding the agent skill in the binary, the `skill show` / `skill install` verbs, version stamping, ownership marker and overwrite rules, and the `--check` drift mode.

### Modified Capabilities
<!-- none: adding a verb does not change existing requirements -->

## Impact

- New package `skills/particulars` (Go file + `//go:embed SKILL.md`) imported by `internal/cli`; no new third-party dependencies.
- `internal/cli`: new `cmd_skill.go`; root command registration; in-process tests.
- `.claude/skills/particulars/SKILL.md` regenerated (content identical plus marker); `.github/workflows/ci.yml` gains a `skill install --check` step; `Makefile` gains `make skill`.
- README, `docs/review-workflow.md` (setup line), new `docs/examples/claude-settings.json`.
- Not in scope: other harnesses' skill locations beyond `--dir`; an MCP transport; the root-level workspace discovery pointer (separate change).
