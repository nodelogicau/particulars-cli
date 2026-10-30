package query

import (
	"fmt"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Node is one element of a provenance tree.
type Node struct {
	ID         string     `json:"id"`
	Type       dkf.Type   `json:"type,omitempty"`
	Subject    string     `json:"subject,omitempty"`
	Content    string     `json:"content,omitempty"`
	Timestamp  string     `json:"timestamp,omitempty"`
	Confidence *float64   `json:"confidence,omitempty"`
	Role       dkf.Role   `json:"role,omitempty"`
	Weight     dkf.Weight `json:"weight,omitempty"`
	Retracted  bool       `json:"retracted"`
	// SupersededBy is the retracted node's superseded-by pointer, when any.
	// It is informational and is never expanded as an input.
	SupersededBy string  `json:"superseded_by,omitempty"`
	Unresolved   string  `json:"unresolved,omitempty"`
	Inputs       []*Node `json:"inputs"`
	// Basis holds an inferred claim's basis members: what it was inferred
	// from, as evidence. Kept apart from Inputs, and carrying no role or
	// weight, because nothing in a basis is a thesis or an antithesis.
	Basis     []*Node `json:"basis"`
	Missing   bool    `json:"missing,omitempty"`   // referenced id has no file
	Truncated bool    `json:"truncated,omitempty"` // depth limit reached before expanding
	Cycle     bool    `json:"cycle,omitempty"`     // id already on the current path
	// Repeated marks an object already expanded elsewhere in the tree: the
	// traversal visits each object once, so a shared ancestor is shown in
	// full at its first occurrence and only named at the others.
	Repeated bool `json:"repeated,omitempty"`
}

// Lineage builds the provenance tree rooted at id, following synthesis
// inputs and inferred claims' bases, and expanding each object at most once.
// depth <= 0 means unlimited. Returns store.ErrNotFound when the root does not
// exist.
func Lineage(g *store.Graph, id string, depth int) (*Node, error) {
	root := g.Assertion(id)
	if root == nil {
		return nil, fmt.Errorf("%s: %w", id, store.ErrNotFound)
	}
	t := &tracer{g: g, path: map[string]bool{}, expanded: map[string]bool{}}
	return t.expand(root, "", "", depth), nil
}

type tracer struct {
	g        *store.Graph
	path     map[string]bool // objects on the current path, for cycles
	expanded map[string]bool // objects already shown in full
}

func (t *tracer) expand(a dkf.Assertion, role dkf.Role, weight dkf.Weight, depth int) *Node {
	n := &Node{
		ID: a.ObjectID(), Type: a.ObjectType(), Subject: a.SubjectID(), Content: a.GetContent(),
		Timestamp: dkf.FormatTime(a.GetTimestamp()), Confidence: a.GetConfidence(),
		Role: role, Weight: weight, Retracted: a.GetRetracted() != nil, Inputs: []*Node{}, Basis: []*Node{},
	}
	if r := a.GetRetracted(); r != nil {
		n.SupersededBy = r.SupersededBy
	}
	t.expanded[n.ID] = true
	var inputs []dkf.Input
	if s, ok := a.(*dkf.Synthesis); ok {
		n.Unresolved = s.Unresolved
		inputs = s.Inputs
	}
	basis := a.BasisIDs()
	if len(inputs) == 0 && len(basis) == 0 {
		return n
	}
	if depth == 1 {
		n.Truncated = true
		return n
	}
	t.path[n.ID] = true
	defer delete(t.path, n.ID)
	for _, in := range inputs {
		w := in.Weight
		if w == "" {
			w = dkf.WeightPrimary
		}
		n.Inputs = append(n.Inputs, t.child(in.ID, in.Role, w, depth))
	}
	for _, id := range basis {
		n.Basis = append(n.Basis, t.child(id, "", "", depth))
	}
	return n
}

// child expands one cited object, or names it when it cannot or need not be
// expanded: missing, on the current path, or already shown.
func (t *tracer) child(id string, role dkf.Role, weight dkf.Weight, depth int) *Node {
	c := t.g.Assertion(id)
	stub := &Node{ID: id, Role: role, Weight: weight, Inputs: []*Node{}, Basis: []*Node{}}
	switch {
	case c == nil:
		stub.Missing = true
	case t.path[id]:
		stub.Type, stub.Cycle = c.ObjectType(), true
	case t.expanded[id]:
		stub.Type, stub.Repeated = c.ObjectType(), true
	default:
		return t.expand(c, role, weight, depth-1)
	}
	return stub
}
