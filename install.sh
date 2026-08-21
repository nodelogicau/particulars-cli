#!/bin/sh
# particulars installer — https://github.com/nodelogicau/particulars-cli
#
#   curl -sSL https://raw.githubusercontent.com/nodelogicau/particulars-cli/main/install.sh | sh
#
# Configuration (environment only; the script is usually piped):
#   PARTICULARS_VERSION      tag to install, with or without a leading "v" (default: latest release)
#   PARTICULARS_INSTALL_DIR  directory to install into (default: see choose_dir)
#   PARTICULARS_REPO         owner/name to download from (default: nodelogicau/particulars-cli)
#
# Never prompts. Verifies the SHA-256 published with the release. Exits non-zero
# on an unsupported platform, a failed download, or a checksum mismatch.
set -eu

REPO="${PARTICULARS_REPO:-nodelogicau/particulars-cli}"
BIN="particulars"

say() { printf '%s\n' "$*" >&2; }
die() { say "install.sh: $*"; exit 1; }

need_one() {
  for c in "$@"; do
    if command -v "$c" >/dev/null 2>&1; then printf '%s' "$c"; return 0; fi
  done
  return 1
}

FETCHER="$(need_one curl wget)" || die "curl or wget is required"

fetch() { # url -> stdout
  if [ "$FETCHER" = curl ]; then curl -fsSL "$1"; else wget -qO- "$1"; fi
}

fetch_to() { # url file
  if [ "$FETCHER" = curl ]; then curl -fsSL -o "$2" "$1"; else wget -qO "$2" "$1"; fi
}

latest_tag() {
  # Follow the releases/latest redirect and read the tag from the Location header.
  # No API call, no rate limit, no jq.
  if [ "$FETCHER" = curl ]; then
    curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest"
  else
    wget -q --max-redirect=5 -S -O /dev/null "https://github.com/${REPO}/releases/latest" 2>&1 \
      | awk 'tolower($1)=="location:"{u=$2} END{print u}'
  fi | sed -n 's#.*/tag/##p'
}

case "$(uname -s)" in
  Linux)  OS=linux ;;
  Darwin) OS=darwin ;;
  *) die "unsupported OS $(uname -s); download a release archive from https://github.com/${REPO}/releases" ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  ARCH=amd64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) die "unsupported architecture $(uname -m); download a release archive from https://github.com/${REPO}/releases" ;;
esac

TAG="${PARTICULARS_VERSION:-}"
if [ -z "$TAG" ]; then
  TAG="$(latest_tag)" || true
  [ -n "$TAG" ] || die "could not determine the latest release; set PARTICULARS_VERSION"
fi
case "$TAG" in v*) ;; *) TAG="v${TAG}" ;; esac
VER="${TAG#v}"

ARCHIVE="${BIN}_${VER}_${OS}_${ARCH}.tar.gz"
BASE="https://github.com/${REPO}/releases/download/${TAG}"

SHA_TOOL="$(need_one sha256sum shasum)" || die "sha256sum or shasum is required to verify the download"

TMP="$(mktemp -d 2>/dev/null || mktemp -d -t particulars)"
trap 'rm -rf "$TMP"' EXIT INT TERM

say "Downloading ${BIN} ${VER} (${OS}/${ARCH})"
fetch_to "${BASE}/${ARCHIVE}" "${TMP}/${ARCHIVE}" || die "download failed: ${BASE}/${ARCHIVE}"
fetch_to "${BASE}/checksums.txt" "${TMP}/checksums.txt" || die "download failed: ${BASE}/checksums.txt"

EXPECTED="$(awk -v f="$ARCHIVE" '$2==f{print $1}' "${TMP}/checksums.txt")"
[ -n "$EXPECTED" ] || die "${ARCHIVE} not listed in checksums.txt"
if [ "$SHA_TOOL" = sha256sum ]; then
  ACTUAL="$(sha256sum "${TMP}/${ARCHIVE}" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "${TMP}/${ARCHIVE}" | awk '{print $1}')"
fi
[ "$EXPECTED" = "$ACTUAL" ] || die "checksum mismatch for ${ARCHIVE}: expected ${EXPECTED}, got ${ACTUAL}"

# Extract in the temp dir, never the cwd (a workspace checkout has a particulars/ directory).
tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP" "$BIN" || die "could not extract ${BIN} from ${ARCHIVE}"

choose_dir() {
  if [ -n "${PARTICULARS_INSTALL_DIR:-}" ]; then
    mkdir -p "$PARTICULARS_INSTALL_DIR" || die "cannot create ${PARTICULARS_INSTALL_DIR}"
    printf '%s' "$PARTICULARS_INSTALL_DIR"; return
  fi
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then printf '%s' /usr/local/bin; return; fi
  if command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then printf '%s' "sudo:/usr/local/bin"; return; fi
  mkdir -p "${HOME}/.local/bin" || die "cannot create ${HOME}/.local/bin"
  printf '%s' "${HOME}/.local/bin"
}

TARGET="$(choose_dir)"
case "$TARGET" in
  sudo:*)
    DIR="${TARGET#sudo:}"
    sudo -n install -m 0755 "${TMP}/${BIN}" "${DIR}/${BIN}" || die "install to ${DIR} failed"
    ;;
  *)
    DIR="$TARGET"
    install -m 0755 "${TMP}/${BIN}" "${DIR}/${BIN}" || die "install to ${DIR} failed"
    ;;
esac

say "Installed $("${DIR}/${BIN}" version) at ${DIR}/${BIN}"
case ":${PATH}:" in
  *":${DIR}:"*) ;;
  *) say "note: ${DIR} is not on your PATH; add it, e.g.  export PATH=\"${DIR}:\$PATH\"" ;;
esac
