// Package viz projects a workspace into a renderer-independent graph model.
//
// The model carries semantics — this synthesis is the current belief, that
// claim is not yet reconciled — and never presentation. Each renderer decides
// what those states look like in its own format, so DOT and Mermaid cannot
// disagree about what the workspace means.
package viz

import (
	"fmt"
	"sort"
	"strings"
)

// Kind is what an object is.
type Kind string

const (
	KindClaim      Kind = "claim"
	KindSynthesis  Kind = "synthesis"
	KindParticular Kind = "particular"
)

// State is how the workspace currently treats an object. A node has exactly
// one, in this order of precedence: retracted, then current, then stale, then
// unsynthesised, then plain.
type State string

const (
	StatePlain         State = "plain"
	StateCurrent       State = "current"
	StateUnsynthesised State = "unsynthesised"
	StateStale         State = "stale"
	StateRetracted     State = "retracted"
)

// EdgeKind distinguishes the relationships that can join two nodes.
type EdgeKind string

const (
	EdgeInput      EdgeKind = "input"      // an assertion cited by a synthesis
	EdgeSuperseded EdgeKind = "superseded" // a retracted object's replacement
	EdgeMerge      EdgeKind = "merge"      // two particulars declared the same
)

// Node is one object in the drawing.
type Node struct {
	ID       string // generated: n1, n2, … — never derived from ObjectID
	ObjectID string
	Kind     Kind
	State    State
	Label    string // the human-readable text, unescaped
	Foreign  bool   // an input about a particular outside the view
	Weight   int    // map view: non-retracted assertions about this particular
	Priority int    // map view: conflict priority
}

// Edge joins two nodes by their generated ids.
type Edge struct {
	From, To string
	Kind     EdgeKind
	Role     string // input role, e.g. "thesis" or "antithesis:qualifying"
}

// View names which projection produced a model, for the --json summary.
type View string

const (
	ViewLineage View = "lineage"
	ViewMap     View = "map"
)

// Model is a complete drawing: nodes and edges in a stable order.
type Model struct {
	View    View
	Subject string // lineage view: the particular's label
	Nodes   []Node
	Edges   []Edge
}

// builder accumulates nodes before ids are assigned, so that ordering never
// depends on Go's deliberately randomised map iteration.
type builder struct {
	view    View
	subject string
	nodes   map[string]*Node // by object id
	edges   []Edge           // object ids until resolved
}

func newBuilder(view View, subject string) *builder {
	return &builder{view: view, subject: subject, nodes: map[string]*Node{}}
}

// node records an object, keeping the strongest state if called twice: a node
// added plainly as somebody's input may later be found to be the current
// belief, and the later, more specific call must win.
func (b *builder) node(objectID string, k Kind, state State, label string) {
	if n, ok := b.nodes[objectID]; ok {
		if rank(state) > rank(n.State) {
			n.State = state
		}
		return
	}
	b.nodes[objectID] = &Node{ObjectID: objectID, Kind: k, State: state, Label: label}
}

func rank(s State) int {
	switch s {
	case StateRetracted:
		return 4
	case StateCurrent:
		return 3
	case StateStale:
		return 2
	case StateUnsynthesised:
		return 1
	}
	return 0
}

func (b *builder) mark(objectID string, f func(*Node)) {
	if n, ok := b.nodes[objectID]; ok {
		f(n)
	}
}

func (b *builder) edge(from, to string, k EdgeKind, role string) {
	b.edges = append(b.edges, Edge{From: from, To: to, Kind: k, Role: role})
}

// build sorts, assigns generated ids, and drops edges whose endpoints are not
// both in the model (an input outside a depth bound, say).
func (b *builder) build() *Model {
	objs := make([]*Node, 0, len(b.nodes))
	for _, n := range b.nodes {
		objs = append(objs, n)
	}
	sort.Slice(objs, func(i, j int) bool { return objs[i].ObjectID < objs[j].ObjectID })

	m := &Model{View: b.view, Subject: b.subject, Nodes: make([]Node, 0, len(objs))}
	id := map[string]string{}
	for i, n := range objs {
		// Sequential ids: unique by construction, valid in every format
		// without escaping, and stable given the sort above. Object ids are
		// never truncated to make one — UUIDv7 ids share a long prefix, so
		// any truncation collapses distinct objects into a single node.
		n.ID = fmt.Sprintf("n%d", i+1)
		id[n.ObjectID] = n.ID
		m.Nodes = append(m.Nodes, *n)
	}

	seen := map[Edge]bool{}
	for _, e := range b.edges {
		from, okF := id[e.From]
		to, okT := id[e.To]
		if !okF || !okT {
			continue
		}
		re := Edge{From: from, To: to, Kind: e.Kind, Role: e.Role}
		if seen[re] {
			continue
		}
		seen[re] = true
		m.Edges = append(m.Edges, re)
	}
	sort.Slice(m.Edges, func(i, j int) bool {
		a, c := m.Edges[i], m.Edges[j]
		if a.From != c.From {
			return less(a.From, c.From)
		}
		if a.To != c.To {
			return less(a.To, c.To)
		}
		if a.Kind != c.Kind {
			return a.Kind < c.Kind
		}
		return a.Role < c.Role
	})
	return m
}

// less orders generated ids numerically ("n2" before "n10"), so the emitted
// order matches the node order rather than sorting lexically.
func less(a, b string) bool {
	na, nb := num(a), num(b)
	if na != nb {
		return na < nb
	}
	return a < b
}

func num(s string) int {
	n := 0
	for _, r := range strings.TrimPrefix(s, "n") {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

// Tag renders an object id compactly for a label: the type prefix, an ellipsis,
// and enough of the tail to find the file. Full ids are 40 characters and
// differ only near the end.
func Tag(objectID string) string {
	prefix, rest, ok := strings.Cut(objectID, "_")
	if !ok || len(rest) <= 6 {
		return objectID
	}
	return prefix + "…" + rest[len(rest)-6:]
}

// Truncate shortens content for a node label at a rune boundary.
func Truncate(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRight(string(r[:n]), " ") + "…"
}

// plural renders a count with its noun, so a map node reads "1 claim" rather
// than "1 claims".
func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
