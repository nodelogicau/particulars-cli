package viz

import (
	"strings"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

func (f *fixture) inferred(p *dkf.Particular, content string, basis ...string) *dkf.Claim {
	f.t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: content, Evidential: dkf.EvidentialInferred,
		Basis: basis, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func TestLineageDrawsBasisEdges(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Certs")
	other := f.particular("DNS")
	a := f.claim(p, "renewal A went wrong")
	foreign := f.claim(other, "TTL was a day")
	rule := f.inferred(p, "renewals go wrong when the TTL is long", a.ID, foreign.ID)
	f.synthesis(p, "current", in(rule.ID, dkf.RoleThesis))

	m := Lineage(f.graph(), p, Options{})
	ruleNode := nodeFor(m, rule.ID)
	var basis []Edge
	for _, e := range m.Edges {
		if e.Kind == EdgeBasis {
			basis = append(basis, e)
			if e.To != ruleNode.ID {
				t.Errorf("a basis edge runs from the member to the claim: %+v", e)
			}
		}
	}
	if len(basis) != 2 {
		t.Fatalf("want two basis edges, got %+v", m.Edges)
	}
	if n := nodeFor(m, foreign.ID); n == nil || !n.Foreign {
		t.Errorf("a basis member about another particular is a foreign node: %+v", n)
	}

	dot := DOT(m)
	if !strings.Contains(dot, `style=dotted, label="basis"`) {
		t.Errorf("DOT basis edges are dotted and labelled:\n%s", dot)
	}
	mer := Mermaid(m)
	if !strings.Contains(mer, "-.->|basis|") {
		t.Errorf("Mermaid basis edges are dotted and labelled:\n%s", mer)
	}

	// A basis edge counts as a level.
	shallow := Lineage(f.graph(), p, Options{Depth: 1})
	if nodeFor(shallow, rule.ID) == nil || nodeFor(shallow, foreign.ID) != nil {
		t.Error("depth 1 reaches the rule but not its foreign basis member")
	}
}

func TestStaleRuleIsMarked(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Certs")
	a := f.claim(p, "renewal A went wrong")
	b := f.claim(p, "renewal B went wrong")
	rule := f.inferred(p, "rule", a.ID, b.ID)
	f.retract(a.ID, "")
	m := Lineage(f.graph(), p, Options{})
	if n := nodeFor(m, rule.ID); n == nil || n.State != StateStale {
		t.Errorf("a rule resting on a retracted claim is drawn stale: %+v", n)
	}
}
