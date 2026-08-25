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
	// documentHash and quote make the document checkable; hashDocument
	// computes the hash from the local file the document resolves to.
	documentHash, quote, quoteFile string
	hashDocument                   bool
}

// resolveDocument turns the document flags into a dkf.Document, hashing and
// reading files as needed. The locator flags require --document: a hash or a
// quote with nothing to point at records evidence for a source we did not name.
func (a *app) resolveDocument(ws *store.Workspace, f provenanceFlags) (dkf.Document, error) {
	doc := dkf.Document{URI: strings.TrimSpace(f.document), Hash: strings.TrimSpace(f.documentHash), Quote: f.quote}
	if f.quoteFile != "" {
		if doc.Quote != "" {
			return doc, usageErr("--quote and --quote-file are alternatives")
		}
		data, err := a.readContent("", f.quoteFile)
		if err != nil {
			return doc, err
		}
		doc.Quote = data
	}
	if doc.URI == "" {
		for name, set := range map[string]bool{"--document-hash": doc.Hash != "", "--quote": doc.Quote != "", "--hash-document": f.hashDocument} {
			if set {
				return doc, usageErr("%s needs --document: there is nothing to point at without it", name)
			}
		}
		return doc, nil
	}
	if f.hashDocument {
		if doc.Hash != "" {
			return doc, usageErr("--document-hash and --hash-document are alternatives")
		}
		path, ok := query.LocalDocumentPath(ws, doc.URI)
		if !ok {
			return doc, usageErr("--hash-document needs %s to resolve to a file in the workspace; pass --document-hash if you hashed it elsewhere", doc.URI)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return doc, notFoundErr("--hash-document: %v", err)
		}
		doc.Hash = dkf.HashDocumentBytes(data)
	}
	if doc.Quote != "" && strings.TrimSpace(doc.Quote) == "" {
		return doc, usageErr("--quote is blank; omit it rather than recording nothing")
	}
	return doc, nil
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
	return prov.Resolve(ws.Config.Defaults.Source, prov.Explicit{Author: f.author, Harness: f.harness, Model: f.model,
		Document: f.document, DocumentHash: f.documentHash, Quote: f.quote}, "")
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
