// Package query implements the read-side operations over a loaded graph:
// subject resolution, recall, lineage, conflict detection, and validation.
// Everything here is pure with respect to the filesystem except Validate,
// which consults the workspace for index consistency.
package query

import (
	"regexp"
	"sort"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/apperr"
	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Resolve returns every particular matching query: id or uri exactly, a uri
// reached through non-retracted merge records, or label/alias
// case-insensitively. Results are ordered by id.
func Resolve(g *store.Graph, query string) []*dkf.Particular {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	if p := g.Particular(query); p != nil {
		return []*dkf.Particular{p}
	}
	if p := g.ParticularByURI(query); p != nil {
		return []*dkf.Particular{p}
	}
	if ps := g.MergedURIOwners(query); len(ps) > 0 {
		return ps
	}
	out := nameMatches(g, query)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// nameMatches returns the particulars whose label or any alias equals the
// query case-insensitively, unordered.
func nameMatches(g *store.Graph, query string) []*dkf.Particular {
	var out []*dkf.Particular
	for _, p := range g.Particulars {
		if strings.EqualFold(p.Label, query) {
			out = append(out, p)
			continue
		}
		for _, a := range p.Aliases {
			if strings.EqualFold(a, query) {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// Collapse reduces matches that are all one merge equivalence class to the
// lowest-id member: the workspace asserts they are the same thing, so
// treating them as an ambiguity would refuse a reference the merge record
// exists to make unambiguous. Matches spanning classes are returned as given.
func Collapse(g *store.Graph, matches []*dkf.Particular) []*dkf.Particular {
	if len(matches) < 2 {
		return matches
	}
	class := map[string]bool{}
	for _, id := range g.ClassOf(matches[0].ID) {
		class[id] = true
	}
	for _, m := range matches[1:] {
		if !class[m.ID] {
			return matches
		}
	}
	return matches[:1]
}

// RefKind classifies a particular reference by shape: a par_ id, a
// scheme-prefixed URI, or a bare name. A name containing a colon after a
// scheme-shaped prefix classifies as a URI and is simply written and read
// unchanged — a harmless degradation the spec accepts.
type RefKind int

// The three reference forms of source-attribution.
const (
	RefID RefKind = iota
	RefURI
	RefName
)

var schemePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// ClassifyRef reports which reference form a value takes.
func ClassifyRef(v string) RefKind {
	switch {
	case strings.HasPrefix(v, "par_"):
		return RefID
	case schemePattern.MatchString(v):
		return RefURI
	}
	return RefName
}

// ResolveAuthor resolves an author reference — id, URI, or bare name — to
// the particular it identifies, per source-attribution: an id exactly; a URI
// by uri, including through merges (any class member stands for the class); a
// name by label or alias when exactly one particular matches. It returns nil
// when the value resolves to nothing, and nil plus the candidate ids when a
// name matches more than one particular — resolution never guesses.
func ResolveAuthor(g *store.Graph, value string) (*dkf.Particular, []string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	switch ClassifyRef(value) {
	case RefID:
		return g.Particular(value), nil
	case RefURI:
		if p := g.ParticularByURI(value); p != nil {
			return p, nil
		}
		if ps := g.MergedURIOwners(value); len(ps) > 0 {
			return ps[0], nil
		}
		return nil, nil
	}
	ms := Collapse(g, nameMatches(g, value))
	switch len(ms) {
	case 0:
		return nil, nil
	case 1:
		return ms[0], nil
	}
	ids := make([]string, len(ms))
	for i, m := range ms {
		ids[i] = m.ID
	}
	sort.Strings(ids)
	return nil, ids
}

// ResolveAuthorForWrite applies the writer half of source-attribution to one
// author value and returns what the file should carry. A defined particular
// is written as its uri — the identifier that survives leaving the workspace,
// and the one that freezes a resolution that was unambiguous at write time. A
// URI is written unchanged whether or not a particular carries it: it is the
// right identity of someone defined elsewhere. An id that names nothing is
// refused — nobody is called par_… — and a bare name matching several
// particulars is refused when the caller gave it explicitly, since the caller
// can pass an id or URI instead; a default author falls through unchanged,
// with the candidates returned so the caller can say so, because failing
// there would block every write in the workspace until an alias is edited.
func ResolveAuthorForWrite(g *store.Graph, value string, explicit bool, field string) (written string, ambiguous []string, err error) {
	value = strings.TrimSpace(value)
	if value == "" || g == nil {
		return value, nil, nil
	}
	switch ClassifyRef(value) {
	case RefID:
		p, _ := ResolveAuthor(g, value)
		if p == nil {
			return "", nil, apperr.NotFound("%s %s names no particular — pass a URI or name, or define the particular first", field, value)
		}
		return p.URI, nil, nil
	case RefURI:
		return value, nil, nil
	}
	p, candidates := ResolveAuthor(g, value)
	if p != nil {
		return p.URI, nil, nil
	}
	if len(candidates) > 0 && explicit {
		return "", nil, apperr.Usage("%s %q is ambiguous; it matches %s — use an id or uri", field, value, strings.Join(candidates, ", "))
	}
	return value, candidates, nil
}
