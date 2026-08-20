package query

import (
	"sort"

	"github.com/nodelogicau/particulars-cli/internal/store"
)

// TopicSummary is one row of a topic listing.
type TopicSummary struct {
	Topic       string `json:"topic"`
	Assertions  int    `json:"assertions"`  // claims + syntheses carrying the topic
	Particulars int    `json:"particulars"` // distinct subjects among them
}

// Topics lists every topic in use, honouring the subject, scope, and
// retracted filters of opts (Topics and Limit are ignored). Results are
// sorted by topic name.
func Topics(g *store.Graph, opts RecallOptions) []TopicSummary {
	counts := map[string]int{}
	subjects := map[string]map[string]bool{}
	candidates := g.SortedAssertions()
	if opts.Subject != "" {
		candidates = g.BySubject[opts.Subject]
	}
	for _, a := range candidates {
		if !opts.IncludeRetracted && a.GetRetracted() != nil {
			continue
		}
		ctx := a.GetContext()
		if opts.Scope != "" && ctx.Scope != opts.Scope {
			continue
		}
		for _, t := range ctx.Topics {
			counts[t]++
			if subjects[t] == nil {
				subjects[t] = map[string]bool{}
			}
			subjects[t][a.SubjectID()] = true
		}
	}
	out := make([]TopicSummary, 0, len(counts))
	for t, n := range counts {
		out = append(out, TopicSummary{Topic: t, Assertions: n, Particulars: len(subjects[t])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Topic < out[j].Topic })
	return out
}
