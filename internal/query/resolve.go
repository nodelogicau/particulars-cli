// Package query implements the read-side operations over a loaded graph:
// subject resolution, recall, lineage, conflict detection, and validation.
// Everything here is pure with respect to the filesystem except Validate,
// which consults the workspace for index consistency.
package query

import (
	"sort"
	"strings"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Resolve returns every particular matching query: id or uri exactly, or
// label/alias case-insensitively. Results are ordered by id.
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
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
