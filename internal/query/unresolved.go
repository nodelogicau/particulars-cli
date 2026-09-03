package query

import (
	"sort"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// NoneIdentified is the conventional unresolved value meaning the producer
// considered the question and found nothing outstanding.
const NoneIdentified = "None identified"

// UnresolvedEntry is one admitted open question: the current synthesis of a
// merge equivalence class and what it says it could not settle.
type UnresolvedEntry struct {
	Particular    string    `json:"particular"`
	Label         string    `json:"label"`
	URI           string    `json:"uri"`
	Members       []string  `json:"members,omitempty"` // all class members when > 1
	Synthesis     string    `json:"synthesis"`         // the current synthesis
	Timestamp     time.Time `json:"timestamp"`
	Unresolved    string    `json:"unresolved"`
	Unsynthesised int       `json:"unsynthesised"` // class assertions not reconciled into current
}

// UnresolvedOptions narrows an Unresolved listing.
type UnresolvedOptions struct {
	Subject     string    // a particular id; "" for every class
	Scope       dkf.Scope // effective scope of the current synthesis; "" for any
	IncludeNone bool      // keep entries whose unresolved is NoneIdentified
}

// Unresolved lists, for every class with a current synthesis (or the class of
// opts.Subject), that synthesis's unresolved text, oldest current synthesis
// first. current and the unsynthesised count are exactly what Conflicts
// computes; classes with no current synthesis have admitted nothing and are
// omitted.
func Unresolved(g *store.Graph, opts UnresolvedOptions) []UnresolvedEntry {
	out := []UnresolvedEntry{}
	for _, p := range classTargets(g, opts.Subject) {
		r := Analyse(g, p)
		if r.Current == "" {
			continue
		}
		s, ok := g.Assertion(r.Current).(*dkf.Synthesis)
		if !ok {
			continue
		}
		if !opts.IncludeNone && s.Unresolved == NoneIdentified {
			continue
		}
		if opts.Scope != "" && g.EffectiveScope(s.ID) != opts.Scope {
			continue
		}
		out = append(out, UnresolvedEntry{
			Particular: r.Particular, Label: r.Label, URI: r.URI, Members: r.Members,
			Synthesis: s.ID, Timestamp: s.Timestamp, Unresolved: s.Unresolved,
			Unsynthesised: len(r.Unsynthesised),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Timestamp.Before(out[j].Timestamp)
		}
		return out[i].Synthesis < out[j].Synthesis
	})
	return out
}

// classTargets returns the particular to analyse for subject, or one
// particular per merge equivalence class (its lowest id) when subject is "".
func classTargets(g *store.Graph, subject string) []*dkf.Particular {
	if subject != "" {
		if p := g.Particular(subject); p != nil {
			return []*dkf.Particular{p}
		}
		return nil
	}
	var targets []*dkf.Particular
	seen := map[string]bool{}
	for _, p := range g.SortedParticulars() {
		root := g.ClassOf(p.ID)[0]
		if seen[root] {
			continue
		}
		seen[root] = true
		targets = append(targets, g.Particular(root))
	}
	return targets
}
