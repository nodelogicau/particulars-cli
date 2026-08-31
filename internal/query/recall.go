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
	// Author restricts to objects asserted by or reported from this
	// particular's merge class (a resolved particular id). Each returned
	// entry then carries Relations.
	Author string
}

// Entry is one recalled claim or synthesis, shaped for output.
type Entry struct {
	ID         string         `json:"id"`
	Type       dkf.Type       `json:"type"`
	Subject    string         `json:"subject"`
	Content    string         `json:"content"`
	Source     dkf.Source     `json:"source"`
	Timestamp  string         `json:"timestamp"`
	Confidence *float64       `json:"confidence,omitempty"`
	Evidential dkf.Evidential `json:"evidential,omitempty"`
	Scope      dkf.Scope      `json:"scope"`
	Topics     []string       `json:"topics,omitempty"`
	Retracted  bool           `json:"retracted"`
	// Relations labels an author-filtered result with how it matched:
	// asserted (source.author) and/or reported (source.document.author) —
	// one object can be both, and the two are never collapsed.
	Relations     []string    `json:"relations,omitempty"`
	Current       bool        `json:"current,omitempty"`
	Unsynthesised bool        `json:"unsynthesised,omitempty"`
	Inputs        []dkf.Input `json:"inputs,omitempty"`
	Unresolved    string      `json:"unresolved,omitempty"`
	Method        string      `json:"method,omitempty"`
}

func entryFor(g *store.Graph, a dkf.Assertion) Entry {
	ctx := a.GetContext()
	e := Entry{
		ID: a.ObjectID(), Type: a.ObjectType(), Subject: a.SubjectID(), Content: a.GetContent(), Source: a.GetSource(),
		Timestamp: dkf.FormatTime(a.GetTimestamp()), Confidence: a.GetConfidence(),
		Scope: g.EffectiveScope(a.ObjectID()), Topics: ctx.Topics, Retracted: a.GetRetracted() != nil,
	}
	if c, ok := a.(*dkf.Claim); ok {
		e.Evidential = c.Evidential
	}
	if s, ok := a.(*dkf.Synthesis); ok {
		e.Inputs = s.Inputs
		e.Unresolved = s.Unresolved
		e.Method = s.Method
	}
	return e
}

// newer reports whether a should rank after b as "more recent": greater
// timestamp, ties broken by greater id.
func newer(a, b *dkf.Synthesis) bool {
	if !a.Timestamp.Equal(b.Timestamp) {
		return a.Timestamp.After(b.Timestamp)
	}
	return a.ID > b.ID
}

// CurrentForClass returns the most recent non-retracted synthesis whose
// subject is any member of the class, ordered by timestamp then id, or nil.
func CurrentForClass(g *store.Graph, members []string) *dkf.Synthesis {
	var cur *dkf.Synthesis
	for _, a := range g.ClassAssertions(members) {
		s, ok := a.(*dkf.Synthesis)
		if !ok || s.Retracted != nil {
			continue
		}
		if cur == nil || newer(s, cur) {
			cur = s
		}
	}
	return cur
}

// CurrentSynthesis returns the current synthesis for the class containing
// subject, or nil.
func CurrentSynthesis(g *store.Graph, subject string) *dkf.Synthesis {
	return CurrentForClass(g, g.ClassOf(subject))
}

// classState caches current/reconciled per class root.
type classState struct {
	current    string
	reconciled map[string]bool
}

func stateFor(g *store.Graph, members []string) classState {
	st := classState{reconciled: map[string]bool{}}
	if cur := CurrentForClass(g, members); cur != nil {
		st.current = cur.ID
		closure(g, cur, st.reconciled)
	}
	return st
}

// Recall returns matching assertions in lineage order: every object precedes
// any synthesis that cites it, ties broken by ascending id. When a subject is
// given, every member of its merge equivalence class is included.
func Recall(g *store.Graph, opts RecallOptions) []Entry {
	var candidates []dkf.Assertion
	if opts.Subject != "" {
		candidates = g.ClassAssertions(g.ClassOf(opts.Subject))
	} else {
		candidates = g.SortedAssertions()
	}
	var authorClass map[string]bool
	if opts.Author != "" {
		authorClass = map[string]bool{}
		for _, id := range g.ClassOf(opts.Author) {
			authorClass[id] = true
		}
	}
	relations := map[string][]string{}
	var filtered []dkf.Assertion
	for _, a := range candidates {
		if !opts.IncludeRetracted && a.GetRetracted() != nil {
			continue
		}
		ctx := a.GetContext()
		if opts.Scope != "" && g.EffectiveScope(a.ObjectID()) != opts.Scope {
			continue
		}
		if !hasAllTopics(ctx.Topics, opts.Topics) {
			continue
		}
		if authorClass != nil {
			rels := relationsOf(g, a, authorClass)
			if len(rels) == 0 {
				continue
			}
			relations[a.ObjectID()] = rels
		}
		filtered = append(filtered, a)
	}
	ordered := LineageOrder(filtered)

	states := map[string]classState{} // keyed by class root (first member)
	stateOf := func(subject string) classState {
		members := g.ClassOf(subject)
		key := members[0]
		st, ok := states[key]
		if !ok {
			st = stateFor(g, members)
			states[key] = st
		}
		return st
	}

	if opts.Limit > 0 && len(ordered) > opts.Limit {
		ordered = ordered[len(ordered)-opts.Limit:]
	}
	out := make([]Entry, 0, len(ordered))
	for _, a := range ordered {
		e := entryFor(g, a)
		e.Relations = relations[e.ID]
		st := stateOf(a.SubjectID())
		e.Current = st.current == e.ID
		e.Unsynthesised = !e.Retracted && !e.Current && !st.reconciled[e.ID]
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

// relationsOf reports how an assertion relates to an author class: asserted
// when its source.author resolves into the class, reported when its
// source.document.author does. Both can hold at once — Ben recording his own
// earlier remark — and the two are never collapsed.
func relationsOf(g *store.Graph, a dkf.Assertion, class map[string]bool) []string {
	var rels []string
	src := a.GetSource()
	if p, _ := ResolveAuthor(g, src.Author); p != nil && class[p.ID] {
		rels = append(rels, "asserted")
	}
	if p, _ := ResolveAuthor(g, src.Document.Author); p != nil && class[p.ID] {
		rels = append(rels, "reported")
	}
	return rels
}
