## Context

`skills/particulars/SKILL.md` is the agent's manual for the CLI. It is installed by hand (`cp` into `.claude/skills/particulars/`), duplicated in this repo for dogfooding, and versioned by a hand-edited frontmatter field. The friction surfaced when using the CLI from Claude Desktop remote sessions: each session needs the binary *and* the skill, from different places, with nothing guaranteeing they match.

Constraints carried over: non-interactive, `--json` on every verb, documented exit codes, no new dependencies in the core, nothing in `internal/dkf` changes.

## Goals / Non-Goals

**Goals:**

- One artifact: the binary carries the skill; `particulars skill install` is the whole setup.
- Skill and verbs cannot drift: the embedded text is what the binary ships, stamped with its version.
- Safe overwrite: never clobber a skill file the tool did not write.
- CI-checkable: this repo (and any consumer) can assert the installed skill matches the binary.

**Non-Goals:**

- Knowing other harnesses' skill directories (only Claude Code's project/user paths plus `--dir`).
- Templating the skill per workspace (e.g. injecting `base-uri`); the skill stays generic.
- Auto-installing on every verb, or writing into a workspace directory.

## Decisions

### D1. Embed via a Go package living next to the file

**Choice:** add `skills/particulars/skill.go` (package `particularsskill`) containing `//go:embed SKILL.md` and `func Raw() []byte`. `internal/cli` imports it.

**Why:** `go:embed` cannot reference `..`, so embedding from `internal/cli` would force the file to move. A package co-located with the file keeps `skills/particulars/SKILL.md` as the single canonical, human-browsable copy and still lets the binary ship it. The package has no other code, so it never needs `internal/`.

**Alternative:** `go generate` copying into `internal/` — reintroduces a second copy and a drift window.

### D2. Stamp at render time, not build time

**Choice:** `Render(version string) []byte` rewrites the frontmatter line `  version: "…"` under `metadata:` to the binary version (without a leading `v`) and inserts, immediately after the closing `---`, the line
`<!-- installed by particulars <version>; regenerate with: particulars skill install -->`.
The canonical file in the repo keeps a placeholder `version: "dev"` so it does not have to be edited per release.

**Why:** the binary version already flows in through `-ldflags`; rendering at runtime means no generated files and the committed skill never needs a version bump. The marker doubles as the ownership signal for D4 and as a visible hint to a reviewer reading the installed file.

### D3. Verb shape

```
particulars skill show                      # stdout; --json → {version, content}
particulars skill install                   # ./.claude/skills/particulars/SKILL.md
particulars skill install --user            # $HOME/.claude/skills/particulars/SKILL.md
particulars skill install --dir <path>      # <path>/SKILL.md
particulars skill install [...] --check     # no write; exit 4 if missing or different
particulars skill install [...] --force     # overwrite a foreign file
```

`--user` and `--dir` are mutually exclusive (usage error). Neither subcommand opens a workspace; like `init` and `version` they work anywhere. Paths in JSON output are absolute.

**Why project default:** skills are per repository in Claude Code, and the agent's cwd is the repo. `--user` exists for people who use the CLI across many repos.

### D4. Overwrite rules

- Target absent → write, result `created: true`.
- Target present and contains the marker → overwrite, result `updated: true` (or `unchanged: true` when bytes are identical — no write, so timestamps and git status stay quiet).
- Target present without the marker → refuse with exit 1 (`conflict`) naming the path, unless `--force`.
- Writes are atomic (temp + rename), directories created as needed.

**Why:** a hand-written skill in the same path is somebody's deliberate customisation; the tool must not silently replace it. The marker is cheap and survives a reviewer's read.

### D5. `--check` for CI

**Choice:** `--check` renders the skill, compares byte-for-byte with the target, and exits 4 on any difference (including absence), printing a one-line reason; `--json` gives `{path, status: ok|missing|differs|foreign}`. It never writes.

**Why:** this repo dogfoods the installed copy; a CI step `particulars skill install --check` proves it matches the binary, replacing the hand-run flag audit. Consumers can use the same step in their workspace repos.

### D6. Dogfooding in this repo

`.claude/skills/particulars/SKILL.md` is regenerated with `make skill` (which builds then runs `skill install`) and checked in. CI runs `skill install --check` after build. The `version` stamped in the committed copy will read `dev` when built without a tag (the Makefile's `git describe`), so the check in CI compares content after normalising the stamp — simplest is for `--check` to ignore the version in the marker and frontmatter when comparing (it compares the skill *body*). Decision: `--check` compares with the version lines masked; `status: differs` only for real content drift.

### D7. SessionStart hook (docs only)

Ship `docs/examples/claude-settings.json`:

```json
{
  "hooks": {
    "SessionStart": [{
      "hooks": [{
        "type": "command",
        "command": "command -v particulars >/dev/null || curl -sSL https://raw.githubusercontent.com/nodelogicau/particulars-cli/main/install.sh | sh; particulars skill install --json >/dev/null"
      }]
    }]
  }
}
```

The `install.sh` referenced there does not exist yet; this change ships the hook example with a TODO pointing at the Homebrew-tap/installer follow-up, and the README shows the manual two-liner (download release, `skill install`) as the supported path today.

**Why:** it is the piece that makes remote sessions zero-setup, but the installer is a separate deliverable; documenting the hook shape now lets you wire it as soon as the installer exists.

## Risks / Trade-offs

- [Frontmatter rewrite is regex-based] → the skill's frontmatter is ours and small; a unit test pins the exact lines. If the frontmatter ever gains YAML complexity, switch to a YAML round-trip.
- [Marker is an HTML comment; a harness could render it] → Claude Code ignores HTML comments in skill bodies; `show` keeps it too so the installed and shown text are identical.
- [Masked-version comparison in `--check` could hide a genuine version-only change] → acceptable: version-only changes carry no behavioural content; the body comparison is what matters.
- [`--user` path hard-codes `~/.claude`] → documented; `--dir` covers everything else.

## Migration Plan

Users with a hand-copied `.claude/skills/particulars/SKILL.md` (no marker) will be refused once and must pass `--force` the first time; the README says so. No format or workspace impact. Release v0.2.1.

## Open Questions

- Should `skill show --json` also include the list of verbs the binary actually has (from cobra), so a harness can detect skill/verb mismatch at runtime? Cheap and tempting; deferred until there is a consumer that would read it.
- Whether to also install a short `CLAUDE.md` snippet pointing at the skill. Not now — the skill's frontmatter description already triggers it.
