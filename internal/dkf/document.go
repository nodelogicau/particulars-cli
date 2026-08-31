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
	// Ref identifies the source: a URI, a path resolved against the workspace
	// root, or an identifier for something that cannot be fetched at all —
	// "chat session 2026-08-22", a recollection, a page behind a login. That
	// third case is why the field is not called uri: an unfetchable source can
	// still carry a quote, and quoting what someone said is provenance a
	// reviewer can weigh.
	Ref string `yaml:"ref" json:"ref"`
	// Author names who produced what was read — a particular reference (id,
	// URI, or bare name), distinct from source.author, who read it. It sits
	// after ref because it identifies the source, and identification precedes
	// verification.
	Author string `yaml:"author,omitempty" json:"author,omitempty"`
	Hash   string `yaml:"hash,omitempty" json:"hash,omitempty"`
	Quote  string `yaml:"quote,omitempty" json:"quote,omitempty"`

	// legacyURI records that this document was read from a file written with
	// the pre-rename `uri` key, so validate can say so. Such a file can never
	// be rewritten — appending a retraction is the only permitted
	// modification — so readers accept it in perpetuity and the warning is the
	// only way anyone learns it is there.
	legacyURI bool
}

// LegacyURI reports whether the document was read from a `uri` key.
func (d Document) LegacyURI() bool { return d.legacyURI }

// hashPattern accepts any algorithm-prefixed digest. Writers should write
// sha256; a reader that rejected other algorithms outright would make two
// conformant implementations unable to check each other's hashes, so an
// algorithm we do not implement is reported as unverified instead.
var hashPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*:[0-9a-f]+$`)

// HashAlgorithm is the algorithm prefix of a digest, or "" if malformed.
func HashAlgorithm(hash string) string {
	algo, _, ok := strings.Cut(hash, ":")
	if !ok {
		return ""
	}
	return algo
}

// AlgorithmSHA256 is the digest this implementation computes.
const AlgorithmSHA256 = "sha256"

// IsZero reports whether no part of the document is set.
func (d Document) IsZero() bool {
	return d.Ref == "" && d.Author == "" && d.Hash == "" && d.Quote == ""
}

// structured reports whether the mapping form is needed to represent d.
func (d Document) structured() bool { return d.Author != "" || d.Hash != "" || d.Quote != "" }

// String returns the reference, which is what every pre-existing consumer of
// source.document expected to find there.
func (d Document) String() string { return d.Ref }

// UnmarshalYAML accepts either a scalar reference or the mapping form.
func (d *Document) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		var s string
		if err := n.Decode(&s); err != nil {
			return err
		}
		*d = Document{Ref: s}
		return nil
	}
	// The mapping form, accepting `uri` as a legacy alias for `ref`.
	var raw struct {
		Ref    string `yaml:"ref"`
		URI    string `yaml:"uri"`
		Author string `yaml:"author"`
		Hash   string `yaml:"hash"`
		Quote  string `yaml:"quote"`
	}
	if err := n.Decode(&raw); err != nil {
		return err
	}
	*d = Document{Ref: raw.Ref, Author: raw.Author, Hash: raw.Hash, Quote: raw.Quote}
	if d.Ref == "" && raw.URI != "" {
		d.Ref, d.legacyURI = raw.URI, true
	}
	return nil
}

// MarshalYAML emits a scalar unless an author, hash, or quote needs carrying.
func (d Document) MarshalYAML() (any, error) {
	if !d.structured() {
		return d.Ref, nil
	}
	return struct {
		Ref    string `yaml:"ref"`
		Author string `yaml:"author,omitempty"`
		Hash   string `yaml:"hash,omitempty"`
		Quote  string `yaml:"quote,omitempty"`
	}{d.Ref, d.Author, d.Hash, d.Quote}, nil
}

// UnmarshalJSON mirrors the YAML behaviour so MCP clients may send either.
func (d *Document) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*d = Document{Ref: s}
		return nil
	}
	var raw struct {
		Ref    string `json:"ref"`
		URI    string `json:"uri"`
		Author string `json:"author"`
		Hash   string `json:"hash"`
		Quote  string `json:"quote"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*d = Document{Ref: raw.Ref, Author: raw.Author, Hash: raw.Hash, Quote: raw.Quote}
	if d.Ref == "" && raw.URI != "" {
		d.Ref, d.legacyURI = raw.URI, true
	}
	return nil
}

// MarshalJSON emits a string for the scalar form, so a result that was a
// string before this capability existed is still a string.
func (d Document) MarshalJSON() ([]byte, error) {
	if !d.structured() {
		return json.Marshal(d.Ref)
	}
	return json.Marshal(struct {
		Ref    string `json:"ref"`
		Author string `json:"author,omitempty"`
		Hash   string `json:"hash,omitempty"`
		Quote  string `json:"quote,omitempty"`
	}{d.Ref, d.Author, d.Hash, d.Quote})
}

// Validate checks the document's own fields.
func (d Document) Validate(field string) Problems {
	var ps Problems
	if d.IsZero() {
		return nil
	}
	if strings.TrimSpace(d.Ref) == "" {
		ps = append(ps, Problem{Code: CodeInvalidDocument, Field: field + ".ref",
			Message: "a document that carries an author, a hash, or a quote must name what it refers to"})
	}
	if d.Hash != "" {
		switch {
		case !hashPattern.MatchString(d.Hash):
			ps = append(ps, Problem{Code: CodeInvalidDocument, Field: field + ".hash",
				Message: fmt.Sprintf("hash %q must be <algorithm>:<lowercase hex digest>, for example sha256:…", d.Hash)})
		case HashAlgorithm(d.Hash) == AlgorithmSHA256 && len(d.Hash) != len(AlgorithmSHA256)+1+64:
			// An algorithm we implement is checked for shape; a truncated
			// digest is a typo, not another implementation's choice.
			ps = append(ps, Problem{Code: CodeInvalidDocument, Field: field + ".hash",
				Message: fmt.Sprintf("hash %q is not a sha256 digest: expected 64 hex digits", d.Hash)})
		}
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
	return AlgorithmSHA256 + ":" + hex.EncodeToString(sum[:])
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
