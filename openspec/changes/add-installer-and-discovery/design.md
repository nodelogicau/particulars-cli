## Context

Distribution today is "download the right archive from the releases page". The skill is now embedded (v0.2.1) and `docs/examples/claude-settings.json` already shows a `SessionStart` hook that calls an `install.sh` which does not exist. Separately, the knowledge workspace convention (`knowledge/` inside a repo, or a dedicated repo with `dkf.yaml` at the root) interacts badly with upward-only discovery: a session started at a repo root with the workspace in a subdirectory gets exit 5 unless `DKF_WORKSPACE` is set.

Constraints: the installer must never prompt (agents run it); checksums are mandatory; discovery stays spec-compatible (the spec mandates walking up for `dkf.yaml`; a pointer is an extension and is proposed upstream); no new Go dependencies.

## Goals / Non-Goals

**Goals:**
- One command installs a verified binary on Linux/macOS, interactively or in a sandbox, with or without root.
- `brew install nodelogicau/tap/particulars` for people who live in Homebrew.
- A repository can declare its workspace so no per-session configuration is needed.
- `particulars workspace` makes resolution observable.

**Non-Goals:**
- Windows installer (zips stay; scoop later), signing/notarisation, self-update (`particulars upgrade` is a reasonable follow-on once the tap exists), pointer chains (a pointer may not point at another pointer).

## Decisions

### D1. `install.sh`: POSIX sh, no prompts, env-configured

**Choice:** `#!/bin/sh` with `set -eu`; inputs via environment only, because `curl … | sh` has no argv:

| Variable | Meaning | Default |
|---|---|---|
| `PARTICULARS_VERSION` | tag to install, with or without `v` | latest release |
| `PARTICULARS_INSTALL_DIR` | target directory | see D2 |
| `PARTICULARS_REPO` | `owner/name`, for forks | `nodelogicau/particulars-cli` |

Steps: detect `uname -s` (`Linux`/`Darwin`) and `uname -m` (`x86_64`→`amd64`, `arm64`/`aarch64`→`arm64`), anything else exits 1 with a message pointing at the releases page; resolve "latest" from the `Location` header of `https://github.com/<repo>/releases/latest` (no API rate limit, no `jq`); download `particulars_<ver>_<os>_<arch>.tar.gz` and `checksums.txt` into `mktemp -d`; verify with `sha256sum -c` or `shasum -a 256 -c` (fail if neither exists); extract in the temp dir (never the cwd — the knowledge-repo CI bug); `install -m 0755` into the target; run `particulars version` from the installed path and print it. `curl` or `wget`, whichever exists.

**Why env, not flags:** piping is the common call shape; env also composes with the SessionStart hook and CI.

### D2. Install directory without prompting

Order: `PARTICULARS_INSTALL_DIR` if set → `/usr/local/bin` if writable → `/usr/local/bin` via `sudo -n` if that succeeds non-interactively → `$HOME/.local/bin` (created), with a one-line PATH hint if it is not already on `PATH`. Never `sudo` with a password prompt.

**Why:** sandboxes often have passwordless sudo or none at all; a developer machine usually has a writable `/usr/local/bin` or `~/.local/bin` on PATH. Every branch ends with a working binary somewhere and says where.

### D3. Installer is tested in CI against real releases

A workflow job on `ubuntu-latest` and `macos-latest` runs `shellcheck install.sh`, then `PARTICULARS_INSTALL_DIR=$RUNNER_TEMP/bin sh install.sh` and asserts `particulars version` succeeds; a second run with `PARTICULARS_VERSION=v0.2.1` pins a known tag. It runs on PRs that touch `install.sh` and on a weekly schedule (so a release-page change breaks loudly rather than for the next user).

### D4. Homebrew tap via GoReleaser — as a cask

GoReleaser removed `brews` in v2.16 (our workflow pins `~> v2`), so the tap entry is a **cask**:

```yaml
homebrew_casks:
  - name: particulars
    repository: {owner: nodelogicau, name: homebrew-tap, branch: main, token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"}
    directory: Casks
    url: {verified: github.com/nodelogicau/particulars-cli/}
    binaries: [particulars]
    skip_upload: auto
    hooks:
      post:
        install: |
          if OS.mac?
            system_command "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "#{staged_path}/particulars"]
          end
```

Consequences: casks are macOS-only, so `brew install` is a macOS path and Linux uses `install.sh`; the binary is not notarised, so the post-install hook strips the quarantine attribute (standard for GoReleaser casks). `release.yml` passes `HOMEBREW_TAP_GITHUB_TOKEN`; the default `GITHUB_TOKEN` cannot push to another repository, so the PAT is a hard prerequisite and the release task checks it exists before tagging.

**Why a tap, not homebrew-core:** core has notability requirements and a review queue; a tap is immediate and ours. **Why not keep a formula by pinning an old GoReleaser:** pinning a release tool to a removed feature is debt on day one.

### D5. Workspace pointer: `.dkf`

**Choice:** a plain-text file named `.dkf` whose first non-blank, non-`#` line is a path to the workspace root, relative to the `.dkf` file's directory or absolute. Discovery at each ancestor: `dkf.yaml` present → that directory; else `.dkf` present → resolve and `Open` the target (which must contain `dkf.yaml`; otherwise exit 5 naming the pointer and the target). Pointers do not chain. `--workspace` and `DKF_WORKSPACE` still win.

`particulars init <dir> --pointer` writes `./.dkf` containing the relative path to `<dir>` (refusing, exit 1, if a `.dkf` with different content exists). `init` without `<dir>` ignores `--pointer` (usage error).

**Why a separate file rather than a `dkf.yaml` with a `root:` key:** a `dkf.yaml` at the repo root would *be* a workspace marker to every other DKF implementation, which is exactly the confusion to avoid. `.dkf` is invisible to spec-conformant readers — they still find the real `dkf.yaml` by walking up from inside the workspace — so this is a pure implementation extension. Proposed upstream as SPEC-FEEDBACK item 12.

### D6. `workspace` verb

`particulars workspace [--json]` → `{"root": <abs>, "via": "flag" | "env" | "dkf.yaml" | "pointer", "pointer": <abs .dkf path, when via=pointer>, "id": <workspace.id>, "base_uri": …}`; exit 5 with the usual message when nothing resolves. `store.Discover` returns a small `Resolution` struct so the CLI does not re-derive it.

### D7. SessionStart hook becomes live

`docs/examples/claude-settings.json` loses its "does not exist yet" caveat:

```
command -v particulars >/dev/null 2>&1 || curl -sSL https://raw.githubusercontent.com/nodelogicau/particulars-cli/main/install.sh | sh
particulars skill install --json >/dev/null
```

With a `.dkf` committed at the repo root, a remote session needs nothing else.

## Risks / Trade-offs

- [`curl | sh` is a trust decision] → the script verifies checksums published by the same release pipeline; the README also shows manual and brew paths. We do not sign artifacts yet.
- [GitHub changes the `releases/latest` redirect] → the CI schedule catches it; `PARTICULARS_VERSION` bypasses it.
- [Tap publication fails if the PAT is missing/expired] → the release still ships binaries; only the cask step fails. The task list makes the secret a precondition and `release.yml` names the secret in the error.
- [`.dkf` pointer diverges from spec discovery] → readers that ignore it still work from inside the workspace; the pointer only helps tools run from above it. Raised upstream.
- [Pointer to a missing target confuses users] → exit 5 names both paths; `particulars workspace` shows resolution.

## Migration Plan

Nothing changes for existing workspaces. Repos that relied on `DKF_WORKSPACE` can add a `.dkf` and drop the env var. Release v0.3.0; the SessionStart example and README switch to the installer.

## Open Questions

- Should `install.sh` also run `particulars skill install` when it detects a `.claude/` directory in the cwd? Tempting, but it conflates two concerns; the hook does it explicitly. Default no.
- `particulars upgrade` (re-run the installer logic from inside the binary) — natural once the release URL scheme is relied upon by the installer; not in this change.
