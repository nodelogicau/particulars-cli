# Using particulars from ChatGPT desktop and Codex

Each release attaches `particulars-<version>-plugin.zip`, an
[Agent Plugins](https://developers.openai.com/plugins/build/plugins) package that
ChatGPT desktop and Codex install. It is **skills only**: it teaches the
assistant to drive the `particulars` CLI and contains no server and no binary.

```
particulars-<version>-plugin.zip
├── plugin.json                  requires_local_executor: true → "Desktop only"
├── skills/particulars/SKILL.md  the same skill `particulars skill show` prints
├── skills/setup/SKILL.md        onboarding: binary and workspace checks
└── assets/icon.png
```

Why a skill and not the MCP server: the skill costs ~86 tokens until something
triggers it, against ~4,900 for the server's tools in every session (see
[mcp.md](mcp.md#which-should-i-use--the-mcp-server-or-the-skill-and-cli)), and the
CLI finds the workspace on every call instead of binding one at startup — which
matters because a plugin has no folder picker. Plugins with a local MCP server
can also only be submitted by partners OpenAI has allowlisted.

## 1. Install the binary

The plugin needs `particulars` on `PATH` on the machine where commands run:

```sh
brew install nodelogicau/tap/particulars                                                     # macOS
curl -sSL https://raw.githubusercontent.com/nodelogicau/particulars-cli/main/install.sh | sh  # macOS, Linux
```

Windows: download the `.zip` for your architecture from the
[releases page](https://github.com/nodelogicau/particulars-cli/releases) and put
`particulars.exe` in a folder on `PATH`. The setup skill offers the same
commands if the binary is missing.

## 2. Install the plugin

**ChatGPT desktop.** Download the zip from the releases page and upload it in
the Plugins interface; to share it inside your organisation, publish it to your
workspace (**Personal** → ⋯ → **Publish**). It is marked **Desktop only** and can
be used only in threads that run locally. Web and mobile ChatGPT cannot use it:
they run in OpenAI's cloud with no access to your files.

**Codex.** Codex installs plugins only from a marketplace; unzipping into
`~/.codex/plugins/` is not picked up. A local marketplace is a folder holding the
unzipped plugin and a `.agents/plugins/marketplace.json` that names it:

```sh
mkdir -p ~/codex-marketplace/.agents/plugins ~/codex-marketplace/plugins
unzip particulars-<version>-plugin.zip -d ~/codex-marketplace/plugins/particulars
cat > ~/codex-marketplace/.agents/plugins/marketplace.json <<'EOF'
{
  "name": "local",
  "plugins": [
    { "name": "particulars",
      "source": { "source": "local", "path": "./plugins/particulars" },
      "policy": { "installation": "AVAILABLE", "authentication": "ON_INSTALL" } }
  ]
}
EOF
codex plugin marketplace add ~/codex-marketplace
codex plugin add particulars@local
```

`codex plugin list` then shows `particulars@local  installed, enabled`. To
upgrade, unzip the new release over the old one and run
`codex plugin marketplace upgrade`.

Do not also run `particulars skill install --harness agents` for the same
harness: the skill would be in context twice. Use one or the other; the plugin
is the one that updates with a new release download.

## 3. First run

After install, the `setup` skill runs. It checks `particulars version`, then
`particulars workspace --json`. When no workspace resolves from the thread's
directory it offers to:

- create one with `particulars init <dir> --pointer`, which also writes a `.dkf`
  pointer so the current directory finds it;
- point at an existing one with `particulars workspace pointer <dir>`; or
- use `DKF_WORKSPACE=<dir>` for one workspace everywhere.

The CLI cannot detect that it is running inside ChatGPT or Codex, so pass
`--harness chatgpt` / `--harness codex` on writes, or set `DKF_HARNESS`, for
provenance to name the harness.

## Workspace discovery in local threads

The CLI looks at `--workspace`, then `$DKF_WORKSPACE`, then the nearest ancestor
of the current directory holding `dkf.yaml` or a `.dkf` pointer. A ChatGPT
desktop local thread runs commands in its folder, and in Codex the current
directory is the project. Either way, a `.dkf` pointer in that folder (which the
setup skill can write) works as it does in Claude Code. Use `DKF_WORKSPACE` only
when you want one workspace regardless of folder.

## Publishing to the OpenAI Plugin Directory

Not done for this package. Public submission needs a verified developer or
business identity in the
[plugin submission portal](https://platform.openai.com/plugins); a skills-only
plugin with `requires_local_executor` follows the standard flow. Uploading the
zip yourself or publishing it to a workspace needs neither.

## Building it

```sh
make plugin                                   # builds a host binary to render the skill
PARTICULARS_BIN=dist/particulars make plugin  # or render with one already built
```

`scripts/build-plugin.sh` refuses to write a zip when the manifest does not
parse, its version or the skill's stamp differs from the build version,
`requires_local_executor` is not set, a manifest path does not resolve, a skill
lacks `name`/`description`, or anything executable or an `mcp.json` is staged.
