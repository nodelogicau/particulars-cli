// Package particularsskill embeds the agent-facing skill so the binary can
// show and install it for several harnesses. The file next to this package is
// the canonical copy; every variant shares its body and differs only in the
// wrapper (SKILL.md frontmatter, Cursor .mdc frontmatter, or an AGENTS.md
// section). The binary stamps its own version into each at render time.
package particularsskill

import (
	"bytes"
	_ "embed"
	"errors"
	"regexp"
	"strings"
)

//go:embed SKILL.md
var raw []byte

var (
	versionLine = regexp.MustCompile(`(?m)^(\s+version: ).*$`)
	descLine    = regexp.MustCompile(`(?m)^description: (.*)$`)
	// markerLine matches the single-line ownership marker of SKILL.md and
	// .mdc variants, whatever preset hint it carries.
	markerLine = regexp.MustCompile(`(?m)^<!-- installed by particulars [^;\n]*; regenerate with: particulars skill install[^>\n]*-->\n?`)
	// sectionStart / sectionEnd bound the region owned inside AGENTS.md.
	sectionStart = regexp.MustCompile(`(?m)^<!-- particulars:skill:start — installed by particulars [^;\n]*; regenerate with: particulars skill install --harness agents-md -->\n`)
	sectionEnd   = regexp.MustCompile(`(?m)^<!-- particulars:skill:end -->\n?`)
	headingLine  = regexp.MustCompile(`^(#{1,5}) `)
)

// ErrBrokenSection is returned when a start marker has no matching end marker.
var ErrBrokenSection = errors.New("AGENTS.md has a particulars start marker but no end marker; fix or remove it by hand")

// Raw returns the embedded skill as committed, with line endings normalised
// to LF (a Windows checkout with autocrlf would otherwise embed CRLF).
func Raw() []byte { return normalise(raw) }

func normalise(data []byte) []byte {
	return bytes.ReplaceAll(append([]byte(nil), data...), []byte("\r\n"), []byte("\n"))
}

// NormaliseVersion strips a leading "v" and maps empty to "dev".
func NormaliseVersion(v string) string {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if v == "" {
		return "dev"
	}
	return v
}

// Marker is the ownership line written after the frontmatter of a SKILL.md.
func Marker(version string) string { return markerFor(version, "") }

func markerFor(version, hint string) string {
	return "<!-- installed by particulars " + NormaliseVersion(version) + "; regenerate with: particulars skill install" + hint + " -->"
}

// SectionStart is the opening marker of the AGENTS.md section.
func SectionStart(version string) string {
	return "<!-- particulars:skill:start — installed by particulars " + NormaliseVersion(version) + "; regenerate with: particulars skill install --harness agents-md -->"
}

// SectionEnd is the closing marker of the AGENTS.md section.
const SectionEnd = "<!-- particulars:skill:end -->"

// Body returns the skill without its frontmatter or any marker.
func Body() []byte {
	_, body, ok := splitFrontmatter(markerLine.ReplaceAll(Raw(), nil))
	if !ok {
		return Raw()
	}
	return body
}

// Description returns the frontmatter description, verbatim.
func Description() string {
	head, _, ok := splitFrontmatter(Raw())
	if !ok {
		return ""
	}
	m := descLine.FindSubmatch(head)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(string(m[1]))
}

// Render returns the SKILL.md variant: metadata.version stamped and the
// marker inserted immediately after the frontmatter's closing ---.
func Render(version string) []byte {
	v := NormaliseVersion(version)
	out := markerLine.ReplaceAll(Raw(), nil)
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

// RenderCursorRule returns the Cursor .mdc variant: description +
// alwaysApply frontmatter, the marker (naming the preset), then the body.
func RenderCursorRule(version string) []byte {
	var b bytes.Buffer
	b.WriteString("---\ndescription: " + quoteYAML(Description()) + "\nalwaysApply: false\n---\n")
	b.WriteString(markerFor(version, " --harness cursor"))
	b.WriteByte('\n')
	b.Write(Body())
	return b.Bytes()
}

// RenderAgentsSection returns the region installed into AGENTS.md: start
// marker, a heading, the body with every heading demoted one level, and the
// end marker.
func RenderAgentsSection(version string) []byte {
	var b bytes.Buffer
	b.WriteString(SectionStart(version))
	b.WriteString("\n## particulars — capturing knowledge\n\n")
	b.Write(demoteHeadings(Body()))
	if !bytes.HasSuffix(b.Bytes(), []byte("\n")) {
		b.WriteByte('\n')
	}
	b.WriteString("\n" + SectionEnd + "\n")
	return b.Bytes()
}

// demoteHeadings adds one level to every Markdown heading outside fenced code.
func demoteHeadings(body []byte) []byte {
	lines := strings.Split(string(body), "\n")
	inFence := false
	for i, l := range lines {
		trimmed := strings.TrimLeft(l, " ")
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if !inFence && headingLine.MatchString(l) {
			lines[i] = "#" + l
		}
	}
	return []byte(strings.Join(lines, "\n"))
}

func quoteYAML(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
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

// HasMarker reports whether data was written by `skill install` (a SKILL.md
// or .mdc marker line, or an AGENTS.md section start).
func HasMarker(data []byte) bool {
	d := normalise(data)
	return markerLine.Match(d) || sectionStart.Match(d)
}

// FindSection locates the owned region in an AGENTS.md-style file. ok is
// false when there is no start marker; broken is true when a start marker
// has no end marker.
func FindSection(data []byte) (start, end int, ok, broken bool) {
	loc := sectionStart.FindIndex(data)
	if loc == nil {
		return 0, 0, false, false
	}
	rel := sectionEnd.FindIndex(data[loc[0]:])
	if rel == nil {
		return loc[0], 0, true, true
	}
	return loc[0], loc[0] + rel[1], true, false
}

// SpliceSection returns data with section installed: created when data is
// empty, appended after a blank line when no markers exist, or replacing
// exactly the bounded region. Bytes outside the region are untouched.
func SpliceSection(data, section []byte) ([]byte, error) {
	data = normalise(data)
	start, end, ok, broken := FindSection(data)
	switch {
	case broken:
		return nil, ErrBrokenSection
	case !ok:
		if len(bytes.TrimSpace(data)) == 0 {
			return append([]byte(nil), section...), nil
		}
		var b bytes.Buffer
		b.Write(data)
		if !bytes.HasSuffix(data, []byte("\n")) {
			b.WriteByte('\n')
		}
		if !bytes.HasSuffix(b.Bytes(), []byte("\n\n")) {
			b.WriteByte('\n')
		}
		b.Write(section)
		return b.Bytes(), nil
	default:
		var b bytes.Buffer
		b.Write(data[:start])
		b.Write(section)
		b.Write(data[end:])
		return b.Bytes(), nil
	}
}

// Mask replaces the stamped frontmatter version with a placeholder and drops
// marker lines (or stamps them with X for section markers), so renders from
// different binaries — and the committed file itself — compare equal.
// Ownership is checked separately via HasMarker.
func Mask(data []byte) []byte {
	out := versionLine.ReplaceAll(normalise(data), []byte(`${1}"X"`))
	out = markerLine.ReplaceAll(out, nil)
	return sectionStart.ReplaceAll(out, []byte(SectionStart("X")+"\n"))
}

// BodyEqual reports whether a and b are the same skill apart from version.
func BodyEqual(a, b []byte) bool { return bytes.Equal(Mask(a), Mask(b)) }
