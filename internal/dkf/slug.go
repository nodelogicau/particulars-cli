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
// base + slug (a "/" is inserted if base is hierarchical and lacks a trailing
// delimiter); without one it is urn:dkf:<workspaceID>:<slug>.
func MintURI(baseURI, workspaceID, slug string) string {
	if baseURI == "" {
		return URNPrefix + workspaceID + ":" + slug
	}
	if !strings.HasSuffix(baseURI, "/") && !strings.HasSuffix(baseURI, "#") && !strings.HasSuffix(baseURI, ":") {
		baseURI += "/"
	}
	return baseURI + slug
}
