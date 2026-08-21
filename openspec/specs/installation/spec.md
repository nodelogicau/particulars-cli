# Installation

## Purpose

How the binary is distributed and installed: the checksum-verifying `install.sh` and its install-directory rules, the CI job that exercises it against real releases, and the Homebrew cask published to the tap on each release.

## Requirements

### Requirement: Install script
The repository SHALL provide `install.sh`, a POSIX `sh` script that, without prompting, detects the operating system (`Linux`, `Darwin`) and architecture (`amd64`, `arm64`), resolves the release to install (`PARTICULARS_VERSION` with or without a leading `v`, else the latest release), downloads that release's archive and `checksums.txt`, verifies the archive's SHA-256 against `checksums.txt`, extracts the binary in a temporary directory, installs it, and prints the installed version. It SHALL exit non-zero with a message on an unsupported platform, a failed download, a checksum mismatch, or when neither `sha256sum` nor `shasum` is available. `PARTICULARS_REPO` SHALL override the `owner/name` used for downloads.

#### Scenario: Latest release
- **WHEN** `curl -sSL …/install.sh | sh` is run on a supported platform with `PARTICULARS_INSTALL_DIR` set to a writable directory
- **THEN** `<dir>/particulars` exists, is executable, and `particulars version` prints the latest release version

#### Scenario: Pinned version
- **WHEN** the script is run with `PARTICULARS_VERSION=v0.2.1`
- **THEN** the installed binary reports `particulars 0.2.1`

#### Scenario: Checksum mismatch
- **WHEN** the downloaded archive does not match `checksums.txt`
- **THEN** the script exits non-zero, reports the mismatch, and installs nothing

#### Scenario: Unsupported platform
- **WHEN** the script is run where `uname -m` is neither `x86_64` nor `arm64`/`aarch64`
- **THEN** it exits non-zero and points at the releases page

### Requirement: Install directory selection
The script SHALL install to `PARTICULARS_INSTALL_DIR` when set; otherwise to `/usr/local/bin` when writable; otherwise to `/usr/local/bin` via `sudo -n` when that succeeds without a password; otherwise to `$HOME/.local/bin`, creating it. It SHALL never prompt for a password. When the chosen directory is not on `PATH`, it SHALL print a one-line hint.

#### Scenario: No privileges, no sudo
- **WHEN** `/usr/local/bin` is not writable and `sudo -n true` fails
- **THEN** the binary is installed to `$HOME/.local/bin` and a PATH hint is printed if needed

#### Scenario: Explicit directory
- **WHEN** `PARTICULARS_INSTALL_DIR=/opt/tools/bin` is set and writable
- **THEN** the binary is installed there and no other location is tried

### Requirement: Installer is exercised in CI
A GitHub Actions job SHALL run `shellcheck` on `install.sh` and SHALL execute it on Linux and macOS runners — once for the latest release and once pinned to a known tag — asserting `particulars version` succeeds. The job SHALL run on pull requests that change `install.sh` and on a weekly schedule.

#### Scenario: Script regression
- **WHEN** a pull request changes `install.sh` so that extraction fails
- **THEN** the installer job fails on that pull request

### Requirement: Homebrew tap
Each tagged release SHALL publish a `particulars` cask to `nodelogicau/homebrew-tap` (`Casks/particulars.rb`) via GoReleaser's `homebrew_casks`, using `HOMEBREW_TAP_GITHUB_TOKEN`, so that `brew install nodelogicau/tap/particulars` installs the released binary on macOS (casks are macOS-only; Linux uses `install.sh`). The cask SHALL clear the macOS quarantine attribute on install because the binary is not notarised. Pre-release tags SHALL NOT update the cask. A missing token SHALL fail only the cask step, not the binary release.

#### Scenario: Brew install
- **WHEN** `brew install nodelogicau/tap/particulars` is run on macOS after a release
- **THEN** `particulars version` reports that release's version

#### Scenario: Missing token
- **WHEN** the release workflow runs without `HOMEBREW_TAP_GITHUB_TOKEN`
- **THEN** the release archives are still published and the cask step reports the missing secret
