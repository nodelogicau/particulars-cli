## Context

particulars reaches agents in two ways today. The `.mcpb` bundle (`bundle/`, `scripts/build-bundle.sh`) runs the stdio MCP server inside Claude Desktop. `skill install` writes the embedded skill for shell harnesses, which then drive the CLI.

ChatGPT desktop and Codex share the portable Agent Plugins format:

```
plugin-root/
├── plugin.json      name, version, description, $schema, extensions["com.openai"]
├── mcp.json         optional (stdio | streamable-http | sse)
├── skills/<name>/SKILL.md
└── assets/
```

ChatGPT desktop runs plugins that need local execution only in threads that run locally, and marks them **Desktop only**. There are two ways to get there:

- a stdio `mcp.json`, which only allowlisted partners may submit;
- a skills-only package with `extensions.com.openai.requires_local_executor: true`, which goes through the standard flow.

Constraints:
- The Agent Plugins `mcp.json` has no per-platform `command`, and the manifest has no `user_config` form like `.mcpb`'s directory picker.
- `install.sh` covers Linux and macOS only. ChatGPT desktop runs on macOS and Windows.
- The skill text is canonical in `skills/particulars/SKILL.md`. Every rendered variant keeps its body byte-identical (skill-distribution spec).

## Goals / Non-Goals

**Goals:**
- One zip per release that ChatGPT desktop accepts by upload or workspace publishing, and that Codex can install from a local path.
- Reuse the embedded skill unchanged, so the package can't drift from `skill show`.
- On first use, guide the user to a working binary and workspace.

**Non-Goals:**
- Running the MCP server from the plugin.
- Shipping binaries.
- A Codex marketplace file.
- Public directory submission.
- A Windows install script.

## Decisions

### Skills-only, not stdio MCP
The package has no `mcp.json`.
- Local MCP submission is gated to allowlisted partners.
- The README already recommends the skill and CLI over the server for harnesses with a shell (about 86 tokens collapsed against about 4,900).
- The CLI resolves the workspace on every call rather than binding one at startup, which matters because the plugin has no folder picker.

*Alternative:* a stdio server with `command: particulars` on PATH. Rejected for now because of submission gating and context cost. It could be added later as a second package without changing this one.

### Binary on PATH, not bundled
The skill already declares `compatibility: Requires the particulars binary on PATH`.
- Bundling binaries would mean OS-specific paths in the skill text, which breaks byte-identity.
- It would make the zip about four times the binary size.
- It would leave macOS quarantine handling to the plugin host.

The setup skill closes the gap instead. It points to `brew install nodelogicau/tap/particulars` or `install.sh` on macOS and Linux, and to the release archive on Windows.

*Alternative:* `${PLUGIN_ROOT}/bin/<os>/particulars` plus a wrapper. Rejected for the reasons above.

### Source layout mirrors `bundle/`
- `plugin/plugin.json.tmpl` uses `__VERSION__` substitution, like `bundle/manifest.json.tmpl`.
- `plugin/skills/setup/SKILL.md` is the onboarding skill, kept in the repo as-is.
- The icon is reused from `bundle/icon.png`, so there is one icon asset.

### Render the main skill with the binary
`scripts/build-plugin.sh` writes `skills/particulars/SKILL.md` by running the freshly built binary's `skill show`. With no host binary in `dist/`, it falls back to `go run` with the version ldflag. Either way, the version stamp, the marker, and the body are exactly what `skill show` prints for that release. No second copy is committed.

*Alternative:* copy `skills/particulars/SKILL.md` raw. Rejected because it would carry `version: "dev"` and no marker.

### Build wiring
- `make plugin` runs the script against `$(VERSION)` and `$(DIST)`.
- GoReleaser runs it as a `before`/post-build hook alongside `build-bundle.sh`.
- `dist/*-plugin.zip` is added to `release.extra_files` and `checksum.extra_files`.
- The script validates its output before zipping, the way `build-bundle.sh` does: `plugin.json` parses, `name` is `particulars`, `version` matches, `requires_local_executor` is true, the `onboardingSkill` path exists, both skills have `name`/`description` frontmatter, and there is no `mcp.json`.

### Setup skill scope
The setup skill is a short procedure:
1. Run `particulars version`. If it fails, give the install instructions for the user's OS and stop.
2. Run `particulars workspace --json` to see whether a workspace resolves from the thread's directory.
3. If none resolves, offer `particulars init <dir> --pointer` for a new workspace, or `particulars workspace pointer <existing>`, and mention `DKF_WORKSPACE`.
4. Suggest `DKF_HARNESS=chatgpt` or `codex`, or passing `--harness`, so provenance records the right harness.

It never edits YAML and never installs software without the user agreeing.

## Risks / Trade-offs

- **Working directory for local ChatGPT threads.** Verified 2026-11-02 on ChatGPT desktop (macOS): a local thread runs in its folder, the setup skill wrote a `.dkf` pointer there to an existing workspace, and `particulars workspace` then resolved via that pointer. So cwd and `.dkf` discovery work as they do in Claude Code. `DKF_WORKSPACE` stays a fallback in the setup skill.
- **Agent Plugins schema is new (1.0.0) and OpenAI's overlay keys may change.** → Keep the manifest minimal, pin `$schema`, and validate in the build script only what we rely on.
- **Windows users have no one-line installer.** → The setup skill links the release archive. A PowerShell installer is out of scope.
- **The harness is not detected automatically.** The CLI cannot tell it is running under ChatGPT. → The setup skill and docs recommend `DKF_HARNESS`. The main skill already asks the agent to pass `--harness`.
- **Two install surfaces (skill install and plugin) could both be present in Codex**, putting the skill in context twice. → `docs/plugin.md` says to use one or the other.

## Migration Plan

Additive. A new release asset; nothing existing changes. Rollback: remove the hook and extra-files entries.

## Open Questions

- ~~What directory does a ChatGPT desktop local thread run in?~~ Resolved: the thread's folder. Upload install, the onboarding skill, and invoking the main skill by mention all worked in ChatGPT desktop.
- Codex (CLI 0.166.0), verified 2026-11-02: installs plugins only from a marketplace; unzipping into `~/.codex/plugins/` is not picked up. Through a local marketplace (`codex plugin marketplace add <dir>`, `codex plugin add particulars@<name>`), the package installs as version 0.19.0, both skills are listed, and the agent drives the CLI. That makes the deferred repo-level marketplace the natural Codex route; until then `docs/plugin.md` gives the local-marketplace recipe.
- Does the plugin portal accept `requires_local_executor` for this organisation without allowlisting? This only matters for public submission, which is out of scope.
