package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/prov"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Environment variables for provenance defaults (shared via internal/prov).
const (
	EnvAuthor  = prov.EnvAuthor
	EnvHarness = prov.EnvHarness
	EnvModel   = prov.EnvModel
)

// provenanceFlags are the shared --author/--harness/--model/--document flags.
type provenanceFlags struct {
	author, harness, model, document string
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// resolveSource applies flag → env → dkf.yaml defaults.
func resolveSource(ws *store.Workspace, f provenanceFlags) dkf.Source {
	return prov.Resolve(ws.Config.Defaults.Source, prov.Explicit{Author: f.author, Harness: f.harness, Model: f.model, Document: f.document}, "")
}

// requireProvenance enforces the format's source minimum (author or harness),
// plus harness when needHarness is set (syntheses).
func requireProvenance(src dkf.Source, needHarness bool) error { return prov.Require(src, needHarness) }

// resolveScope applies flag → dkf.yaml default → personal.
func resolveScope(ws *store.Workspace, flag string) (dkf.Scope, error) {
	s := dkf.Scope(firstNonEmpty(flag, string(ws.Config.Defaults.Scope), string(dkf.ScopePersonal)))
	if !dkf.ValidScope(s) {
		return "", usageErr("invalid scope %q: must be personal, organisation, or public", s)
	}
	return s, nil
}

// readContent returns content from --content or --content-file (path or "-").
func (a *app) readContent(content, contentFile string) (string, error) {
	switch {
	case content != "" && contentFile != "":
		return "", usageErr("use either --content or --content-file, not both")
	case content != "":
		return content, nil
	case contentFile == "-":
		if a.stdinIsTerminal != nil && a.stdinIsTerminal() {
			return "", usageErr("--content-file - requires piped stdin; refusing to read from a terminal")
		}
		b, err := io.ReadAll(a.stdin)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(string(b)) == "" {
			return "", usageErr("stdin was empty")
		}
		return string(b), nil
	case contentFile != "":
		b, err := os.ReadFile(contentFile)
		if err != nil {
			return "", usageErr("--content-file: %v", err)
		}
		return string(b), nil
	}
	return "", usageErr("content is required: pass --content <text> or --content-file <path|->")
}

func parseConfidence(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, usageErr("invalid --confidence %q: must be a number in [0, 1]", s)
	}
	if f < 0 || f > 1 {
		return nil, usageErr("invalid --confidence %v: must be in [0, 1]", f)
	}
	return &f, nil
}

func parseTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Now().UTC().Truncate(time.Second), nil
	}
	t, err := dkf.ParseTime(s)
	if err != nil {
		return time.Time{}, usageErr("%v", err)
	}
	return t, nil
}

// resolveSubject finds exactly one particular for a query.
func resolveSubject(g *store.Graph, q string) (*dkf.Particular, error) {
	matches := query.Resolve(g, q)
	switch len(matches) {
	case 0:
		return nil, notFoundErr("no particular matches %q", q)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}
	return nil, usageErr("%q is ambiguous; it matches %s — use an id or uri", q, strings.Join(ids, ", "))
}

// loadGraph loads the workspace and refuses to proceed on unreadable files.
func loadGraph(ws *store.Workspace) (*store.Graph, error) {
	g, err := ws.Load()
	if err != nil {
		return nil, err
	}
	if err := g.Err(); err != nil {
		return nil, err
	}
	return g, nil
}

func fmtConfidence(f *float64) string {
	if f == nil {
		return "-"
	}
	return strconv.FormatFloat(*f, 'g', -1, 64)
}

func oneLine(s string, max int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if r := []rune(s); max > 0 && len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	if strings.HasSuffix(word, "y") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(word, "y"))
	}
	return fmt.Sprintf("%d %ss", n, word)
}
