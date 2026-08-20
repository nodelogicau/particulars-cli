package query

import (
	"sort"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// RecallOptions filters and shapes a recall.
type RecallOptions struct {
	Subject          string    // particular id; "" means any subject
	Topics           []string  // all must be present
	Scope            dkf.Scope // "" means any
	IncludeRetracted bool
	Limit            int // keep the most recent N in lineage order; <=0 means all
}

// Entry is one recalled claim or synthesis, shaped for output.
type Entry struct {
	ID         string      `json:"id"`
	Type       dkf.Type    `json:"type"`
	Subject    string      `json:"subject"`
	Content    string      `json:"content"`
	Timestamp  string      `json:"timestamp"`
	Confidence *float64    `json:"confidence,omitempty"`
	Scope      dkf.Scope   `json:"scope"`
	Topics     []string    `json:"topics,omitempty"`
	Retracted  bool        `json:"retracted"`
	Current    bool        `json:"current,omitempty"`
	Inputs     []dkf.Input `json:"inputs,omitempty"`
	Unresolved string      `json:"unresolved,omitempty"`
	Method     string      `json:"method,omitempty"`
}

func entryFor(a dkf.Assertion) Entry {
	ctx := a.GetContext()
	e := Entry{
		ID: a.ObjectID(), Type: a.ObjectType(), Subject: a.SubjectID(), Content: a.GetContent(),
		Timestamp: dkf.FormatTime(a.GetTimestamp()), Confidence: a.GetConfidence(),
		Scope: ctx.Scope, Topics: ctx.Topics, Retracted: a.GetRetracted() != nil,
	}
	if s, ok := a.(*dkf.Synthesis); ok {
		e.Inputs = s.Inputs
		e.Unresolved = s.Unresolved
		e.Method = s.Method
	}
	return e
}

// CurrentSynthesis returns the most recent (highest id) non-retracted
// synthesis about subject, or nil.
func CurrentSynthesis(g *store.Graph, subject string) *dkf.Synthesis {
	var cur *dkf.Synthesis
	for _, a := range g.BySubject[subject] {
		s, ok := a.(*dkf.Synthesis)
		if !ok || s.Retracted != nil {
			continue
		}
		if cur == nil || s.ID > cur.ID {
			cur = s
		}
	}
	return cur
}

// Recall returns matching assertions in lineage order: every object precedes
// any synthesis that cites it, ties broken by ascending id.
func Recall(g *store.Graph, opts RecallOptions) []Entry {
	var candidates []dkf.Assertion
	if opts.Subject != "" {
		candidates = append(candidates, g.BySubject[opts.Subject]...)
	} else {
		candidates = g.SortedAssertions()
	}
	var filtered []dkf.Assertion
	for _, a := range candidates {
		if !opts.IncludeRetracted && a.GetRetracted() != nil {
			continue
		}
		ctx := a.GetContext()
		if opts.Scope != "" && ctx.Scope != opts.Scope {
			continue
		}
		if !hasAllTopics(ctx.Topics, opts.Topics) {
			continue
		}
		filtered = append(filtered, a)
	}
	ordered := LineageOrder(filtered)

	currentIDs := map[string]bool{}
	if opts.Subject != "" {
		if cur := CurrentSynthesis(g, opts.Subject); cur != nil {
			currentIDs[cur.ID] = true
		}
	} else {
		seen := map[string]bool{}
		for _, a := range ordered {
			if seen[a.SubjectID()] {
				continue
			}
			seen[a.SubjectID()] = true
			if cur := CurrentSynthesis(g, a.SubjectID()); cur != nil {
				currentIDs[cur.ID] = true
			}
		}
	}

	if opts.Limit > 0 && len(ordered) > opts.Limit {
		ordered = ordered[len(ordered)-opts.Limit:]
	}
	out := make([]Entry, 0, len(ordered))
	for _, a := range ordered {
		e := entryFor(a)
		e.Current = currentIDs[e.ID]
		out = append(out, e)
	}
	return out
}

func hasAllTopics(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := map[string]bool{}
	for _, t := range have {
		set[t] = true
	}
	for _, t := range want {
		if !set[t] {
			return false
		}
	}
	return true
}

// LineageOrder sorts assertions so that inputs precede the syntheses citing
// them (within the given set), ties broken by id. Cycles cannot occur in a
// well-formed workspace; if present, remaining nodes are appended by id.
func LineageOrder(as []dkf.Assertion) []dkf.Assertion {
	byID := map[string]dkf.Assertion{}
	for _, a := range as {
		byID[a.ObjectID()] = a
	}
	pending := append([]dkf.Assertion{}, as...)
	sort.Slice(pending, func(i, j int) bool { return pending[i].ObjectID() < pending[j].ObjectID() })
	emitted := map[string]bool{}
	out := make([]dkf.Assertion, 0, len(as))
	for len(pending) > 0 {
		progressed := false
		for i, a := range pending {
			ready := true
			for _, in := range a.InputIDs() {
				if _, inSet := byID[in]; inSet && !emitted[in] && in != a.ObjectID() {
					ready = false
					break
				}
			}
			if ready {
				out = append(out, a)
				emitted[a.ObjectID()] = true
				pending = append(pending[:i], pending[i+1:]...)
				progressed = true
				break
			}
		}
		if !progressed {
			out = append(out, pending...)
			break
		}
	}
	return out
}
