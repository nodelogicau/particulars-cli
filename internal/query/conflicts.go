package query

import (
	"sort"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Report is the structural conflict summary for one particular.
type Report struct {
	Particular    string   `json:"particular"`
	Label         string   `json:"label"`
	URI           string   `json:"uri"`
	Current       string   `json:"current,omitempty"`
	Unsynthesised []string `json:"unsynthesised"`
	Stale         []string `json:"stale"`
	Priority      int      `json:"priority"`
}

// Analyse computes the conflict structure for one particular regardless of
// whether it meets the reporting threshold.
func Analyse(g *store.Graph, p *dkf.Particular) Report {
	r := Report{Particular: p.ID, Label: p.Label, URI: p.URI, Unsynthesised: []string{}, Stale: []string{}}
	cur := CurrentSynthesis(g, p.ID)
	reconciled := map[string]bool{}
	if cur != nil {
		r.Current = cur.ID
		closure(g, cur, reconciled)
	}
	for _, a := range g.BySubject[p.ID] {
		if a.GetRetracted() != nil {
			continue
		}
		id := a.ObjectID()
		if cur == nil || (id != cur.ID && !reconciled[id]) {
			r.Unsynthesised = append(r.Unsynthesised, id)
		}
		if s, ok := a.(*dkf.Synthesis); ok {
			for _, in := range s.Inputs {
				if child := g.Assertion(in.ID); child != nil && child.GetRetracted() != nil {
					r.Stale = append(r.Stale, s.ID)
					break
				}
			}
		}
	}
	sort.Strings(r.Unsynthesised)
	sort.Strings(r.Stale)
	r.Priority = len(r.Unsynthesised) + len(r.Stale)
	return r
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

// Conflicts returns reports for the given particular id (or all particulars
// when subject is ""), filtered by the threshold and ordered by priority
// descending, then particular id ascending.
func Conflicts(g *store.Graph, subject string) []Report {
	var targets []*dkf.Particular
	if subject != "" {
		if p := g.Particular(subject); p != nil {
			targets = append(targets, p)
		}
	} else {
		targets = g.SortedParticulars()
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
