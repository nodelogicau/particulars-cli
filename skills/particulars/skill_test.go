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
