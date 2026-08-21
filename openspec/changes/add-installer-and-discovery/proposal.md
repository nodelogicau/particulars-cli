## Why

Using `particulars` from a fresh session — especially a remote Claude Desktop / Claude Code sandbox — still needs three manual steps: find and unpack the right release archive, put it on `PATH`, and tell the CLI where the workspace is when the checkout's root is not the workspace root (upward discovery never looks *down* into `knowledge/`). v0.2.1 made the skill ship with the binary; this change makes the binary itself one command away and lets a repository declare where its workspace lives, so the documented `SessionStart` hook becomes real and sessions start with zero setup.

## What Changes

- **`install.sh`** at the repository root (stable raw URL): POSIX `sh`, detects OS/arch, resolves the latest release (or `PARTICULARS_VERSION`), downloads the archive and `checksums.txt`, verifies SHA-256, installs to `PARTICULARS_INSTALL_DIR` / `/usr/local/bin` / `~/.local/bin` without ever prompting, prints the installed version. Tested in CI against the latest release on Linux and macOS; `shellcheck`-clean.
- **Homebrew tap** (macOS): GoReleaser publishes a cask to `nodelogicau/homebrew-tap` on each release, so `brew install nodelogicau/tap/particulars` works. (`brews` formulae were removed in GoReleaser 2.16; casks are macOS-only, so Linux uses `install.sh`.) Requires a tap repository and a `HOMEBREW_TAP_GITHUB_TOKEN` secret (user-side setup, listed in tasks).
- **Workspace pointer**: a `.dkf` file containing a path (relative to the file's directory, or absolute). Discovery walks up as before; at each level `dkf.yaml` wins, else a `.dkf` pointer redirects. `init <dir> --pointer` writes it.
- **`workspace` verb**: prints the resolved workspace root and how it was found (`flag`, `env`, `dkf.yaml`, `pointer`), or exits 5 — the diagnostic for "why is this session in the wrong workspace".
- Docs: README install section (brew, curl | sh, manual), the `SessionStart` hook example made live, review-workflow setup line; `SPEC-FEEDBACK.md` item 12 proposing the pointer upstream.
- Release as **v0.3.0** (new discovery behaviour and a new distribution channel).

## Capabilities

### New Capabilities
- `installation`: `install.sh` behaviour and guarantees, Homebrew tap publication, and the CI test that exercises the installer.

### Modified Capabilities
- `workspace`: discovery honours a `.dkf` pointer; `init --pointer`; new `workspace` verb.

## Impact

- New files: `install.sh`, `.github/workflows/installer.yml` (or a job in `ci.yml`), `internal/cli/cmd_workspace.go`; `.goreleaser.yaml` gains `homebrew_casks`; `release.yml` passes the tap token.
- `internal/store`: `Discover` learns the pointer and returns how it resolved; `Init` can write the pointer.
- User-side prerequisites before the first tagged release: create `nodelogicau/homebrew-tap`, add `HOMEBREW_TAP_GITHUB_TOKEN` (a fine-grained PAT with contents:write on that repo) to this repo's Actions secrets.
- Not in scope: Windows installer (release zips remain the path; scoop/winget later), package signing/notarisation, `brew` support for Linux beyond what the tap gives for free, the MCP transport.
