## Why

ChatGPT desktop and Codex now install plugins in the portable Agent Plugins format (a `plugin.json` plus `skills/`), and ChatGPT desktop runs them in threads that execute locally. particulars has no package for either: Claude Desktop gets the `.mcpb`, shell harnesses get `skill install`, but a ChatGPT desktop or Codex user has to copy the skill by hand. A skills-only plugin reuses the skill we already embed, avoids the local-MCP submission route (limited to allowlisted partners), and keeps the ~86-token collapsed cost rather than the ~4,900 tokens of MCP tools.

## What Changes

- New release asset `particulars-<version>-plugin.zip`: an Agent Plugins package containing a root `plugin.json`, the version-stamped `particulars` skill, an onboarding `setup` skill, and the icon. Listed in `checksums.txt`.
- The package is skills-only: no `mcp.json`, no bundled binaries. `extensions.com.openai.requires_local_executor` is `true`, which marks it **Desktop only**.
- The package names `skills/setup/SKILL.md` as its `onboardingSkill`. That skill checks that `particulars` is on PATH, says how to install it if not, and helps the user find or create a workspace (`particulars init --pointer` / `workspace pointer`).
- New `plugin/` source directory (sibling to `bundle/`): `plugin.json.tmpl`, the setup skill, and assets. New `scripts/build-plugin.sh` and `make plugin` target, wired into the GoReleaser release like the `.mcpb`.
- Docs: a README section and `docs/plugin.md` covering installation by upload, workspace publishing, and Codex, and what public-directory submission would need.

Non-goals:
- A local stdio MCP server in the package (needs partner allowlisting; the `.mcpb` remains the MCP route).
- Bundling binaries in the package.
- A repo-level Codex marketplace (`.agents/plugins/marketplace.json`). It can follow once the package exists.
- Submitting to the public OpenAI Plugin Directory (needs organisational identity verification).
- A Windows installer script.

## Capabilities

### New Capabilities
- `plugin-package`: the Agent Plugins package for ChatGPT desktop and Codex. Covers its manifest, contents, onboarding skill, build target, release attachment, and documentation.

### Modified Capabilities
<!-- none: the skill text, the .mcpb, and install.sh are unchanged -->

## Impact

- New files: `plugin/plugin.json.tmpl`, `plugin/skills/setup/SKILL.md`, `scripts/build-plugin.sh`, `docs/plugin.md`.
- `Makefile` (`plugin` target), `.goreleaser.yaml` (build hook, `release.extra_files`, `checksum.extra_files`), `README.md`, `CHANGELOG.md`.
- No Go code changes. The build renders the skill with the binary's existing `skill show`.
- Unverified: the working directory a ChatGPT desktop local thread starts in. This decides whether workspace discovery by cwd or `.dkf` works there, or whether the setup skill must point users at `DKF_WORKSPACE`. It is a verification task before release.
