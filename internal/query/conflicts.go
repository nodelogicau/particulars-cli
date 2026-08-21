package query

import (
	"fmt"
	"sort"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Report is the structural conflict summary for one particular (or merge
// equivalence class, keyed by the queried or lowest particular).
type Report struct {
	Particular    string   `json:"particular"`
	Label         string   `json:"label"`
	URI           string   `json:"uri"`
	Members       []string `json:"members,omitempty"` // all class members when > 1
	Set           []string `json:"set,omitempty"`     // the given claim set, for AnalyseSet
	Current       string   `json:"current,omitempty"`
	Unsynthesised []string `json:"unsynthesised"`
	Stale         []string `json:"stale"`
	Priority      int      `json:"priority"`
}

// Analyse computes the conflict structure for the class containing p,
// regardless of whether it meets the reporting threshold.
func Analyse(g *store.Graph, p *dkf.Particular) Report {
	members := g.ClassOf(p.ID)
	r := Report{Particular: p.ID, Label: p.Label, URI: p.URI, Unsynthesised: []string{}, Stale: []string{}}
	if len(members) > 1 {
		r.Members = members
	}
	cur := CurrentForClass(g, members)
	reconciled := map[string]bool{}
	if cur != nil {
		r.Current = cur.ID
		closure(g, cur, reconciled)
	}
	memo := map[string]bool{}
	for _, a := range g.ClassAssertions(members) {
		if a.GetRetracted() != nil {
			continue
		}
		id := a.ObjectID()
		if cur == nil || (id != cur.ID && !reconciled[id]) {
			r.Unsynthesised = append(r.Unsynthesised, id)
		}
		if s, ok := a.(*dkf.Synthesis); ok && CitesRetracted(g, s, memo) {
			r.Stale = append(r.Stale, s.ID)
		}
	}
	sort.Strings(r.Unsynthesised)
	sort.Strings(r.Stale)
	r.Priority = len(r.Unsynthesised) + len(r.Stale)
	return r
}

// AnalyseSet treats the given claim/synthesis ids as the whole universe:
// current is the most recent non-retracted synthesis in the set, unsynthesised
// the members outside its transitive inputs, stale the member syntheses citing
// a retracted object. Unknown ids are an error.
func AnalyseSet(g *store.Graph, ids []string) (Report, error) {
	r := Report{Unsynthesised: []string{}, Stale: []string{}}
	var members []dkf.Assertion
	seen := map[string]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		a := g.Assertion(id)
		if a == nil {
			return r, fmt.Errorf("%s: %w", id, store.ErrNotFound)
		}
		members = append(members, a)
		r.Set = append(r.Set, id)
	}
	sort.Strings(r.Set)
	var cur *dkf.Synthesis
	for _, a := range members {
		if s, ok := a.(*dkf.Synthesis); ok && s.Retracted == nil && (cur == nil || newer(s, cur)) {
			cur = s
		}
	}
	reconciled := map[string]bool{}
	if cur != nil {
		r.Current = cur.ID
		closure(g, cur, reconciled)
	}
	memo := map[string]bool{}
	for _, a := range members {
		if a.GetRetracted() != nil {
			continue
		}
		id := a.ObjectID()
		if cur == nil || (id != cur.ID && !reconciled[id]) {
			r.Unsynthesised = append(r.Unsynthesised, id)
		}
		if s, ok := a.(*dkf.Synthesis); ok && CitesRetracted(g, s, memo) {
			r.Stale = append(r.Stale, s.ID)
		}
	}
	sort.Strings(r.Unsynthesised)
	sort.Strings(r.Stale)
	r.Priority = len(r.Unsynthesised) + len(r.Stale)
	return r, nil
}

// CitesRetracted reports whether s cites, directly or transitively, a
// retracted object. memo caches results across calls; cycles are guarded.
func CitesRetracted(g *store.Graph, s *dkf.Synthesis, memo map[string]bool) bool {
	if v, ok := memo[s.ID]; ok {
		return v
	}
	memo[s.ID] = false // cycle guard: assume not stale while exploring
	stale := false
	for _, in := range s.Inputs {
		child := g.Assertion(in.ID)
		if child == nil {
			continue
		}
		if child.GetRetracted() != nil {
			stale = true
			break
		}
		if cs, ok := child.(*dkf.Synthesis); ok && CitesRetracted(g, cs, memo) {
			stale = true
			break
		}
	}
	memo[s.ID] = stale
	return stale
}

// closure marks every transitive input of s.
func closure(g *store.Graph, s *dkf.Synthesis, seen map[string]bool) {
	for _, in := range s.Inputs {
		if seen[in.ID] {
			continue
		}
		seen[in.ID] = true
		if child, ok := g.Assertion(in.ID).(*dkf.Synthesis); ok {
			closure(g, child, seen)
		}
	}
}

// Reportable applies the threshold: a current synthesis with anything
// unsynthesised, or no current and two or more unsynthesised, or any stale.
func Reportable(r Report) bool {
	if len(r.Stale) > 0 {
		return true
	}
	if r.Current != "" {
		return len(r.Unsynthesised) > 0
	}
	return len(r.Unsynthesised) >= 2
}

// Conflicts returns reports for the class of the given particular id (or,
// when subject is "", one report per class keyed by its lowest particular),
// filtered by the threshold and ordered by priority descending, then id.
func Conflicts(g *store.Graph, subject string) []Report {
	var targets []*dkf.Particular
	if subject != "" {
		if p := g.Particular(subject); p != nil {
			targets = append(targets, p)
		}
	} else {
		seen := map[string]bool{}
		for _, p := range g.SortedParticulars() {
			root := g.ClassOf(p.ID)[0]
			if seen[root] {
				continue
			}
			seen[root] = true
			targets = append(targets, g.Particular(root))
		}
	}
	out := []Report{}
	for _, p := range targets {
		r := Analyse(g, p)
		if Reportable(r) {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority > out[j].Priority
		}
		return out[i].Particular < out[j].Particular
	})
	return out
}
