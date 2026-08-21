// Package particularsskill embeds the agent-facing skill so the binary can
// show and install it. The file next to this package is the canonical copy;
// the binary stamps its own version into it at render time.
package particularsskill

import (
	"bytes"
	_ "embed"
	"regexp"
	"strings"
)

//go:embed SKILL.md
var raw []byte

var (
	versionLine = regexp.MustCompile(`(?m)^(\s+version: ).*$`)
	markerLine  = regexp.MustCompile(`(?m)^<!-- installed by particulars [^;\n]*; regenerate with: particulars skill install -->\n?`)
)

// Raw returns the embedded skill exactly as committed.
func Raw() []byte { return append([]byte(nil), raw...) }

// NormaliseVersion strips a leading "v" and maps empty to "dev".
func NormaliseVersion(v string) string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" {
		return "dev"
	}
	return v
}

// Marker is the ownership line written after the frontmatter.
func Marker(version string) string {
	return "<!-- installed by particulars " + NormaliseVersion(version) + "; regenerate with: particulars skill install -->"
}

// Render returns the skill with metadata.version stamped and the marker
// inserted immediately after the frontmatter's closing ---.
func Render(version string) []byte {
	v := NormaliseVersion(version)
	out := markerLine.ReplaceAll(raw, nil)
	head, body, ok := splitFrontmatter(out)
	if !ok {
		return out
	}
	head = versionLine.ReplaceAll(head, []byte(`${1}"`+v+`"`))
	var b bytes.Buffer
	b.Write(head)
	b.WriteString(Marker(v))
	b.WriteByte('\n')
	b.Write(body)
	return b.Bytes()
}

// splitFrontmatter splits "---\n...\n---\n" from the rest. head includes the
// closing delimiter line.
func splitFrontmatter(data []byte) (head, body []byte, ok bool) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, data, false
	}
	idx := bytes.Index(data[4:], []byte("\n---\n"))
	if idx < 0 {
		return nil, data, false
	}
	end := 4 + idx + len("\n---\n")
	return data[:end], data[end:], true
}

// HasMarker reports whether data was written by `skill install`.
func HasMarker(data []byte) bool { return markerLine.Match(data) }

// Mask replaces the stamped frontmatter version with a placeholder and drops
// the marker line, so renders from different binaries — and the committed
// file itself — compare equal. Ownership is checked separately via HasMarker.
func Mask(data []byte) []byte {
	out := versionLine.ReplaceAll(data, []byte(`${1}"X"`))
	return markerLine.ReplaceAll(out, nil)
}

// BodyEqual reports whether a and b are the same skill apart from version.
func BodyEqual(a, b []byte) bool { return bytes.Equal(Mask(a), Mask(b)) }
