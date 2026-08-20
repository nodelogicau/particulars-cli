package query

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

var ts = time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

type fixture struct {
	t *testing.T
	w *store.Workspace
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Defaults.Source.Author = "ben"
	w, err := store.Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, w: w}
}

func (f *fixture) particular(label string, aliases ...string) *dkf.Particular {
	f.t.Helper()
	p, _, err := f.w.UpsertParticular(dkf.MintURI("", f.w.Config.Workspace.ID, dkf.Slugify(label)), label, aliases)
	if err != nil {
		f.t.Fatal(err)
	}
	return p
}

func (f *fixture) claim(subject, content string, topics ...string) *dkf.Claim {
	f.t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal, Topics: topics}, Timestamp: ts}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func (f *fixture) synthesis(subject, content string, inputs ...dkf.Input) *dkf.Synthesis {
	f.t.Helper()
	s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: subject, Content: content, Inputs: inputs, Unresolved: "none", ProducedBy: dkf.ProducedBy{Harness: "test"}, Method: dkf.DefaultMethod, Timestamp: ts, Context: dkf.Context{Scope: dkf.ScopePersonal}}
	if err := f.w.Create(s); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) retract(id string) {
	f.t.Helper()
	a, err := f.w.Retract(id, &dkf.Retracted{Timestamp: ts, Reason: "test", Source: dkf.Source{Author: "ben"}})
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(a); err != nil {
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

func ids(es []Entry) string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.ID
	}
	return strings.Join(out, " ")
}

func TestResolve(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X", "project_x", "PX")
	f.particular("Auth Service", "auth")
	f.particular("Auth Team", "auth")
	g := f.graph()
	for _, q := range []string{p.ID, p.URI, "project x", "PROJECT_X", "px"} {
		if got := Resolve(g, q); len(got) != 1 || got[0].ID != p.ID {
			t.Errorf("Resolve(%q) = %v", q, got)
		}
	}
	if got := Resolve(g, "auth"); len(got) != 2 {
		t.Errorf("ambiguous alias should return 2, got %d", len(got))
	}
	if got := Resolve(g, "nothing"); len(got) != 0 {
		t.Errorf("expected no match, got %v", got)
	}
}

func TestRecallLineageOrderAndCurrent(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	b := f.claim(p.ID, "B")
	c := f.synthesis(p.ID, "C", in(a.ID, dkf.RoleThesis), in(b.ID, dkf.RoleAntithesis))
	g := f.graph()
	got := Recall(g, RecallOptions{Subject: p.ID})
	if ids(got) != a.ID+" "+b.ID+" "+c.ID {
		t.Errorf("order = %s", ids(got))
	}
	if !got[2].Current || got[0].Current {
		t.Errorf("current flag wrong: %+v", got)
	}
	if len(Recall(g, RecallOptions{Subject: f.particular("Empty").ID})) != 0 {
		t.Error("empty recall should be empty")
	}
	// Limit keeps the most recent.
	if lim := Recall(g, RecallOptions{Subject: p.ID, Limit: 1}); ids(lim) != c.ID {
		t.Errorf("limit = %s", ids(lim))
	}
}

func TestRecallFilters(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	q := f.particular("Project Y")
	a := f.claim(p.ID, "A", "architecture")
	b := f.claim(q.ID, "B", "architecture", "perf")
	c := f.claim(p.ID, "C")
	f.retract(a.ID)
	g := f.graph()

	if got := Recall(g, RecallOptions{Subject: p.ID}); ids(got) != c.ID {
		t.Errorf("retracted should be excluded: %s", ids(got))
	}
	got := Recall(g, RecallOptions{Subject: p.ID, IncludeRetracted: true})
	if ids(got) != a.ID+" "+c.ID || !got[0].Retracted {
		t.Errorf("include retracted: %+v", got)
	}
	if got := Recall(g, RecallOptions{Topics: []string{"architecture"}, IncludeRetracted: true}); ids(got) != a.ID+" "+b.ID {
		t.Errorf("topic across particulars: %s", ids(got))
	}
	if got := Recall(g, RecallOptions{Topics: []string{"architecture", "perf"}}); ids(got) != b.ID {
		t.Errorf("all topics: %s", ids(got))
	}
	if got := Recall(g, RecallOptions{Subject: p.ID, Scope: dkf.ScopePublic}); len(got) != 0 {
		t.Errorf("scope filter: %s", ids(got))
	}
}

func TestLineage(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	b := f.claim(p.ID, "B")
	c := f.synthesis(p.ID, "C", in(a.ID, dkf.RoleThesis), in(b.ID, dkf.RoleThesis))
	e := f.claim(p.ID, "E")
	d := f.synthesis(p.ID, "D", in(c.ID, dkf.RoleThesis), in(e.ID, dkf.RoleAntithesis))
	f.retract(a.ID)
	g := f.graph()

	tree, err := Lineage(g, d.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Inputs) != 2 || tree.Inputs[0].ID != c.ID || tree.Inputs[0].Role != dkf.RoleThesis || tree.Inputs[1].ID != e.ID || tree.Inputs[1].Role != dkf.RoleAntithesis {
		t.Fatalf("level 1 wrong: %+v", tree.Inputs)
	}
	if len(tree.Inputs[0].Inputs) != 2 || !tree.Inputs[0].Inputs[0].Retracted {
		t.Errorf("level 2 wrong: %+v", tree.Inputs[0].Inputs)
	}
	shallow, _ := Lineage(g, d.ID, 1)
	if len(shallow.Inputs) != 0 || !shallow.Truncated {
		t.Errorf("depth 1 should show root only, truncated: %+v", shallow)
	}
	one, _ := Lineage(g, d.ID, 2)
	if len(one.Inputs) != 2 || len(one.Inputs[0].Inputs) != 0 || !one.Inputs[0].Truncated {
		t.Errorf("depth 2 should stop at children: %+v", one.Inputs[0])
	}
	leaf, _ := Lineage(g, a.ID, 0)
	if len(leaf.Inputs) != 0 || !leaf.Retracted {
		t.Errorf("leaf: %+v", leaf)
	}
	if _, err := Lineage(g, "clm_nope", 0); err == nil {
		t.Error("unknown id should error")
	}
}

func TestConflicts(t *testing.T) {
	f := newFixture(t)
	x := f.particular("Project X")
	y := f.particular("Project Y")
	z := f.particular("Project Z")
	solo := f.particular("Solo")

	xa := f.claim(x.ID, "A")
	xb := f.claim(x.ID, "B")
	xz := f.claim(x.ID, "Z never reconciled")
	xc := f.synthesis(x.ID, "C", in(xa.ID, dkf.RoleThesis), in(xb.ID, dkf.RoleAntithesis))
	xd := f.claim(x.ID, "D after synthesis")

	ya := f.claim(y.ID, "A")
	ys := f.synthesis(y.ID, "S", in(ya.ID, dkf.RoleThesis))
	f.retract(ya.ID)

	za := f.claim(z.ID, "A")
	zb := f.claim(z.ID, "B")

	f.claim(solo.ID, "only one")

	g := f.graph()
	reports := Conflicts(g, "")
	byID := map[string]Report{}
	for _, r := range reports {
		byID[r.Particular] = r
	}
	if _, ok := byID[solo.ID]; ok {
		t.Error("single claim without synthesis should not be reported")
	}
	rx := byID[x.ID]
	if rx.Current != xc.ID || strings.Join(rx.Unsynthesised, " ") != xz.ID+" "+xd.ID || len(rx.Stale) != 0 || rx.Priority != 2 {
		t.Errorf("X report: %+v", rx)
	}
	ry := byID[y.ID]
	if ry.Current != ys.ID || len(ry.Unsynthesised) != 0 || strings.Join(ry.Stale, " ") != ys.ID || ry.Priority != 1 {
		t.Errorf("Y report: %+v", ry)
	}
	rz := byID[z.ID]
	if rz.Current != "" || strings.Join(rz.Unsynthesised, " ") != za.ID+" "+zb.ID || rz.Priority != 2 {
		t.Errorf("Z report: %+v", rz)
	}
	// Ordering: priority desc, then id asc.
	if len(reports) != 3 || reports[2].Particular != y.ID {
		t.Errorf("ordering: %v", reports)
	}
	if reports[0].Priority != 2 || reports[1].Priority != 2 || reports[0].Particular > reports[1].Particular {
		t.Errorf("tie-break: %v", reports)
	}
	if one := Conflicts(g, x.ID); len(one) != 1 || one[0].Particular != x.ID {
		t.Errorf("single particular: %v", one)
	}
}

func codes(fs Findings) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Severity+":"+f.Code]++
	}
	return m
}

func TestValidateClean(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	f.synthesis(p.ID, "S", in(a.ID, dkf.RoleThesis))
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("clean workspace should have no findings: %+v", fs)
	}
}

func TestValidateFindings(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	orphan := f.particular("Orphan")
	a := f.claim(p.ID, "A")
	s := f.synthesis(p.ID, "S", in(a.ID, dkf.RoleThesis))
	f.retract(a.ID)

	// Duplicate URI via hand-written particular.
	dup := &dkf.Particular{ID: dkf.NewID(dkf.TypeParticular), URI: p.URI, Label: "Dup"}
	if err := f.w.Create(dup); err != nil {
		t.Fatal(err)
	}
	// Dangling subject + non-canonical ordering, hand-written.
	dangling := dkf.NewID(dkf.TypeClaim)
	_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeClaim), dangling+".yaml"), []byte("type: claim\nid: "+dangling+"\nsubject: par_missing\ncontent: x\nsource:\n  author: ben\ncontext:\n  scope: personal\ntimestamp: 2026-08-20T09:00:00Z\n"), 0o644)
	// Missing unresolved + invalid role + dangling input.
	bad := dkf.NewID(dkf.TypeSynthesis)
	_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeSynthesis), bad+".yaml"), []byte("id: "+bad+"\ntype: synthesis\nsubject: "+p.ID+"\ncontent: x\ninputs:\n  - id: clm_missing\n    role: support\nproduced-by:\n  harness: t\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
	// Cycle: two syntheses citing each other.
	c1, c2 := dkf.NewID(dkf.TypeSynthesis), dkf.NewID(dkf.TypeSynthesis)
	for _, pair := range [][2]string{{c1, c2}, {c2, c1}} {
		_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeSynthesis), pair[0]+".yaml"), []byte("id: "+pair[0]+"\ntype: synthesis\nsubject: "+p.ID+"\ncontent: x\ninputs:\n  - id: "+pair[1]+"\n    role: thesis\nunresolved: n\nproduced-by:\n  harness: t\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
	}

	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	c := codes(fs)
	want := map[string]int{
		"error:duplicate_uri":       2,
		"error:dangling_reference":  2, // subject par_missing, input clm_missing
		"error:missing_field":       1, // unresolved
		"error:invalid_enum":        1, // role support
		"error:cycle":               2,
		"error:index_stale":         1,
		"warning:stale_synthesis":   1,
		"warning:orphan_particular": 2, // Orphan and Dup
		"warning:non_canonical":     2, // dangling claim (key order) + bad synthesis (missing field)
	}
	for k, n := range want {
		if c[k] != n {
			t.Errorf("%s: got %d want %d\nall: %+v", k, c[k], n, fs)
		}
	}
	_ = orphan
	_ = s
	if !fs.HasErrors() {
		t.Error("HasErrors should be true")
	}

	// Missing index → warning only once the other errors are gone.
	g := newFixture(t)
	q := g.particular("Q")
	g.claim(q.ID, "x")
	_ = os.Remove(g.w.IndexPath())
	fs, _ = Validate(g.w)
	if c := codes(fs); c["warning:index_missing"] != 1 || fs.HasErrors() {
		t.Errorf("missing index: %+v", fs)
	}
}

func TestTopics(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	q := f.particular("Project Y")
	a := f.claim(p.ID, "A", "architecture", "db")
	f.claim(q.ID, "B", "architecture")
	f.claim(p.ID, "C")
	f.retract(a.ID)
	g := f.graph()

	got := Topics(g, RecallOptions{})
	if len(got) != 1 || got[0].Topic != "architecture" || got[0].Assertions != 1 || got[0].Particulars != 1 {
		t.Errorf("retracted excluded: %+v", got)
	}
	got = Topics(g, RecallOptions{IncludeRetracted: true})
	if len(got) != 2 || got[0].Topic != "architecture" || got[0].Assertions != 2 || got[0].Particulars != 2 || got[1].Topic != "db" {
		t.Errorf("include retracted: %+v", got)
	}
	got = Topics(g, RecallOptions{Subject: q.ID})
	if len(got) != 1 || got[0].Particulars != 1 {
		t.Errorf("subject filter: %+v", got)
	}
	if got := Topics(g, RecallOptions{Scope: dkf.ScopePublic}); len(got) != 0 {
		t.Errorf("scope filter: %+v", got)
	}
}
