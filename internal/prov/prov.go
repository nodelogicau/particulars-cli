// Package prov resolves provenance (source.author/harness/model/document)
// the same way for every front-end: explicit values, then DKF_* environment
// variables, then dkf.yaml defaults, then a front-end supplied fallback for
// the harness (e.g. an MCP client's name).
package prov

import (
	"os"
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

func first(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
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
