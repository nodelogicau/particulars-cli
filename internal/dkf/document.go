package dkf

import (
	"crypto/sha256"
	"crypto/sha512"
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

// hashGrammar is the OCI image-spec digest grammar that DKF pins
// source.document.hash to: algorithm components [a-z0-9]+ joined by single
// separators from + . _ -, a colon, and an encoded part.
var hashGrammar = regexp.MustCompile(`^[a-z0-9]+(?:[+._-][a-z0-9]+)*:[a-zA-Z0-9=_-]+$`)

var hexDigits = regexp.MustCompile(`^[0-9a-fA-F]+$`)

// The registered algorithms. AlgorithmSHA256 is the one writers write.
const (
	AlgorithmSHA256 = "sha256"
	AlgorithmSHA512 = "sha512"
)

// registeredHashes maps each registered algorithm to its digest length in
// hex digits.
var registeredHashes = map[string]int{AlgorithmSHA256: 64, AlgorithmSHA512: 128}

// HashForm classifies a hash as written.
type HashForm int

const (
	// HashCanonical: what a writer may write.
	HashCanonical HashForm = iota
	// HashNoncanonical: a registered algorithm a reader accepts but a writer
	// refuses — uppercase hex, an uppercase name, or sha-256/sha-512.
	HashNoncanonical
	// HashMalformed: outside the grammar, or a registered algorithm whose
	// digest is not hex or not the registered length. Read as no hash.
	HashMalformed
)

// ParsedHash is a hash read as the spec defines it.
type ParsedHash struct {
	// Algorithm is the registered name (sha256, sha512) when registered,
	// otherwise the algorithm as written.
	Algorithm string
	// Digest is lowercase hex for a registered algorithm; otherwise the
	// encoded part as written, which is opaque.
	Digest     string
	Registered bool
	Form       HashForm
}

// ParseHash reads a source.document.hash. Readers accept, for the registered
// algorithms, hex in either case, the name in any case, and the spellings
// sha-256 and sha-512; no other alias is recognised, so sha2-256 is an
// unregistered algorithm.
func ParseHash(hash string) ParsedHash {
	algo, enc, ok := strings.Cut(hash, ":")
	if !ok {
		return ParsedHash{Form: HashMalformed}
	}
	name := strings.ToLower(algo)
	switch name {
	case "sha-256":
		name = AlgorithmSHA256
	case "sha-512":
		name = AlgorithmSHA512
	}
	if n, ok := registeredHashes[name]; ok {
		h := ParsedHash{Algorithm: name, Registered: true}
		if len(enc) != n || !hexDigits.MatchString(enc) {
			h.Form = HashMalformed
			return h
		}
		h.Digest = strings.ToLower(enc)
		if algo != name || enc != h.Digest {
			h.Form = HashNoncanonical
		}
		return h
	}
	if !hashGrammar.MatchString(hash) {
		return ParsedHash{Algorithm: algo, Form: HashMalformed}
	}
	return ParsedHash{Algorithm: algo, Digest: enc, Form: HashCanonical}
}

// ImplementedHash reports whether this implementation computes algo.
func ImplementedHash(algo string) bool { return algo == AlgorithmSHA256 || algo == AlgorithmSHA512 }

// HashWriteProblem returns why a writer must refuse hash, naming the
// canonical form, or nil when it may be written. A writer never normalises
// what it was given: the caller may have computed it wrongly.
func HashWriteProblem(hash, field string) *Problem {
	h := ParsedHash{Form: HashCanonical}
	if hash != "" {
		h = ParseHash(hash)
	}
	var msg string
	switch {
	case h.Form == HashCanonical:
		return nil
	case h.Form == HashNoncanonical:
		msg = fmt.Sprintf("hash %q is not in canonical form; write it as %s:%s (lowercase name and hex)", hash, h.Algorithm, h.Digest)
	case h.Registered:
		msg = fmt.Sprintf("hash %q is not a %s digest: expected %s:<%d lowercase hex digits>", hash, h.Algorithm, h.Algorithm, registeredHashes[h.Algorithm])
	default:
		msg = fmt.Sprintf("hash %q must be <algorithm>:<digest>, for example sha256:<64 lowercase hex digits>", hash)
	}
	return &Problem{Code: CodeInvalidDocument, Field: field, Message: msg}
}

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
	// A malformed or non-canonical hash is not invalid here either, for the
	// same reason as an empty quote below: validate reports hash_malformed or
	// hash_noncanonical, and writers refuse both (HashWriteProblem).
	//
	// An empty quote is not invalid here: a claim file can never be
	// rewritten, so a read-time rejection could never be satisfied. Writers
	// refuse it; readers treat the mapping as carrying no quote and warn
	// (empty_quote), which is what HasQuote lets them do.
	return ps
}

// HasQuote reports whether the document carries a quote that says anything —
// one that folds to non-empty under the whitespace rule. This is the single
// notion of "carries a quote" shared by drift checking, the disclosure notes,
// and both write boundaries, so none of them can disagree about whether a
// quote exists. A quote that folds to nothing is a malformed write, read as
// no quote.
func (d Document) HasQuote() bool { return foldWhitespace(d.Quote) != "" }

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
	d, _ := DigestBytes(AlgorithmSHA256, data)
	return AlgorithmSHA256 + ":" + d
}

// DigestBytes returns the lowercase hex digest of content under algo, with
// CRLF normalised to LF as for HashDocument, or false when algo is not one
// this implementation computes.
func DigestBytes(algo string, data []byte) (string, bool) {
	norm := []byte(normaliseNewlines(string(data)))
	switch algo {
	case AlgorithmSHA256:
		sum := sha256.Sum256(norm)
		return hex.EncodeToString(sum[:]), true
	case AlgorithmSHA512:
		sum := sha512.Sum512(norm)
		return hex.EncodeToString(sum[:]), true
	}
	return "", false
}

// QuoteMatches reports whether quote appears in document.
//
// Words must match exactly; whitespace does not. Both sides are folded —
// every maximal run of characters with the Unicode White_Space property
// becomes one space, and the ends are trimmed — and the folded quote must be
// a substring of the folded document. So a sentence quoted from prose still
// matches after the document is wrapped at another column, checked out with
// CRLF, re-indented, or pasted with a non-breaking space. Case, punctuation,
// and Unicode normalisation form are compared verbatim: for everything other
// than whitespace, the quote agrees with the hash about what an edit is;
// whitespace is where the two signals part on purpose, since the hash asks
// whether anything changed and the quote whether the words are still there.
//
// The rule is format-blind: it strips no `>`, `//`, or `#` from a
// continuation line, so a quote across such a wrap must come from one line.
// The stored quote is never folded; this is a property of the comparison,
// not of the file.
func QuoteMatches(document, quote string) bool {
	q := foldWhitespace(quote)
	if q == "" {
		return false
	}
	return strings.Contains(foldWhitespace(document), q)
}

// foldWhitespace collapses every run of whitespace to one space and trims the
// ends. strings.Fields splits on unicode.IsSpace, which is the Unicode
// White_Space property, so tabs, newlines, blank lines, NBSP, NEL, and the
// U+2000 block all fold; CRLF needs no special case because CR is whitespace.
func foldWhitespace(s string) string { return strings.Join(strings.Fields(s), " ") }

func normaliseNewlines(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }
