package dkf

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Document is the evidence a claim rests on. It is either a bare reference —
// a URI, a repo-relative path, or a description of something unfetchable like
// a conversation — or that reference plus what makes it checkable: a hash of
// the document as it stood, and the sentence that supports the claim.
//
// Both forms are valid and a bare reference is not inferior provenance. The
// scalar form serialises as a scalar, so every claim written before the
// mapping existed is untouched.
type Document struct {
	URI   string `yaml:"uri" json:"uri"`
	Hash  string `yaml:"hash,omitempty" json:"hash,omitempty"`
	Quote string `yaml:"quote,omitempty" json:"quote,omitempty"`
}

// hashPattern is what this implementation writes and accepts.
var hashPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

// IsZero reports whether no part of the document is set.
func (d Document) IsZero() bool { return d.URI == "" && d.Hash == "" && d.Quote == "" }

// structured reports whether the mapping form is needed to represent d.
func (d Document) structured() bool { return d.Hash != "" || d.Quote != "" }

// String returns the reference, which is what every pre-existing consumer of
// source.document expected to find there.
func (d Document) String() string { return d.URI }

// UnmarshalYAML accepts either a scalar reference or the mapping form.
func (d *Document) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		var s string
		if err := n.Decode(&s); err != nil {
			return err
		}
		*d = Document{URI: s}
		return nil
	}
	type plain Document // avoid recursing into this method
	var p plain
	if err := n.Decode(&p); err != nil {
		return err
	}
	*d = Document(p)
	return nil
}

// MarshalYAML emits a scalar unless a hash or quote needs carrying.
func (d Document) MarshalYAML() (any, error) {
	if !d.structured() {
		return d.URI, nil
	}
	type plain Document
	return plain(d), nil
}

// UnmarshalJSON mirrors the YAML behaviour so MCP clients may send either.
func (d *Document) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*d = Document{URI: s}
		return nil
	}
	type plain Document
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*d = Document(p)
	return nil
}

// MarshalJSON emits a string for the scalar form, so a result that was a
// string before this capability existed is still a string.
func (d Document) MarshalJSON() ([]byte, error) {
	if !d.structured() {
		return json.Marshal(d.URI)
	}
	type plain Document
	return json.Marshal(plain(d))
}

// Validate checks the document's own fields.
func (d Document) Validate(field string) Problems {
	var ps Problems
	if d.IsZero() {
		return nil
	}
	if strings.TrimSpace(d.URI) == "" {
		ps = append(ps, Problem{Code: CodeInvalidDocument, Field: field + ".uri",
			Message: "a document that carries a hash or a quote must name what it refers to"})
	}
	if d.Hash != "" && !hashPattern.MatchString(d.Hash) {
		ps = append(ps, Problem{Code: CodeInvalidDocument, Field: field + ".hash",
			Message: fmt.Sprintf("hash %q must be sha256:<64 lowercase hex digits>", d.Hash)})
	}
	if d.Quote != "" && strings.TrimSpace(d.Quote) == "" {
		ps = append(ps, Problem{Code: CodeInvalidDocument, Field: field + ".quote",
			Message: "quote is empty; omit it rather than recording nothing"})
	}
	return ps
}

// HashDocument returns sha256 over the content with CRLF normalised to LF and
// nothing else normalised.
//
// Line endings differ between checkouts of the same commit, so hashing raw
// bytes would report drift for every claim on a Windows checkout — the
// cries-wolf failure the quote locator exists to avoid. Everything else is
// left alone deliberately: trailing whitespace changes a hash without changing
// meaning, but normalising it would blind the check to a class of real edit.
func HashDocument(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return HashDocumentBytes(data), nil
}

// HashDocumentBytes is HashDocument for content already in memory.
func HashDocumentBytes(data []byte) string {
	sum := sha256.Sum256([]byte(normaliseNewlines(string(data))))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// QuoteMatches reports whether quote appears verbatim in document.
//
// Line endings are normalised on both sides and the quote's own leading and
// trailing whitespace is trimmed — a YAML block scalar acquires a trailing
// newline that is an artefact of the file format, not of the source. Internal
// whitespace is compared exactly: the indentation inside a quoted code block
// is part of what was quoted.
func QuoteMatches(document, quote string) bool {
	q := strings.TrimSpace(normaliseNewlines(quote))
	if q == "" {
		return false
	}
	return strings.Contains(normaliseNewlines(document), q)
}

func normaliseNewlines(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
