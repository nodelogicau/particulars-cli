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
	Unresolved string     `json:"unresolved,omitempty"`
	Inputs     []*Node    `json:"inputs"`
	Missing    bool       `json:"missing,omitempty"`   // referenced id has no file
	Truncated  bool       `json:"truncated,omitempty"` // depth limit reached before expanding
	Cycle      bool       `json:"cycle,omitempty"`     // id already on the current path
}

// Lineage builds the provenance tree rooted at id. depth <= 0 means
// unlimited. Returns store.ErrNotFound when the root does not exist.
func Lineage(g *store.Graph, id string, depth int) (*Node, error) {
	root := g.Assertion(id)
	if root == nil {
		return nil, fmt.Errorf("%s: %w", id, store.ErrNotFound)
	}
	return expand(g, root, "", "", depth, map[string]bool{}), nil
}

func expand(g *store.Graph, a dkf.Assertion, role dkf.Role, weight dkf.Weight, depth int, path map[string]bool) *Node {
	n := &Node{
		ID: a.ObjectID(), Type: a.ObjectType(), Subject: a.SubjectID(), Content: a.GetContent(),
		Timestamp: dkf.FormatTime(a.GetTimestamp()), Confidence: a.GetConfidence(),
		Role: role, Weight: weight, Retracted: a.GetRetracted() != nil, Inputs: []*Node{},
	}
	s, ok := a.(*dkf.Synthesis)
	if !ok {
		return n
	}
	n.Unresolved = s.Unresolved
	if depth == 1 {
		n.Truncated = len(s.Inputs) > 0
		return n
	}
	path[s.ID] = true
	defer delete(path, s.ID)
	for _, in := range s.Inputs {
		w := in.Weight
		if w == "" {
			w = dkf.WeightPrimary
		}
		child := g.Assertion(in.ID)
		switch {
		case child == nil:
			n.Inputs = append(n.Inputs, &Node{ID: in.ID, Role: in.Role, Weight: w, Missing: true, Inputs: []*Node{}})
		case path[in.ID]:
			n.Inputs = append(n.Inputs, &Node{ID: in.ID, Type: child.ObjectType(), Role: in.Role, Weight: w, Cycle: true, Inputs: []*Node{}})
		default:
			n.Inputs = append(n.Inputs, expand(g, child, in.Role, w, depth-1, path))
		}
	}
	return n
}
