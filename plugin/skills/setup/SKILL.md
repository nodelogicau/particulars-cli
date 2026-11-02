---
name: setup
description: Set up particulars after the plugin is installed — check the `particulars` binary is on PATH and that a DKF workspace resolves from here. Use right after installing the plugin, or when a `particulars` command fails with "command not found" or exit code 5 (no workspace).
license: MIT
---

The `particulars` skill in this plugin drives the `particulars` command-line
tool. The plugin does not include the tool itself; this skill checks that it is
installed and that a knowledge workspace can be found. Do each step in order and
stop at the first one that needs the user. Never install software or create
files without the user agreeing first, and never edit workspace YAML by hand.

## 1. Is the binary installed?

Run `particulars version`. If it prints a version, go to step 2.

If the command is not found, tell the user it needs installing and offer the
command for their system:

- macOS: `brew install nodelogicau/tap/particulars`
- macOS or Linux, without Homebrew:
  `curl -sSL https://raw.githubusercontent.com/nodelogicau/particulars-cli/main/install.sh | sh`
  (verifies the release checksum; installs to `/usr/local/bin` or `~/.local/bin`)
- Windows: download the `windows_amd64` or `windows_arm64` `.zip` from
  https://github.com/nodelogicau/particulars-cli/releases and put
  `particulars.exe` in a folder on `PATH`.

Run the install only if the user asks you to. Afterwards, run
`particulars version` again; if it is still not found, the install directory is
not on `PATH` — say which directory it went to and stop.

## 2. Does a workspace resolve?

Run `particulars workspace --json`. It looks, in order, at `--workspace`,
`$DKF_WORKSPACE`, then the nearest ancestor directory holding `dkf.yaml` or a
`.dkf` pointer file. If it succeeds, tell the user which workspace (`root`) and
how it was found (`via`), and you are done.

If it exits with code 5, no workspace resolves from the current directory
(`pwd`). Ask the user which of these they want:

- **A new workspace.** `particulars init <dir> --pointer --author <their name> --json`
  creates it and writes a `.dkf` pointer here so this directory finds it. Suggest
  a directory they will commit to git, such as `./knowledge`.
- **An existing workspace.** `particulars workspace pointer <dir> --json` writes a
  `.dkf` pointer here to the folder that holds its `dkf.yaml`.
- **One workspace for everything.** Set `DKF_WORKSPACE=<dir>` in the
  environment this app runs commands in.

Then run `particulars workspace --json` again to confirm.

## 3. Record the harness

Claims record which tool wrote them. The CLI cannot tell it is running inside
this app, so pass `--harness chatgpt` (or `--harness codex` in Codex) on
commands that write, or suggest the user set `DKF_HARNESS` once.

From here on, follow the `particulars` skill.
