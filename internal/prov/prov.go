// Package prov resolves provenance (source.author/harness/model/document)
// the same way for every front-end: explicit values, then DKF_* environment
// variables, then dkf.yaml defaults, then a front-end supplied fallback for
// the harness (e.g. an MCP client's name).
package prov

import (
	"os"
	"regexp"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// Environment variables for provenance defaults.
const (
	EnvAuthor  = "DKF_AUTHOR"
	EnvHarness = "DKF_HARNESS"
	EnvModel   = "DKF_MODEL"
)

// Explicit holds values supplied directly by a caller (flags or tool args).
type Explicit struct {
	Author, Harness, Model, Document string
}

// placeholder matches an unsubstituted template variable such as
// "${user_config.author}". Hosts that expand a manifest leave these literal
// when the value is optional and unset, and a literal recorded as
// source.author is worse than no author at all — the format allows harness
// alone, so treat it as absent.
var placeholder = regexp.MustCompile(`^\$\{[^}]*\}$`)

// IsPlaceholder reports whether v is an unsubstituted template variable.
func IsPlaceholder(v string) bool { return placeholder.MatchString(strings.TrimSpace(v)) }

func first(vs ...string) string {
	for _, v := range vs {
		if v = strings.TrimSpace(v); v != "" && !IsPlaceholder(v) {
			return v
		}
	}
	return ""
}

// Resolve layers explicit → env → defaults → fallbackHarness.
func Resolve(defaults dkf.Source, e Explicit, fallbackHarness string) dkf.Source {
	return dkf.Source{
		Author:   first(e.Author, os.Getenv(EnvAuthor), defaults.Author),
		Harness:  first(e.Harness, os.Getenv(EnvHarness), defaults.Harness, fallbackHarness),
		Model:    first(e.Model, os.Getenv(EnvModel), defaults.Model),
		Document: strings.TrimSpace(e.Document),
	}
}

// Require enforces the format's minimum (author or harness), plus harness
// when needHarness is set (syntheses).
func Require(src dkf.Source, needHarness bool) error {
	if needHarness && src.Harness == "" {
		return apperr.Usage("source.harness is required for a synthesis: pass --harness, set %s, or configure defaults.source.harness in dkf.yaml", EnvHarness)
	}
	if src.Author == "" && src.Harness == "" {
		return apperr.Usage("a source needs at least one of author or harness: pass --author/--harness, set %s/%s, or configure defaults.source in dkf.yaml", EnvAuthor, EnvHarness)
	}
	return nil
}
