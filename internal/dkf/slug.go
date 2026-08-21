package dkf

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Slugify lowercases, folds Unicode to ASCII (NFKD then strip combining marks),
// collapses every run of non-alphanumerics to a single hyphen, and trims
// leading/trailing hyphens. Returns "" when nothing survives.
func Slugify(label string) string {
	t := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(t, label)
	if err != nil {
		folded = label
	}
	var b strings.Builder
	pendingHyphen := false
	for _, r := range strings.ToLower(folded) {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		switch {
		case isAlnum:
			if pendingHyphen && b.Len() > 0 {
				b.WriteByte('-')
			}
			pendingHyphen = false
			b.WriteRune(r)
		default:
			pendingHyphen = true
		}
	}
	return b.String()
}

// URNPrefix is the scheme used for particulars in workspaces without a base URI.
const URNPrefix = "urn:dkf:"

// MintURI builds a particular URI from a slug. With a base URI the result is
// base + slug (the base must already end in "/", see ValidBaseURI); without
// one it is urn:dkf:<workspaceID>:<slug>.
func MintURI(baseURI, workspaceID, slug string) string {
	if baseURI == "" {
		return URNPrefix + workspaceID + ":" + slug
	}
	return baseURI + slug
}

// NormaliseBaseURI appends a trailing "/" when base is non-empty and lacks one.
func NormaliseBaseURI(base string) string {
	base = strings.TrimSpace(base)
	if base == "" || strings.HasSuffix(base, "/") {
		return base
	}
	return base + "/"
}

// ValidBaseURI reports whether base is empty or ends in "/", as the format
// requires of workspace.base-uri.
func ValidBaseURI(base string) bool { return base == "" || strings.HasSuffix(base, "/") }
