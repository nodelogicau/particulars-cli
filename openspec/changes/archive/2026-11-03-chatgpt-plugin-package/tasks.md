## 1. Verification spike

- [x] 1.1 Hand-build a minimal package (plugin.json with `requires_local_executor`, the rendered skill, a stub setup skill). Upload it to ChatGPT desktop and confirm it installs, shows **Desktop only**, and that a local thread can run `particulars version`.
- [x] 1.2 Record the working directory a ChatGPT desktop local thread starts in (`pwd`, and whether it follows a chosen project folder). Check whether `particulars workspace --json` finds a workspace through cwd or `.dkf`.
- [x] 1.3 Install the same package in Codex from a local path and confirm both skills load and the onboarding skill fires.
- [x] 1.4 Write the findings into design.md (close the open questions) and adjust the setup-skill requirement if the working directory turns out to be `$HOME`.

## 2. Package sources

- [x] 2.1 Add `plugin/plugin.json.tmpl` per the manifest requirement (`__VERSION__` placeholder, `extensions["com.openai"]` with `requires_local_executor`, `onboardingSkill`, `interface`).
- [x] 2.2 Write `plugin/skills/setup/SKILL.md`: binary check with per-OS install instructions, workspace check and the init, pointer, or env options, and the `DKF_HARNESS` recommendation. No YAML edits, no installs without the user's agreement.

## 3. Build

- [x] 3.1 Write `scripts/build-plugin.sh <version> <dist>`. It stages the package, renders `skills/particulars/SKILL.md` via the built host binary's `skill show` (falling back to `go run` with the version ldflag), copies the setup skill and `bundle/icon.png`, and substitutes the version.
- [x] 3.2 Add validation to the script before zipping: the manifest parses, name and version match, `requires_local_executor` is true, every `./` path resolves, both skills have `name` and `description` frontmatter, and there is no `mcp.json` or binary. On any failure, exit non-zero and write no zip.
- [x] 3.3 Add the `plugin` target to `Makefile` (and to `.PHONY`).
- [x] 3.4 Wire into `.goreleaser.yaml`: run the script during the release build, and add `dist/*-plugin.zip` to `release.extra_files` and `checksum.extra_files`.
- [x] 3.5 Add a test or CI step that runs `make plugin`, unzips the result, and diffs `skills/particulars/SKILL.md` against `dist/particulars skill show`.
- [x] 3.6 Run `goreleaser release --snapshot --clean` locally and confirm both the `.mcpb` and the plugin zip are produced and checksummed.

## 4. Documentation

- [x] 4.1 Write `docs/plugin.md`: ChatGPT desktop upload and workspace publishing, Codex local install, installing the binary separately, workspace discovery in local threads (from 1.2), not combining the plugin with `skill install`, and what public submission would require.
- [x] 4.2 Link `docs/plugin.md` from the README's harness section, next to the `.mcpb` and Microsoft 365 Copilot paragraphs.
- [x] 4.3 Add a CHANGELOG entry.
