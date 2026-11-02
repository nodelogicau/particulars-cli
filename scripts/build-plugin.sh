#!/bin/sh
# Assemble the ChatGPT desktop / Codex plugin (Agent Plugins package, skills
# only) from plugin/ and the embedded skill.
#
#   scripts/build-plugin.sh <version> <dist-dir>
#
# The main skill is rendered by the binary itself (`skill show`), so the packaged
# copy carries this release's version stamp and marker and cannot drift from it.
# PARTICULARS_BIN names a host binary to render with; otherwise one is built
# here with the release version. Validates the staged package and writes no zip
# when any check fails.
set -eu

VERSION="${1:?version}"; DIST="${2:?dist dir}"
VERSION="${VERSION#v}"
HERE="$(cd "$(dirname "$0")/.." && pwd)"
mkdir -p "$DIST"
# Absolute, so the zip step works from any working directory.
OUT="$(cd "$DIST" && pwd)/particulars-${VERSION}-plugin.zip"

STAGE="$(mktemp -d)"; trap 'rm -rf "$STAGE"' EXIT
rm -f "$OUT"

BIN="${PARTICULARS_BIN:-}"
if [ -z "$BIN" ]; then
  BIN="$STAGE/.bin/particulars"
  ( cd "$HERE" && CGO_ENABLED=0 go build -trimpath \
      -ldflags "-X github.com/nodelogicau/particulars-cli/internal/cli.version=${VERSION}" \
      -o "$BIN" ./cmd/particulars )
fi

PKG="$STAGE/pkg"
mkdir -p "$PKG/skills/particulars" "$PKG/skills/setup" "$PKG/assets"
sed "s/__VERSION__/${VERSION}/g" "$HERE/plugin/plugin.json.tmpl" > "$PKG/plugin.json"
"$BIN" skill show > "$PKG/skills/particulars/SKILL.md"
cp "$HERE/plugin/skills/setup/SKILL.md" "$PKG/skills/setup/SKILL.md"
cp "$HERE/bundle/icon.png" "$PKG/assets/icon.png"

# Sanity: manifest fields we rely on, every ./ path resolves, both skills have
# frontmatter, and nothing executable or MCP-shaped slipped in.
python3 - "$PKG" "$VERSION" <<'EOF'
import json, os, re, sys
pkg, version = sys.argv[1], sys.argv[2]
def fail(msg):
    sys.exit("build-plugin: " + msg)
try:
    m = json.load(open(os.path.join(pkg, "plugin.json")))
except ValueError as e:
    fail("plugin.json does not parse: %s" % e)
if m.get("$schema") != "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json":
    fail("plugin.json $schema is not Agent Plugins 1.0.0")
if m.get("name") != "particulars":
    fail("plugin.json name is %r, want 'particulars'" % m.get("name"))
if m.get("version") != version:
    fail("plugin.json version is %r, want %r" % (m.get("version"), version))
if not m.get("description"):
    fail("plugin.json has no description")
oa = (m.get("extensions") or {}).get("com.openai") or {}
if oa.get("requires_local_executor") is not True:
    fail("extensions.com.openai.requires_local_executor must be true")
paths = [oa.get("onboardingSkill"), (oa.get("interface") or {}).get("logo")]
for p in paths:
    if not p or not p.startswith("./"):
        fail("manifest path %r must start with ./" % p)
    full = os.path.realpath(os.path.join(pkg, p))
    if not full.startswith(os.path.realpath(pkg) + os.sep) or not os.path.isfile(full):
        fail("manifest path %s does not resolve inside the package" % p)
for name in ("particulars", "setup"):
    text = open(os.path.join(pkg, "skills", name, "SKILL.md")).read()
    fm = re.match(r"---\n(.*?)\n---\n", text, re.S)
    if not fm:
        fail("skills/%s/SKILL.md has no frontmatter" % name)
    if not re.search(r"^name: %s$" % name, fm.group(1), re.M):
        fail("skills/%s/SKILL.md frontmatter name is not %s" % (name, name))
    if not re.search(r"^description: \S", fm.group(1), re.M):
        fail("skills/%s/SKILL.md has no description" % name)
    if name == "particulars" and not re.search(r'^\s+version: "%s"$' % re.escape(version), fm.group(1), re.M):
        fail("skills/particulars/SKILL.md is not stamped %s; PARTICULARS_BIN is a different version" % version)
for root, _, files in os.walk(pkg):
    for f in files:
        full = os.path.join(root, f)
        rel = os.path.relpath(full, pkg)
        if rel == "mcp.json" or rel.startswith(".codex-plugin"):
            fail("%s must not be in a skills-only package" % rel)
        if os.access(full, os.X_OK):
            fail("%s is executable; the package ships no binaries" % rel)
EOF

( cd "$PKG" && zip -q -r -X "$OUT" plugin.json skills assets )
echo "build-plugin: wrote $OUT" >&2
