package viz

import (
	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Options bound what a view contains.
type Options struct {
	// Depth limits how many levels of inputs are followed from each root
	// (the current belief and any unreconciled assertion). Zero means the
	// whole closure.
	Depth int
	// IncludeRetracted draws retracted objects, marked as such, along with
	// superseded-by edges. Off by default, matching recall.
	IncludeRetracted bool
	// Scope, when set, restricts the drawing to assertions at least that
	// wide. Unlike the Graph export, personal is a legitimate value here.
	Scope dkf.Scope
	// LabelChars bounds label length; zero uses a sensible default.
	LabelChars int
}

const defaultLabelChars = 56

func (o Options) labelChars() int {
	if o.LabelChars > 0 {
		return o.LabelChars
	}
	return defaultLabelChars
}

func (o Options) visible(g *store.Graph, a dkf.Assertion) bool {
	if a.GetRetracted() != nil && !o.IncludeRetracted {
		return false
	}
	if o.Scope != "" && !dkf.ScopeAtLeast(g.EffectiveScope(a.ObjectID()), o.Scope) {
		return false
	}
	return true
}

// Lineage builds the dialectic for the merge equivalence class containing p:
// a node per claim and synthesis, an edge per input labelled with its role.
func Lineage(g *store.Graph, p *dkf.Particular, o Options) *Model {
	members := g.ClassOf(p.ID)
	inClass := map[string]bool{}
	for _, m := range members {
		inClass[m] = true
	}
	rep := query.Analyse(g, p)
	state := map[string]State{}
	for _, id := range rep.Unsynthesised {
		state[id] = StateUnsynthesised
	}
	for _, id := range rep.Stale {
		state[id] = StateStale
	}
	if rep.Current != "" {
		state[rep.Current] = StateCurrent
	}

	b := newBuilder(ViewLineage, p.Label)

	// Roots are the current belief and everything not reconciled into it —
	// the loose claims are the point of the view, so they are never dropped
	// for being unreachable from the current synthesis.
	var roots []string
	if rep.Current != "" {
		roots = append(roots, rep.Current)
	}
	roots = append(roots, rep.Unsynthesised...)
	if o.IncludeRetracted {
		for _, a := range g.ClassAssertions(members) {
			if a.GetRetracted() != nil {
				roots = append(roots, a.ObjectID())
			}
		}
	}

	// Breadth-first from each root, so Depth counts levels of inputs.
	depth := map[string]int{}
	queue := make([]string, 0, len(roots))
	for _, id := range roots {
		if _, seen := depth[id]; seen {
			continue
		}
		depth[id] = 0
		queue = append(queue, id)
	}
	for i := 0; i < len(queue); i++ {
		id := queue[i]
		a := g.Assertion(id)
		if a == nil || !o.visible(g, a) {
			continue
		}
		label, tooltip := clip(a.GetContent(), o.labelChars())
		b.node(id, kindOf(a), state[id], label)
		if a.GetRetracted() != nil {
			b.node(id, kindOf(a), StateRetracted, label)
		}
		b.mark(id, func(n *Node) { n.Tooltip = tooltip })
		if !inClass[a.SubjectID()] {
			b.mark(id, func(n *Node) { n.Foreign = true })
		}
		if r := a.GetRetracted(); r != nil && r.SupersededBy != "" && o.IncludeRetracted {
			b.edge(id, r.SupersededBy, EdgeSuperseded, "")
			if _, seen := depth[r.SupersededBy]; !seen {
				depth[r.SupersededBy] = depth[id]
				queue = append(queue, r.SupersededBy)
			}
		}
		s, ok := a.(*dkf.Synthesis)
		if !ok {
			continue
		}
		if o.Depth > 0 && depth[id] >= o.Depth {
			continue
		}
		for _, in := range s.Inputs {
			ia := g.Assertion(in.ID)
			if ia == nil || !o.visible(g, ia) {
				continue
			}
			b.edge(in.ID, id, EdgeInput, string(in.Role)+weightSuffix(in.Weight))
			if _, seen := depth[in.ID]; !seen {
				depth[in.ID] = depth[id] + 1
				queue = append(queue, in.ID)
			}
		}
	}
	return b.build()
}

func weightSuffix(w dkf.Weight) string {
	if w == "" || w == dkf.WeightPrimary {
		return ""
	}
	return ":" + string(w)
}

func kindOf(a dkf.Assertion) Kind {
	if _, ok := a.(*dkf.Synthesis); ok {
		return KindSynthesis
	}
	return KindClaim
}
