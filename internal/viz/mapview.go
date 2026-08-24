package viz

import (
	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Map builds the workspace view: one node per particular, weighted by what is
// known about it and carrying its conflict priority, with non-retracted merge
// records joining the members of each equivalence class.
func Map(g *store.Graph, o Options) *Model {
	b := newBuilder(ViewMap, "")
	for _, p := range g.SortedParticulars() {
		rep := query.Analyse(g, p)
		state := StatePlain
		switch {
		case len(rep.Stale) > 0:
			state = StateStale
		case len(rep.Unsynthesised) > 0:
			state = StateUnsynthesised
		case rep.Current != "":
			state = StateCurrent
		}
		b.node(p.ID, KindParticular, state, p.Label)
		weight := 0
		for _, a := range g.BySubject[p.ID] {
			if a.GetRetracted() == nil && o.visible(g, a) {
				weight++
			}
		}
		b.mark(p.ID, func(n *Node) {
			n.Weight = weight
			n.Priority = rep.Priority
		})
	}
	// Merge records are keyed on URIs, which may name particulars that do not
	// exist locally; those edges are dropped when the model resolves ids.
	for _, m := range g.SortedMerges() {
		if m.Retracted != nil {
			continue
		}
		a, bb := g.ParticularByURI(m.URIs[0]), g.ParticularByURI(m.URIs[1])
		if a == nil || bb == nil {
			continue
		}
		b.edge(a.ID, bb.ID, EdgeMerge, "")
	}
	return b.build()
}

// Build selects the view: a lineage when subject is non-nil, otherwise the map.
func Build(g *store.Graph, subject *dkf.Particular, o Options) *Model {
	if subject != nil {
		return Lineage(g, subject, o)
	}
	return Map(g, o)
}
