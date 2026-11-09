package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// hashFixture writes docs/a.md and returns a fixture with a subject.
func hashFixture(t *testing.T) (*fixture, string, []byte) {
	t.Helper()
	f := newFixture(t)
	p := f.particular("Project X")
	f.particular("Ben", "ben")
	content := []byte("In staging, the billing service listens on 443.\n")
	path := filepath.Join(f.w.Root, "docs", "a.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	return f, p.ID, content
}

// rawDocClaim writes a claim citing docs/a.md without the write path's
// checks, as another implementation might have written it.
func (f *fixture) rawDocClaim(subject, hash, quote string) *dkf.Claim {
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: "listens on 443", Evidential: dkf.EvidentialObserved,
		Source:  dkf.Source{Author: "ben", Document: dkf.Document{Ref: "docs/a.md", Hash: hash, Quote: quote}},
		Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	f.rawClaim(c)
	return c
}

// codesFor returns the finding codes against one claim, sorted by severity
// then code, as "severity:code".
func codesFor(t *testing.T, f *fixture, id string) []string {
	t.Helper()
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := f.w.Load()
	var out []string
	for _, fi := range fs {
		if fi.Path == g.Files[id] && fi.Code != CodeAuthorUnresolved {
			out = append(out, fi.Severity+":"+fi.Code)
		}
	}
	return out
}

func TestHashReadByValue(t *testing.T) {
	f, subj, content := hashFixture(t)
	d256, _ := dkf.DigestBytes(dkf.AlgorithmSHA256, content)
	d512, _ := dkf.DigestBytes(dkf.AlgorithmSHA512, content)
	const quote = "listens on 443"
	for name, tc := range map[string]struct {
		hash string
		want string
	}{
		"canonical":          {"sha256:" + d256, ""},
		"uppercase hex":      {"sha256:" + strings.ToUpper(d256), "warning:hash_noncanonical"},
		"IANA spelling":      {"SHA-256:" + d256, "warning:hash_noncanonical"},
		"sha512":             {"sha512:" + d512, ""},
		"sha512 uppercase":   {"sha-512:" + strings.ToUpper(d512), "warning:hash_noncanonical"},
		"sha512 changed":     {"sha512:" + strings.Repeat("0", 128), "warning:context_drift"},
		"unrecognised alias": {"sha2-256:" + d256, "info:unverified_document"},
	} {
		c := f.rawDocClaim(subj, tc.hash, quote)
		got := strings.Join(codesFor(t, f, c.ID), " ")
		if got != tc.want {
			t.Errorf("%s (%s): got %q, want %q", name, tc.hash, got, tc.want)
		}
	}
}

func TestMalformedHashReadAsNone(t *testing.T) {
	f, subj, _ := hashFixture(t)
	for name, tc := range map[string]struct {
		hash, quote string
		want        string
	}{
		// Verified by the quote alone: no drift, and never context drift.
		"truncated, quote present": {"sha256:9f2a", "listens on 443", "warning:hash_malformed info:unverified_document"},
		"truncated, quote absent":  {"sha256:9f2a", "listens on 8443", "warning:hash_malformed warning:quote_drift"},
		"base64, no quote":         {"sha256:n4bQgYhMfWWaL+qgxVrQFaO/TxsrC4Is0V1sFbDwCgg=", "", "warning:hash_malformed info:unverified_document"},
		"no prefix":                {strings.Repeat("ab", 32), "", "warning:hash_malformed info:unverified_document"},
		"bad grammar":              {"a--b:" + strings.Repeat("ab", 32), "listens on 443", "warning:hash_malformed info:unverified_document"},
	} {
		c := f.rawDocClaim(subj, tc.hash, tc.quote)
		got := strings.Join(codesFor(t, f, c.ID), " ")
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	if fs.HasErrors() {
		t.Errorf("a malformed hash must never fail validation: %+v", fs)
	}
	if !IsCorpusFact(CodeHashNoncanonical) || IsCorpusFact(CodeHashMalformed) {
		t.Error("hash_noncanonical is a corpus fact; hash_malformed is about the object")
	}
}

func TestMalformedHashClaimCanBeRetracted(t *testing.T) {
	f, subj, _ := hashFixture(t)
	c := f.rawDocClaim(subj, "sha256:9f2a", "listens on 443")
	f.retract(c.ID) // fails the test if the re-read rejects the file
}

func TestWriterRefusesNoncanonicalHash(t *testing.T) {
	f, subj, content := hashFixture(t)
	d, _ := dkf.DigestBytes(dkf.AlgorithmSHA256, content)
	for _, hash := range []string{"sha256:" + strings.ToUpper(d), "SHA-256:" + d, "sha256:9f2a", d} {
		c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subj, Content: "x", Evidential: dkf.EvidentialObserved,
			Source:  dkf.Source{Author: "ben", Document: dkf.Document{Ref: "docs/a.md", Hash: hash}},
			Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
		err := f.w.Create(c)
		if err == nil || !strings.Contains(err.Error(), "source.document.hash") {
			t.Errorf("Create with %q: %v", hash, err)
		}
	}
	if c := f.docClaim(subj, "x", dkf.Document{Ref: "docs/a.md", Hash: "sha256:" + d}, ""); c == nil {
		t.Error("a canonical hash is written")
	}
}

func TestWriterRefusesNoncanonicalHashOnEverySource(t *testing.T) {
	f, subj, _ := hashFixture(t)
	bad := dkf.Source{Author: "ben", Document: dkf.Document{Ref: "docs/a.md", Hash: "SHA-256:" + strings.Repeat("ab", 32)}}
	c := f.claim(subj, "x")
	if _, err := f.w.Retract(c.ID, &dkf.Retracted{Timestamp: ts, Reason: "r", Source: bad}); err == nil || !strings.Contains(err.Error(), "retracted.source.document.hash") {
		t.Errorf("retraction source: %v", err)
	}
	m := &dkf.Merge{ID: dkf.NewID(dkf.TypeMerge), URIs: []string{"urn:a", "urn:b"}, Source: bad, Timestamp: ts}
	if err := f.w.Create(m); err == nil || !strings.Contains(err.Error(), "source.document.hash") {
		t.Errorf("merge source: %v", err)
	}
	pr := &dkf.Promotion{ID: dkf.NewID(dkf.TypePublish), Claims: []string{c.ID}, Scope: dkf.ScopePublic, Source: bad, Timestamp: ts}
	if err := f.w.Create(pr); err == nil || !strings.Contains(err.Error(), "source.document.hash") {
		t.Errorf("promotion source: %v", err)
	}
}
