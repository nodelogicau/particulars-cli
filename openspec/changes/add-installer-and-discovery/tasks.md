## 1. Workspace Pointer and Verb

- [x] 1.1 `store`: `Resolution{Root, Via, Pointer}`; `Discover` returns it, honouring `.dkf` at each ancestor (dkf.yaml wins; relative/absolute target; comments and blank lines ignored; no chaining; dangling target → `ErrNoWorkspace` wrapped with both paths)
- [x] 1.2 `store`: `WritePointer(dir, target)` for `init --pointer` (refuse differing existing content)
- [x] 1.3 Store tests: pointer from repo root, dkf.yaml-beats-pointer, dangling pointer, absolute target, comment lines, no chaining
- [x] 1.4 CLI: `init --pointer` (usage error without a subdirectory `dir`); result includes `pointer` path
- [x] 1.5 CLI: `workspace` verb (`root`, `via`, `pointer`, `id`, `base_uri`; exit 5 when unresolved); register under root
- [x] 1.6 CLI tests for every scenario in the `workspace` delta spec

## 2. Installer

- [x] 2.1 Write `install.sh` (POSIX sh, `set -eu`; OS/arch detection; latest via `releases/latest` redirect; `PARTICULARS_VERSION` / `PARTICULARS_INSTALL_DIR` / `PARTICULARS_REPO`; curl-or-wget; sha256 verification; extract in `mktemp -d`; directory selection per design D2; prints installed version; PATH hint)
- [x] 2.2 `shellcheck install.sh` clean; local runs: latest, pinned `v0.2.1`, explicit dir, simulated checksum mismatch
- [x] 2.3 `.github/workflows/installer.yml`: shellcheck + ubuntu/macos matrix (latest and pinned), on PRs touching `install.sh` and weekly schedule

## 3. Homebrew Tap

- [x] 3.1 `.goreleaser.yaml`: `homebrew_casks` entry per design D4 (quarantine hook, `skip_upload: auto`); `release.yml` passes `HOMEBREW_TAP_GITHUB_TOKEN`
- [x] 3.2 Create `nodelogicau/homebrew-tap` (repo with `Casks/` and a README) — user-side or with confirmation
- [x] 3.3 Add the `HOMEBREW_TAP_GITHUB_TOKEN` secret (fine-grained PAT, contents:write on the tap repo) — user-side
- [x] 3.4 `goreleaser check` passes locally

## 4. Documentation, Feedback, Release

- [x] 4.1 README Install: brew, `curl | sh` (with the env knobs), manual archive; a "workspace in a subdirectory" note showing `init ./knowledge --pointer` and `particulars workspace`
- [x] 4.2 `docs/examples/claude-settings.json`: live hook (installer + `skill install`), caveat removed; `docs/review-workflow.md` setup line mentions `.dkf`
- [x] 4.3 `SPEC-FEEDBACK.md` item 12: the `.dkf` pointer as a proposed discovery extension
- [x] 4.4 `CHANGELOG.md` v0.3.0 entry
- [x] 4.5 Tag `v0.3.0`; verify release assets, `brew install nodelogicau/tap/particulars` on this Mac, and `install.sh` from the raw URL in a clean temp dir
- [x] 4.6 Record in `~/IdeaProjects/particulars-knowledge` (on a branch) that v0.3.0 shipped the installer, the tap, the `.dkf` pointer, and the `workspace` verb
