package viz

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

var ts = time.Date(2026, 8, 24, 9, 0, 0, 0, time.UTC)

type fixture struct {
	t *testing.T
	w *store.Workspace
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	w, err := store.Init(filepath.Join(t.TempDir(), "ws"), store.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, w: w}
}

func (f *fixture) particular(label string) *dkf.Particular {
	f.t.Helper()
	p, _, err := f.w.UpsertParticular(dkf.MintURI("", f.w.Config.Workspace.ID, dkf.Slugify(label)), label, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fixture) claim(p *dkf.Particular, content string) *dkf.Claim {
	f.t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: content,
		Evidential: dkf.EvidentialObserved, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func (f *fixture) scopedClaim(p *dkf.Particular, content string, sc dkf.Scope) *dkf.Claim {
	f.t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: content,
		Evidential: dkf.EvidentialObserved, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: sc}, Timestamp: ts}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func (f *fixture) synthesis(p *dkf.Particular, content string, inputs ...dkf.Input) *dkf.Synthesis {
	f.t.Helper()
	s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: p.ID, Content: content, Inputs: inputs,
		Unresolved: "None identified", Source: dkf.Source{Harness: "test"}, Method: dkf.DefaultMethod,
		Timestamp: ts, Context: dkf.Context{Scope: dkf.ScopePersonal}}
	if err := f.w.Create(s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) retract(id, supersededBy string) {
	f.t.Helper()
	r := &dkf.Retracted{Timestamp: ts, Reason: "test", Source: dkf.Source{Author: "ben"}, SupersededBy: supersededBy}
	if _, err := f.w.Retract(id, r); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) graph() *store.Graph {
	f.t.Helper()
	g, err := f.w.Load()
	if err != nil {
		f.t.Fatal(err)
	}
	return g
}

func in(id string, role dkf.Role) dkf.Input {
	return dkf.Input{ID: id, Role: role, Weight: dkf.WeightPrimary}
}

func nodeFor(m *Model, objectID string) *Node {
	for i := range m.Nodes {
		if m.Nodes[i].ObjectID == objectID {
			return &m.Nodes[i]
		}
	}
	return nil
}

func TestNodeIdsAreUniqueAndNotTruncations(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	// Minted back to back: UUIDv7 ids share a long prefix, which is what a
	// naive truncation collapses into one node.
	var ids []string
	for i := 0; i < 6; i++ {
		ids = append(ids, f.claim(p, "claim "+string(rune('a'+i))).ID)
	}
	m := Lineage(f.graph(), p, Options{})
	if len(m.Nodes) != 6 {
		t.Fatalf("want 6 nodes, got %d", len(m.Nodes))
	}
	seenID, seenObj := map[string]bool{}, map[string]bool{}
	for _, n := range m.Nodes {
		if seenID[n.ID] {
			t.Errorf("duplicate node id %q", n.ID)
		}
		seenID[n.ID] = true
		seenObj[n.ObjectID] = true
		if strings.HasPrefix(n.ID, n.ObjectID[:8]) {
			t.Errorf("node id %q looks derived from the object id", n.ID)
		}
	}
	for _, id := range ids {
		if !seenObj[id] {
			t.Errorf("%s missing from the model", id)
		}
	}
}

func TestDeterministicAcrossRuns(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a, b := f.claim(p, "alpha"), f.claim(p, "beta")
	f.synthesis(p, "resolved", in(a.ID, dkf.RoleThesis), in(b.ID, dkf.RoleAntithesis))
	f.particular("Other Thing")
	g := f.graph()
	for _, view := range []struct {
		name string
		fn   func() *Model
	}{
		{"lineage", func() *Model { return Lineage(g, p, Options{}) }},
		{"map", func() *Model { return Map(g, Options{}) }},
	} {
		for _, render := range []struct {
			name string
			fn   func(*Model) string
		}{{"dot", DOT}, {"mermaid", Mermaid}} {
			first := render.fn(view.fn())
			for i := 0; i < 5; i++ {
				if got := render.fn(view.fn()); got != first {
					t.Fatalf("%s/%s is not deterministic on run %d", view.name, render.name, i+2)
				}
			}
		}
	}
}

func TestLineageRolesStatesAndForeignInputs(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	other := f.particular("Library Y")
	thesis := f.claim(p, "monolith since 2024")
	anti := f.claim(p, "billing split out again")
	foreign := f.claim(other, "library Y went 2.0")
	cur := f.synthesis(p, "consolidated, billing excepted",
		in(thesis.ID, dkf.RoleThesis), in(anti.ID, dkf.RoleAntithesis),
		dkf.Input{ID: foreign.ID, Role: dkf.RoleThesis, Weight: dkf.WeightQualifying})
	loose := f.claim(p, "not reconciled into anything")

	m := Lineage(f.graph(), p, Options{})

	if n := nodeFor(m, cur.ID); n == nil || n.State != StateCurrent {
		t.Errorf("current synthesis state: %+v", n)
	}
	if n := nodeFor(m, loose.ID); n == nil || n.State != StateUnsynthesised {
		t.Errorf("loose claim state: %+v", n)
	}
	if n := nodeFor(m, foreign.ID); n == nil || !n.Foreign {
		t.Errorf("cross-particular input should be marked foreign: %+v", n)
	}
	if n := nodeFor(m, thesis.ID); n == nil || n.Foreign {
		t.Errorf("same-particular input must not be foreign: %+v", n)
	}
	roles := map[string]string{}
	for _, e := range m.Edges {
		for _, n := range m.Nodes {
			if n.ID == e.From {
				roles[n.ObjectID] = e.Role
			}
		}
	}
	if roles[thesis.ID] != "thesis" || roles[anti.ID] != "antithesis" {
		t.Errorf("roles on edges: %+v", roles)
	}
	if roles[foreign.ID] != "thesis:qualifying" {
		t.Errorf("qualifying weight should show on the edge: %q", roles[foreign.ID])
	}
}

func TestMergedParticularsShareOneView(t *testing.T) {
	f := newFixture(t)
	a, b := f.particular("Project X"), f.particular("ProjectX")
	ca, cb := f.claim(a, "about a"), f.claim(b, "about b")
	if _, err := f.w.CreateMerge(a.URI, b.URI, "same", dkf.Source{Author: "ben"}, ts); err != nil {
		t.Fatal(err)
	}
	g := f.graph()
	for _, subject := range []*dkf.Particular{a, b} {
		m := Lineage(g, subject, Options{})
		if nodeFor(m, ca.ID) == nil || nodeFor(m, cb.ID) == nil {
			t.Errorf("from %s: both members' claims should appear (%d nodes)", subject.Label, len(m.Nodes))
		}
	}
}

func TestRetractedExcludedByDefault(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	old := f.claim(p, "the old belief")
	replacement := f.claim(p, "the corrected belief")
	f.retract(old.ID, replacement.ID)

	m := Lineage(f.graph(), p, Options{})
	if nodeFor(m, old.ID) != nil {
		t.Error("retracted claim should be omitted by default")
	}
	m = Lineage(f.graph(), p, Options{IncludeRetracted: true})
	n := nodeFor(m, old.ID)
	if n == nil || n.State != StateRetracted {
		t.Fatalf("with --include-retracted the claim should appear as retracted: %+v", n)
	}
	var found bool
	for _, e := range m.Edges {
		if e.Kind == EdgeSuperseded && e.From == n.ID {
			found = true
			if e.Kind == EdgeInput {
				t.Error("superseded-by must not be an input edge")
			}
		}
	}
	if !found {
		t.Error("superseded-by should be rendered as its own edge kind")
	}
}

func TestDepthBoundsTheView(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	l1 := f.claim(p, "level one")
	s1 := f.synthesis(p, "first", in(l1.ID, dkf.RoleThesis))
	s2 := f.synthesis(p, "second", in(s1.ID, dkf.RoleThesis))
	s3 := f.synthesis(p, "third", in(s2.ID, dkf.RoleThesis))

	m := Lineage(f.graph(), p, Options{Depth: 1})
	if nodeFor(m, s3.ID) == nil || nodeFor(m, s2.ID) == nil {
		t.Error("depth 1 should reach the current belief and its direct input")
	}
	if nodeFor(m, l1.ID) != nil {
		t.Error("depth 1 should not reach three levels down")
	}
	if m2 := Lineage(f.graph(), p, Options{}); nodeFor(m2, l1.ID) == nil {
		t.Error("unbounded depth should reach the whole closure")
	}
}

func TestScopeFilters(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	personal := f.claim(p, "a personal note")
	org := f.scopedClaim(p, "an organisation fact", dkf.ScopeOrganisation)

	// Personal is included by default: a diagram that omitted it would
	// misrepresent the reasoning.
	m := Lineage(f.graph(), p, Options{})
	if nodeFor(m, personal.ID) == nil {
		t.Error("personal knowledge should be drawn by default")
	}
	m = Lineage(f.graph(), p, Options{Scope: dkf.ScopeOrganisation})
	if nodeFor(m, personal.ID) != nil {
		t.Error("--scope organisation should exclude personal")
	}
	if nodeFor(m, org.ID) == nil {
		t.Error("--scope organisation should keep organisation")
	}
}

func TestMapView(t *testing.T) {
	f := newFixture(t)
	a, b := f.particular("Project X"), f.particular("ProjectX")
	orphan := f.particular("Never Discussed")
	f.claim(a, "one")
	f.claim(a, "two")
	if _, err := f.w.CreateMerge(a.URI, b.URI, "same", dkf.Source{Author: "ben"}, ts); err != nil {
		t.Fatal(err)
	}
	m := Map(f.graph(), Options{})
	if len(m.Nodes) != 3 {
		t.Fatalf("want a node per particular, got %d", len(m.Nodes))
	}
	if n := nodeFor(m, orphan.ID); n == nil {
		t.Error("a particular with no claims must still appear")
	}
	na := nodeFor(m, a.ID)
	if na == nil || na.Weight != 2 {
		t.Errorf("weight should count non-retracted assertions: %+v", na)
	}
	if na.Priority == 0 {
		t.Errorf("unreconciled claims should give a non-zero priority: %+v", na)
	}
	var merges int
	for _, e := range m.Edges {
		if e.Kind == EdgeMerge {
			merges++
		}
	}
	if merges != 1 {
		t.Errorf("want one merge edge, got %d", merges)
	}
}

func TestTruncatedLabelsCarryTooltips(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	long := strings.TrimSpace(strings.Repeat("all work and no play ", 5))
	c := f.claim(p, long)
	brief := f.claim(p, "fits in full")

	m := Lineage(f.graph(), p, Options{})
	n := nodeFor(m, c.ID)
	if n.Tooltip != long {
		t.Errorf("truncated node should carry the full text: %q", n.Tooltip)
	}
	if !strings.HasSuffix(n.Label, "…") {
		t.Errorf("label should still be truncated: %q", n.Label)
	}
	if b := nodeFor(m, brief.ID); b.Tooltip != "" {
		t.Errorf("untruncated node must not carry a tooltip: %q", b.Tooltip)
	}

	mm := Mermaid(m)
	if want := fmt.Sprintf("  click %s callback \"%s\"\n", n.ID, long); !strings.Contains(mm, want) {
		t.Errorf("mermaid should attach the full text as a tooltip:\n%s", mm)
	}
	if strings.Contains(mm, "click "+nodeFor(m, brief.ID).ID+" ") {
		t.Errorf("mermaid must not emit a click line for an untruncated label:\n%s", mm)
	}

	d := DOT(m)
	if want := fmt.Sprintf("tooltip=\"%s\"", long); !strings.Contains(d, want) {
		t.Errorf("DOT should attach the full text as a tooltip:\n%s", d)
	}
	for _, line := range strings.Split(d, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), nodeFor(m, brief.ID).ID+" [") && strings.Contains(line, "tooltip=") {
			t.Errorf("DOT must not emit a tooltip for an untruncated label: %s", line)
		}
	}
}

// A label containing every character that means something to one of the two
// formats. Neither renderer may let it break out of its delimiter.
const hostile = `He said "stop" <now> | ` + "`code`" + ` & {braces} [brackets] \back\ 100% —— ünïcödé`

func TestRenderersEscapeHostileLabels(t *testing.T) {
	f := newFixture(t)
	p := f.particular(hostile)
	c := f.claim(p, hostile)
	f.synthesis(p, hostile, in(c.ID, dkf.RoleThesis))
	g := f.graph()

	for _, m := range []*Model{Lineage(g, p, Options{}), Map(g, Options{})} {
		d := DOT(m)
		if strings.Count(d, `"`)%2 != 0 {
			t.Errorf("DOT has unbalanced quotes:\n%s", d)
		}
		for _, line := range strings.Split(d, "\n") {
			if strings.Contains(line, "label=") && !strings.Contains(line, `label="`) {
				t.Errorf("DOT label not quoted: %s", line)
			}
		}
		mm := Mermaid(m)
		for _, line := range strings.Split(mm, "\n") {
			if !strings.Contains(line, `["`) && !strings.Contains(line, `("`) {
				continue
			}
			// Exactly the opening and closing quote: any third would end the
			// label early and leave the rest as syntax.
			if strings.Count(line, `"`) != 2 {
				t.Errorf("mermaid label does not have exactly two quotes: %s", line)
			}
		}
		// A tooltip string has the same two-quote budget as a label.
		for _, line := range strings.Split(mm, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "click ") && strings.Count(line, `"`) != 2 {
				t.Errorf("mermaid tooltip does not have exactly two quotes: %s", line)
			}
		}
		// A pipe delimits an edge label, so on an edge line they must pair up;
		// inside a node label it is escaped and must not appear raw at all.
		for _, line := range strings.Split(mm, "\n") {
			switch {
			case strings.Contains(line, "-->"), strings.Contains(line, "-.->"), strings.Contains(line, "---"):
				if strings.Count(line, "|")%2 != 0 {
					t.Errorf("mermaid edge label has an unbalanced pipe: %s", line)
				}
			case strings.Contains(line, `["`), strings.Contains(line, `("`):
				if strings.Contains(line, "|") {
					t.Errorf("mermaid node label carries a raw pipe: %s", line)
				}
			}
		}
	}
}

func TestBothRenderersAgreeOnState(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p, "alpha")
	cur := f.synthesis(p, "current belief", in(a.ID, dkf.RoleThesis))
	loose := f.claim(p, "loose end")
	gone := f.claim(p, "withdrawn")
	f.retract(gone.ID, "")

	m := Lineage(f.graph(), p, Options{IncludeRetracted: true})
	d, mm := DOT(m), Mermaid(m)

	id := func(objectID string) string { return nodeFor(m, objectID).ID }
	// DOT carries state as attributes on the node's own line; Mermaid as a
	// class assignment. Both must name the same node for the same state.
	for _, tc := range []struct{ object, dotAttr, mermaidClass string }{
		{cur.ID, "penwidth=3", "current"},
		{loose.ID, "dashed", "unsynthesised"},
		{gone.ID, "dotted", "retracted"},
	} {
		n := id(tc.object)
		var dotLine string
		for _, line := range strings.Split(d, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), n+" [") {
				dotLine = line
			}
		}
		if !strings.Contains(dotLine, tc.dotAttr) {
			t.Errorf("DOT: node %s (%s) missing %q: %s", n, tc.mermaidClass, tc.dotAttr, dotLine)
		}
		var classed bool
		for _, line := range strings.Split(mm, "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "class ") || !strings.HasSuffix(line, " "+tc.mermaidClass) {
				continue
			}
			for _, member := range strings.Split(strings.Fields(line)[1], ",") {
				if member == n {
					classed = true
				}
			}
		}
		if !classed {
			t.Errorf("mermaid: node %s not in class %s", n, tc.mermaidClass)
		}
	}
}

func TestRenderedOutputShape(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	f.claim(p, "something")
	g := f.graph()
	if d := DOT(Lineage(g, p, Options{})); !strings.HasPrefix(d, "digraph particulars {") || !strings.HasSuffix(d, "}\n") {
		t.Errorf("DOT envelope: %q", d)
	}
	if mm := Mermaid(Map(g, Options{})); !strings.HasPrefix(mm, "flowchart BT\n") {
		t.Errorf("mermaid envelope: %q", mm)
	}
	// Tag keeps enough of an id to find the file, without being a prefix that
	// several objects would share.
	if got := Tag("clm_01a027d3-37a8-78c6-b0c1-7f26810848ae"); got != "clm…0848ae" {
		t.Errorf("Tag: %q", got)
	}
	if got := Truncate("a  b\nc", 40); got != "a b c" {
		t.Errorf("Truncate should collapse whitespace: %q", got)
	}
	if got := Truncate(strings.Repeat("x", 50), 10); len([]rune(got)) != 11 {
		t.Errorf("Truncate should cut to n runes plus the ellipsis: %q", got)
	}
}
