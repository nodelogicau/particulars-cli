package particularsskill

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderStampsVersionAndMarker(t *testing.T) {
	out := string(Render("v0.2.1"))
	if !strings.Contains(out, "\n  version: \"0.2.1\"\n") {
		t.Errorf("version not stamped:\n%s", out[:400])
	}
	head, body, ok := splitFrontmatter([]byte(out))
	if !ok {
		t.Fatal("frontmatter not found")
	}
	if !strings.HasPrefix(string(body), Marker("0.2.1")+"\n") {
		t.Errorf("marker not first body line: %q", string(body[:120]))
	}
	if !strings.Contains(string(head), "name: particulars") {
		t.Error("frontmatter lost")
	}
	if !HasMarker([]byte(out)) || HasMarker(Raw()) {
		t.Error("HasMarker wrong")
	}
	if NormaliseVersion("") != "dev" || NormaliseVersion("v1.2.3") != "1.2.3" {
		t.Error("NormaliseVersion")
	}
}

func TestBodyEqualMasksVersions(t *testing.T) {
	a, b := Render("0.2.1"), Render("0.3.0-5-gabc")
	if bytes.Equal(a, b) {
		t.Fatal("renders should differ by version")
	}
	if !BodyEqual(a, b) {
		t.Error("BodyEqual should ignore version")
	}
	if !BodyEqual(a, Raw()) {
		t.Error("rendered body must equal the committed file apart from version/marker")
	}
	changed := bytes.Replace(b, []byte("## The loop"), []byte("## The loop!"), 1)
	if BodyEqual(a, changed) {
		t.Error("real body change must be detected")
	}
}

func TestRenderIdempotent(t *testing.T) {
	once := Render("1.0.0")
	// Rendering output that already has a marker must not double it.
	raw2 := raw
	raw = once
	twice := Render("1.0.0")
	raw = raw2
	if !bytes.Equal(once, twice) {
		t.Errorf("not idempotent:\n%s\n---\n%s", once[:300], twice[:300])
	}
}

func TestCRLFInputIsNormalised(t *testing.T) {
	crlf := bytes.ReplaceAll(Render("1.0.0"), []byte("\n"), []byte("\r\n"))
	if !HasMarker(crlf) {
		t.Error("HasMarker should tolerate CRLF")
	}
	if !BodyEqual(crlf, Render("2.0.0")) {
		t.Error("BodyEqual should tolerate CRLF")
	}
	saved := raw
	raw = crlf
	defer func() { raw = saved }()
	out := Render("3.0.0")
	if bytes.Contains(out, []byte("\r")) || !bytes.Contains(out, []byte("version: \"3.0.0\"")) {
		t.Errorf("CRLF embed should render as LF with stamp:\n%s", out[:200])
	}
}

func TestCursorRule(t *testing.T) {
	out := string(RenderCursorRule("1.2.3"))
	want := "---\ndescription: \"" + Description() + "\"\nalwaysApply: false\n---\n<!-- installed by particulars 1.2.3; regenerate with: particulars skill install --harness cursor -->\n\nYou are the author of a knowledge base"
	if !strings.HasPrefix(out, want) {
		t.Errorf("cursor rule shape:\n%s", out[:300])
	}
	if Description() == "" || strings.Contains(Description(), "\n") {
		t.Errorf("description: %q", Description())
	}
	if !HasMarker([]byte(out)) || !BodyEqual([]byte(out), RenderCursorRule("9.9.9")) {
		t.Error("marker/mask for cursor variant")
	}
	if strings.Contains(out, "name: particulars") {
		t.Error("skill frontmatter leaked into cursor rule")
	}
}

func TestAgentsSectionAndSplice(t *testing.T) {
	sec := RenderAgentsSection("1.0.0")
	s := string(sec)
	if !strings.HasPrefix(s, SectionStart("1.0.0")+"\n## particulars — capturing knowledge\n") || !strings.HasSuffix(s, "\n"+SectionEnd+"\n") {
		t.Fatalf("section shape:\n%s ... %s", s[:200], s[len(s)-80:])
	}
	if !strings.Contains(s, "\n### The loop\n") || strings.Contains(s, "\n## The loop\n") {
		t.Error("headings not demoted")
	}
	// A '#' inside the fenced diagram must not be touched.
	if !strings.Contains(s, "particulars particular resolve \"X\" --json        # does X exist?") {
		t.Error("fenced content altered")
	}
	// Create.
	out, err := SpliceSection(nil, sec)
	if err != nil || string(out) != s {
		t.Errorf("create: %v", err)
	}
	// Append after user content.
	user := "# My project\n\nBuild with make.\n"
	out, err = SpliceSection([]byte(user), sec)
	if err != nil || !strings.HasPrefix(string(out), user+"\n"+SectionStart("1.0.0")) {
		t.Errorf("append:\n%s", out)
	}
	// Replace in place, surroundings byte-identical.
	above, below := "# Above\n\nkeep me\n\n", "\n# Below\n\nand me\n"
	file := above + string(RenderAgentsSection("0.1.0")) + below
	out, err = SpliceSection([]byte(file), RenderAgentsSection("2.0.0"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), above) || !strings.HasSuffix(string(out), below) || !strings.Contains(string(out), "installed by particulars 2.0.0") || strings.Contains(string(out), "0.1.0") {
		t.Errorf("replace:\n%s", out)
	}
	start, end, ok, broken := FindSection(out)
	if !ok || broken || string(out[start:end]) != string(RenderAgentsSection("2.0.0")) {
		t.Error("FindSection bounds")
	}
	if !BodyEqual(out[start:end], sec) {
		t.Error("sections from different versions should be body-equal")
	}
	// Broken markers.
	brokenFile := above + SectionStart("1.0.0") + "\nhalf\n"
	if _, err := SpliceSection([]byte(brokenFile), sec); err == nil {
		t.Error("broken section should be refused")
	}
	if _, _, _, b := FindSection([]byte(brokenFile)); !b {
		t.Error("FindSection should flag broken")
	}
}
