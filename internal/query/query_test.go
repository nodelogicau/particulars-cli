package query

import (
	"bytes"
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
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content, Evidential: dkf.EvidentialObserved, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal, Topics: topics}, Timestamp: ts}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func (f *fixture) synthesis(subject, content string, inputs ...dkf.Input) *dkf.Synthesis {
	return f.synthesisAt(subject, content, ts, inputs...)
}

func (f *fixture) synthesisScoped(subject, content string, sc dkf.Scope, inputs ...dkf.Input) *dkf.Synthesis {
	f.t.Helper()
	s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: subject, Content: content, Inputs: inputs,
		Unresolved: "None identified", Source: dkf.Source{Harness: "test"}, Method: dkf.DefaultMethod,
		Timestamp: ts, Context: dkf.Context{Scope: sc}}
	if err := f.w.Create(s); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) synthesisAt(subject, content string, at time.Time, inputs ...dkf.Input) *dkf.Synthesis {
	f.t.Helper()
	s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: subject, Content: content, Inputs: inputs, Unresolved: "None identified", Source: dkf.Source{Harness: "test"}, Method: dkf.DefaultMethod, Timestamp: at, Context: dkf.Context{Scope: dkf.ScopePersonal}}
	if err := f.w.Create(s); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) merge(a, b string) *dkf.Merge {
	f.t.Helper()
	m, err := f.w.CreateMerge(a, b, "", dkf.Source{Author: "ben"}, ts)
	if err != nil {
		f.t.Fatal(err)
	}
	return m
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
	f.particular("Ben", "ben") // the claims' author resolves; nothing is unresolved
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
	_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeSynthesis), bad+".yaml"), []byte("id: "+bad+"\ntype: synthesis\nsubject: "+p.ID+"\ncontent: x\ninputs:\n  - id: clm_missing\n    role: support\nsource:\n  harness: t\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
	// Cycle: two syntheses citing each other.
	c1, c2 := dkf.NewID(dkf.TypeSynthesis), dkf.NewID(dkf.TypeSynthesis)
	for _, pair := range [][2]string{{c1, c2}, {c2, c1}} {
		_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeSynthesis), pair[0]+".yaml"), []byte("id: "+pair[0]+"\ntype: synthesis\nsubject: "+p.ID+"\ncontent: x\ninputs:\n  - id: "+pair[1]+"\n    role: thesis\nunresolved: n\nsource:\n  harness: t\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
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

func TestCurrentByTimestampAndTransitiveStale(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	s1 := f.synthesisAt(p.ID, "S1", ts.Add(48*time.Hour), in(a.ID, dkf.RoleThesis))
	s2 := f.synthesisAt(p.ID, "S2 minted later but older", ts, in(a.ID, dkf.RoleThesis))
	g := f.graph()
	if cur := CurrentSynthesis(g, p.ID); cur == nil || cur.ID != s1.ID {
		t.Errorf("current should be s1 by timestamp, got %v", cur)
	}
	r := Analyse(g, p)
	if r.Current != s1.ID || strings.Join(r.Unsynthesised, " ") != s2.ID {
		t.Errorf("analyse: %+v", r)
	}
	// Transitive stale: d cites c cites a; retract a.
	c := f.synthesisAt(p.ID, "C", ts.Add(72*time.Hour), in(a.ID, dkf.RoleThesis))
	d := f.synthesisAt(p.ID, "D", ts.Add(96*time.Hour), in(c.ID, dkf.RoleThesis))
	f.retract(a.ID)
	g = f.graph()
	r = Analyse(g, p)
	stale := strings.Join(r.Stale, " ")
	for _, id := range []string{s1.ID, s2.ID, c.ID, d.ID} {
		if !strings.Contains(stale, id) {
			t.Errorf("%s should be stale: %v", id, r.Stale)
		}
	}
	rec := Recall(g, RecallOptions{Subject: p.ID})
	for _, e := range rec {
		switch e.ID {
		case d.ID:
			if !e.Current || e.Unsynthesised {
				t.Errorf("d flags: %+v", e)
			}
		case c.ID:
			if e.Unsynthesised {
				t.Errorf("c is reconciled into d: %+v", e)
			}
		case s1.ID, s2.ID:
			if !e.Unsynthesised {
				t.Errorf("%s should be unsynthesised: %+v", e.ID, e)
			}
		}
		if e.Source.Harness != "test" {
			t.Errorf("source missing on entry: %+v", e)
		}
	}
}

func TestMergeClasses(t *testing.T) {
	f := newFixture(t)
	a := f.particular("A")
	b := f.particular("B")
	y := f.particular("Library Y")
	ca := f.claim(a.ID, "about A")
	cb := f.claim(b.ID, "about B")
	cy := f.claim(y.ID, "about Y")
	f.claim(y.ID, "also about Y")
	sa := f.synthesis(a.ID, "S", in(ca.ID, dkf.RoleThesis), in(cy.ID, dkf.RoleThesis))
	m := f.merge(a.URI, b.URI)
	g := f.graph()

	rec := Recall(g, RecallOptions{Subject: a.ID})
	if ids(rec) != ca.ID+" "+cb.ID+" "+sa.ID {
		t.Errorf("recall across merge: %s", ids(rec))
	}
	if rec[1].Subject != b.ID || !rec[1].Unsynthesised {
		t.Errorf("B's claim keeps its subject and is unsynthesised: %+v", rec[1])
	}
	r := Analyse(g, a)
	if r.Current != sa.ID || strings.Join(r.Unsynthesised, " ") != cb.ID || len(r.Members) != 2 {
		t.Errorf("conflicts across merge: %+v", r)
	}
	// Cross-particular input does not synthesise Y.
	ry := Analyse(g, y)
	if len(ry.Unsynthesised) != 2 || ry.Current != "" {
		t.Errorf("Y should have two unsynthesised: %+v", ry)
	}
	// Sweep reports the class once.
	all := Conflicts(g, "")
	seen := 0
	for _, rep := range all {
		if rep.Particular == a.ID || rep.Particular == b.ID {
			seen++
			if rep.Particular != a.ID {
				t.Errorf("class should be keyed by lowest id: %s", rep.Particular)
			}
		}
	}
	if seen != 1 {
		t.Errorf("class reported %d times", seen)
	}
	// Retract the merge → classes split.
	f.retract(m.ID)
	g = f.graph()
	if got := Recall(g, RecallOptions{Subject: a.ID}); ids(got) != ca.ID+" "+sa.ID {
		t.Errorf("after merge retraction: %s", ids(got))
	}
}

func TestLineageSupersededBy(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	y := f.claim(p.ID, "Y corrected")
	s := f.synthesis(p.ID, "S", in(a.ID, dkf.RoleThesis))
	if _, err := f.w.Retract(a.ID, &dkf.Retracted{Timestamp: ts, Reason: "typo", Source: dkf.Source{Author: "ben"}, SupersededBy: y.ID}); err != nil {
		t.Fatal(err)
	}
	g := f.graph()
	tree, _ := Lineage(g, s.ID, 0)
	if n := tree.Inputs[0]; !n.Retracted || n.SupersededBy != y.ID || len(n.Inputs) != 0 {
		t.Errorf("superseded node: %+v", n)
	}
}

func TestValidateNewFindings(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	f.synthesis(p.ID, "S", in(a.ID, dkf.RoleThesis))
	// Legacy produced-by synthesis and a legacy id, hand-written.
	legacy := "syn_01a0legacy"
	_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeSynthesis), legacy+".yaml"), []byte("id: "+legacy+"\ntype: synthesis\nsubject: "+p.ID+"\ncontent: x\ninputs:\n  - id: "+a.ID+"\n    role: thesis\nunresolved: None identified\nproduced-by:\n  harness: claude\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
	both := dkf.NewID(dkf.TypeSynthesis)
	_ = os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeSynthesis), both+".yaml"), []byte("id: "+both+"\ntype: synthesis\nsubject: "+p.ID+"\ncontent: x\ninputs:\n  - id: "+a.ID+"\n    role: thesis\nunresolved: n\nsource:\n  harness: claude\nproduced-by:\n  harness: claude\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
	// Merge to a foreign uri, twice.
	f.merge(p.URI, "https://www.wikidata.org/entity/Q1")
	dup := &dkf.Merge{ID: dkf.NewID(dkf.TypeMerge), URIs: []string{"https://www.wikidata.org/entity/Q1", p.URI}, Source: dkf.Source{Author: "ben"}, Timestamp: ts}
	if err := f.w.Create(dup); err != nil {
		t.Fatal(err)
	}
	_, _ = f.w.RebuildIndex()
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	c := codes(fs)
	want := map[string]int{
		"warning:legacy_produced_by":   1,
		"warning:legacy_id":            1,
		"error:conflicting_provenance": 1,
		"warning:unknown_merge_uri":    2,
		"warning:duplicate_merge":      2,
	}
	for k, n := range want {
		if c[k] != n {
			t.Errorf("%s: got %d want %d\nall: %+v", k, c[k], n, fs)
		}
	}
	if c["warning:non_canonical"] != 1 { // only the `both` file; the legacy one is suppressed
		t.Errorf("non_canonical: %+v", fs)
	}
}

func TestScopeWiderThanInputsWarning(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	scoped := func(content string, sc dkf.Scope) *dkf.Claim {
		c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: content,
			Evidential: dkf.EvidentialObserved, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: sc}, Timestamp: ts}
		if err := f.w.Create(c); err != nil {
			t.Fatal(err)
		}
		if err := f.w.UpsertIndex(c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	synth := func(content string, sc dkf.Scope, inputs ...dkf.Input) *dkf.Synthesis {
		s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: p.ID, Content: content, Inputs: inputs,
			Unresolved: "None identified", Source: dkf.Source{Harness: "test"}, Method: dkf.DefaultMethod,
			Timestamp: ts, Context: dkf.Context{Scope: sc}}
		if err := f.w.Create(s); err != nil {
			t.Fatal(err)
		}
		if err := f.w.UpsertIndex(s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	priv := scoped("personal input", dkf.ScopePersonal)
	org := scoped("organisation input", dkf.ScopeOrganisation)

	wider := synth("summarises the personal one", dkf.ScopeOrganisation, in(priv.ID, dkf.RoleThesis), in(org.ID, dkf.RoleThesis))
	same := synth("only organisation inputs", dkf.ScopeOrganisation, in(org.ID, dkf.RoleThesis))
	narrower := synth("personal synthesis of an organisation claim", dkf.ScopePersonal, in(org.ID, dkf.RoleThesis))
	chained := synth("public, citing the organisation synthesis", dkf.ScopePublic, in(same.ID, dkf.RoleThesis))
	_, _ = f.w.RebuildIndex()

	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	flagged := map[string]string{}
	for _, fi := range fs {
		if fi.Code == CodeScopeWiderThanInputs {
			if fi.Severity != SeverityWarning {
				t.Errorf("must be a warning, got %s", fi.Severity)
			}
			flagged[fi.Path] = fi.Message
		}
	}
	g, _ := f.w.Load()
	for _, want := range []*dkf.Synthesis{wider, chained} {
		if _, ok := flagged[g.Files[want.ID]]; !ok {
			t.Errorf("%s should be flagged", want.ID)
		}
	}
	for _, notWant := range []*dkf.Synthesis{same, narrower} {
		if _, ok := flagged[g.Files[notWant.ID]]; ok {
			t.Errorf("%s should not be flagged", notWant.ID)
		}
	}
	if msg := flagged[g.Files[wider.ID]]; !strings.Contains(msg, priv.ID) || !strings.Contains(msg, "personal") || strings.Contains(msg, org.ID) {
		t.Errorf("message should name only the narrower input: %s", msg)
	}
	if fs.HasErrors() {
		t.Errorf("this must never be an error: %+v", fs)
	}
	// retracted syntheses are not flagged
	f.retract(wider.ID)
	fs, _ = Validate(f.w)
	for _, fi := range fs {
		if fi.Code == CodeScopeWiderThanInputs && strings.Contains(fi.Path, wider.ID) {
			t.Error("retracted synthesis should not be flagged")
		}
	}
}

// promote is a fixture helper: write a promotion covering ids at scope.
func (f *fixture) promote(scope dkf.Scope, ids ...string) *dkf.Promotion {
	f.t.Helper()
	pr, err := f.w.CreatePromotion(ids, scope, "test", dkf.Source{Author: "ben"}, ts)
	if err != nil {
		f.t.Fatal(err)
	}
	return pr
}

func TestScopeWiderThanInputsUsesEffectiveScope(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	priv := f.claim(p.ID, "a personal observation")
	if err := f.w.Create(&dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: "an organisation fact",
		Evidential: dkf.EvidentialObserved, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopeOrganisation}, Timestamp: ts}); err != nil {
		t.Fatal(err)
	}
	s := f.synthesisScoped(p.ID, "reconciled", dkf.ScopeOrganisation, in(priv.ID, dkf.RoleThesis))
	_, _ = f.w.RebuildIndex()

	warned := func() string {
		g, err := f.w.Load()
		if err != nil {
			t.Fatal(err)
		}
		return ScopeWiderThanInputs(g, g.Assertion(s.ID).(*dkf.Synthesis))
	}

	// Asserted organisation over an asserted personal input: warned.
	if warned() == "" {
		t.Fatal("an organisation synthesis over a personal input should warn")
	}

	// Promoting the input to match clears it — neither file changed.
	before, err := os.ReadFile(filepath.Join(f.w.Root, "syntheses", s.ID+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	f.promote(dkf.ScopeOrganisation, priv.ID)
	if msg := warned(); msg != "" {
		t.Errorf("promoting the input should clear the warning: %s", msg)
	}
	after, _ := os.ReadFile(filepath.Join(f.w.Root, "syntheses", s.ID+".yaml"))
	if !bytes.Equal(before, after) {
		t.Error("the synthesis file must not have changed")
	}

	// Promoting the synthesis past its inputs creates it again, and the
	// message names the promotion responsible rather than only the scope.
	pr := f.promote(dkf.ScopePublic, s.ID)
	msg := warned()
	if msg == "" {
		t.Fatal("promoting the synthesis past its inputs should warn")
	}
	if !strings.Contains(msg, pr.ID) {
		t.Errorf("the message should name the promotion that widened it: %s", msg)
	}
	if !strings.Contains(msg, "public") || !strings.Contains(msg, "organisation") {
		t.Errorf("the message should name both effective scopes: %s", msg)
	}

	// A retracted synthesis is never warned about.
	if _, err := f.w.Retract(s.ID, &dkf.Retracted{Timestamp: ts, Reason: "x", Source: dkf.Source{Author: "ben"}}); err != nil {
		t.Fatal(err)
	}
	if msg := warned(); msg != "" {
		t.Errorf("a retracted synthesis should not warn: %s", msg)
	}
}

func TestPromotionValidationFindings(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "one")
	b := f.claim(p.ID, "two")
	gone := f.claim(p.ID, "withdrawn")
	f.promote(dkf.ScopeOrganisation, a.ID)
	f.promote(dkf.ScopeOrganisation, a.ID, b.ID) // duplicate for a, not for b
	f.promote(dkf.ScopePublic, gone.ID)
	if _, err := f.w.Retract(gone.ID, &dkf.Retracted{Timestamp: ts, Reason: "x", Source: dkf.Source{Author: "ben"}}); err != nil {
		t.Fatal(err)
	}
	_, _ = f.w.RebuildIndex()

	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]int{}
	for _, fi := range fs {
		codes[fi.Code]++
		if fi.Code == CodeDuplicatePromotion || fi.Code == CodePromotionOfRetracted {
			if fi.Severity != SeverityWarning {
				t.Errorf("%s must be a warning", fi.Code)
			}
		}
	}
	if codes[CodeDuplicatePromotion] != 2 {
		t.Errorf("both records promoting %s to organisation should be flagged: %v", a.ID, codes)
	}
	if codes[CodePromotionOfRetracted] != 1 {
		t.Errorf("promoting a retracted claim should warn once: %v", codes)
	}
	if fs.HasErrors() {
		t.Errorf("none of these are errors: %+v", fs)
	}
}

func TestPromotionsAreNotKnowledge(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "one")
	b := f.claim(p.ID, "two")
	s := f.synthesis(p.ID, "reconciled", in(a.ID, dkf.RoleThesis), in(b.ID, dkf.RoleAntithesis))
	loose := f.claim(p.ID, "not yet reconciled")
	_, _ = f.w.RebuildIndex()

	g, err := f.w.Load()
	if err != nil {
		t.Fatal(err)
	}
	before := Analyse(g, p)
	beforeLineage, err := Lineage(g, s.ID, 0)
	if err != nil {
		t.Fatal(err)
	}

	// Promote everything in sight, including the loose claim.
	f.promote(dkf.ScopePublic, a.ID, b.ID, s.ID, loose.ID)
	g, _ = f.w.Load()
	after := Analyse(g, p)

	if after.Current != before.Current || after.Priority != before.Priority ||
		len(after.Unsynthesised) != len(before.Unsynthesised) || len(after.Stale) != len(before.Stale) {
		t.Errorf("promotion must not change any conflict set:\nbefore %+v\nafter  %+v", before, after)
	}
	afterLineage, err := Lineage(g, s.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterLineage.Inputs) != len(beforeLineage.Inputs) {
		t.Errorf("promotion is not provenance: lineage inputs changed from %d to %d",
			len(beforeLineage.Inputs), len(afterLineage.Inputs))
	}
	// And a promotion id is not a lineage subject.
	prs := g.SortedPromotions()
	if _, err := Lineage(g, prs[0].ID, 0); err == nil {
		t.Error("a promotion should not be traceable as provenance")
	}
}

// docClaim writes a claim whose source carries the given document.
func (f *fixture) docClaim(subject, content string, doc dkf.Document, sc dkf.Scope) *dkf.Claim {
	f.t.Helper()
	if sc == "" {
		sc = dkf.ScopePersonal
	}
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content,
		Evidential: dkf.EvidentialObserved, Source: dkf.Source{Author: "ben", Document: doc}, Context: dkf.Context{Scope: sc}, Timestamp: ts}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func TestDocumentVerificationIsOfflineAndAdvisory(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	f.particular("Ben", "ben")
	docPath := filepath.Join(f.w.Root, "docs", "architecture.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# Architecture\n\nIn staging, the billing service listens on 443.\n\nOther prose.\n"
	if err := os.WriteFile(docPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	quote := "In staging, the billing service listens on 443."
	hash := dkf.HashDocumentBytes([]byte(body))

	verified := f.docClaim(p.ID, "billing listens on 443", dkf.Document{Ref: "docs/architecture.md", Hash: hash, Quote: quote}, "")
	remote := f.docClaim(p.ID, "remote evidence", dkf.Document{Ref: "https://example.com/x", Hash: hash, Quote: quote}, "")
	missing := f.docClaim(p.ID, "vanished file", dkf.Document{Ref: "docs/gone.md", Hash: hash}, "")
	bare := f.docClaim(p.ID, "a conversation", dkf.Document{Ref: "chat session 2026-08-22"}, "")
	_, _ = f.w.RebuildIndex()

	codesFor := func(id string) []string {
		fs, err := Validate(f.w)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		g, _ := f.w.Load()
		for _, fi := range fs {
			if fi.Path == g.Files[id] {
				out = append(out, fi.Code)
			}
		}
		return out
	}
	if got := codesFor(verified.ID); len(got) != 0 {
		t.Errorf("an intact document should produce no findings: %v", got)
	}
	for _, tc := range []struct {
		id, why string
	}{{remote.ID, "a remote URI must not be fetched"}, {missing.ID, "a missing file is unverified"}, {bare.ID, "a bare reference is unverified"}} {
		got := codesFor(tc.id)
		if len(got) != 1 || got[0] != CodeUnverifiedDocument {
			t.Errorf("%s: got %v", tc.why, got)
		}
	}

	// Context drift: the quote survives, its surroundings do not.
	edited := strings.Replace(body, "Other prose.", "Rewritten prose.", 1)
	if err := os.WriteFile(docPath, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := codesFor(verified.ID); len(got) != 1 || got[0] != CodeContextDrift {
		t.Errorf("context drift: %v", got)
	}
	fs, _ := Validate(f.w)
	if fs.HasErrors() {
		t.Error("drift must never be an error")
	}
	for _, fi := range fs {
		if fi.Code == CodeContextDrift && fi.Severity != SeverityWarning {
			t.Errorf("drift should be a warning, got %s", fi.Severity)
		}
		if fi.Code == CodeUnverifiedDocument && fi.Severity != SeverityInfo {
			t.Errorf("unverified should be a note, got %s", fi.Severity)
		}
	}

	// Quote drift: the cited sentence is gone.
	gone := strings.Replace(edited, quote, "In production, the billing service listens on 8443.", 1)
	if err := os.WriteFile(docPath, []byte(gone), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := codesFor(verified.ID); len(got) != 1 || got[0] != CodeQuoteDrift {
		t.Errorf("quote drift: %v", got)
	}
}

func TestQuotedSourceIsNotedWhenShared(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	doc := dkf.Document{Ref: "https://example.com/x", Quote: "verbatim source text"}
	priv := f.docClaim(p.ID, "personal note", doc, dkf.ScopePersonal)
	org := f.docClaim(p.ID, "shared fact", doc, dkf.ScopeOrganisation)
	_, _ = f.w.RebuildIndex()

	noted := map[string]bool{}
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := f.w.Load()
	for _, fi := range fs {
		if fi.Code == CodeQuotedSource {
			for _, id := range []string{priv.ID, org.ID} {
				if fi.Path == g.Files[id] {
					noted[id] = true
				}
			}
			if fi.Severity != SeverityInfo {
				t.Errorf("quoted_source should be a note, got %s", fi.Severity)
			}
		}
	}
	if noted[priv.ID] {
		t.Error("a personal quote discloses nothing beyond the workspace and should stay quiet")
	}
	if !noted[org.ID] {
		t.Error("an organisation-scoped quote should be noted")
	}
	// Promoting the personal one brings it into scope for the note.
	f.promote(dkf.ScopeOrganisation, priv.ID)
	fs, _ = Validate(f.w)
	var nowNoted bool
	g, _ = f.w.Load()
	for _, fi := range fs {
		if fi.Code == CodeQuotedSource && fi.Path == g.Files[priv.ID] {
			nowNoted = true
		}
	}
	if !nowNoted {
		t.Error("promoting a quoted claim should bring the disclosure note with it")
	}
}

func TestDefectAgainstDriftedDocumentIsUnverifiable(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	f.particular("Ben", "ben")
	docPath := filepath.Join(f.w.Root, "docs", "a.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "# A\n\nThe service listens on 443.\n\nTail.\n"
	if err := os.WriteFile(docPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := dkf.Document{Ref: "docs/a.md", Hash: dkf.HashDocumentBytes([]byte(body)), Quote: "The service listens on 443."}
	defect := f.docClaim(p.ID, "listens on 443", doc, "")
	supersede := f.docClaim(p.ID, "also listens on 443", doc, "")
	_, _ = f.w.RebuildIndex()

	retract := func(id string, kind dkf.RetractionKind) {
		t.Helper()
		if _, err := f.w.Retract(id, &dkf.Retracted{Timestamp: ts, Reason: "x", Source: dkf.Source{Author: "ben"}, Kind: kind}); err != nil {
			t.Fatal(err)
		}
	}
	retract(defect.ID, dkf.KindDefect)
	retract(supersede.ID, dkf.KindSupersession)
	_, _ = f.w.RebuildIndex()

	codes := func() map[string][]string {
		fs, err := Validate(f.w)
		if err != nil {
			t.Fatal(err)
		}
		g, _ := f.w.Load()
		out := map[string][]string{}
		for _, fi := range fs {
			for _, id := range []string{defect.ID, supersede.ID} {
				if fi.Path == g.Files[id] {
					out[id] = append(out[id], fi.Code)
				}
			}
		}
		return out
	}
	// Intact document: nothing to say about either retraction.
	for id, got := range codes() {
		if len(got) != 0 {
			t.Errorf("%s: an unchanged document should produce no findings: %v", id, got)
		}
	}

	// Now the document drifts.
	if err := os.WriteFile(docPath, []byte("# A\n\nThe service listens on 8443.\n\nTail.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := codes()
	if !contains(got[defect.ID], CodeDefectUnverifiable) {
		t.Errorf("a defect against a drifted document is unverifiable: %v", got[defect.ID])
	}
	// The removed check: a supersession against a changed OR unchanged hash is
	// never judged. Only the drift observation may appear.
	if contains(got[supersede.ID], CodeDefectUnverifiable) {
		t.Errorf("supersession must never be cross-checked: %v", got[supersede.ID])
	}
	fs, _ := Validate(f.w)
	for _, fi := range fs {
		if fi.Code == CodeQuoteDrift || fi.Code == CodeContextDrift || fi.Code == CodeDefectUnverifiable {
			if fi.Severity == SeverityError {
				t.Errorf("no drift finding is ever an error: %+v", fi)
			}
		}
		if fi.Code == CodeQuoteDrift || fi.Code == CodeContextDrift {
			if fi.Severity != SeverityInfo {
				t.Errorf("drift under a retracted claim is an observation, got %s", fi.Severity)
			}
		}
	}
}

func TestUnknownHashAlgorithmIsUnverified(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	f.particular("Ben", "ben")
	docPath := filepath.Join(f.w.Root, "docs", "a.md")
	if err := os.MkdirAll(filepath.Dir(docPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(docPath, []byte("content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := f.docClaim(p.ID, "x", dkf.Document{Ref: "docs/a.md", Hash: "blake3:" + strings.Repeat("a", 32)}, "")
	_, _ = f.w.RebuildIndex()
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := f.w.Load()
	var found bool
	for _, fi := range fs {
		if fi.Path == g.Files[c.ID] {
			if fi.Code != CodeUnverifiedDocument || fi.Severity != SeverityInfo {
				t.Errorf("an unimplemented algorithm is unverified, not invalid: %+v", fi)
			}
			found = true
		}
	}
	if !found {
		t.Error("expected an unverified note for the unknown algorithm")
	}
	if fs.HasErrors() {
		t.Error("another implementation's algorithm must never be an error")
	}
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func TestLegacyDocumentURIReportedOnce(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	c := f.docClaim(p.ID, "x", dkf.Document{Ref: "docs/a.md", Quote: "q"}, "")
	_, _ = f.w.RebuildIndex()
	// Rewrite the file as v0.8.0 would have: ref becomes uri.
	path := filepath.Join(f.w.Root, "claims", c.ID+".yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(data), "    ref: ", "    uri: ", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := f.w.Load()
	var codes []string
	for _, fi := range fs {
		if fi.Path == g.Files[c.ID] {
			codes = append(codes, fi.Code)
		}
	}
	if !contains(codes, CodeLegacyDocumentURI) {
		t.Errorf("the legacy key should be reported: %v", codes)
	}
	if contains(codes, CodeNonCanonical) {
		t.Errorf("no non_canonical for the same cause, matching legacy_produced_by: %v", codes)
	}
	if fs.HasErrors() {
		t.Error("a legacy document is valid")
	}
}

func TestForbiddenAliasIsCaughtByTheNodeWalk(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	// A struct decode silently expands aliases, so only validate's node walk
	// can see this file for what it is.
	id := "clm_01a03999-0000-7000-8000-0000000000aa"
	path := filepath.Join(f.w.Root, "claims", id+".yaml")
	content := "id: " + id + "\ntype: claim\nsubject: " + p.ID + "\ncontent: &c the same text twice\nsource:\n  author: ben\n  document:\n    ref: notes.md\n    quote: *c\ncontext:\n  scope: personal\ntimestamp: 2026-08-26T09:00:00Z\nevidential: observed\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, _ = f.w.RebuildIndex()

	// The file reads: the alias expanded into the quote.
	g, err := f.w.Load()
	if err != nil {
		t.Fatal(err)
	}
	c, ok := g.Assertion(id).(*dkf.Claim)
	if !ok || c.Source.Document.Quote != "the same text twice" {
		t.Fatalf("the alias should have expanded on read: %+v", c)
	}
	// And validate names it as the error it is.
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, fi := range fs {
		if fi.Code == CodeForbiddenAlias {
			found = true
			if fi.Severity != SeverityError {
				t.Errorf("forbidden_alias is an error, got %s", fi.Severity)
			}
		}
	}
	if !found {
		t.Error("an aliased object file should be reported")
	}
	// Plain files never trip it.
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	_, _ = f.w.RebuildIndex()
	fs, _ = Validate(f.w)
	for _, fi := range fs {
		if fi.Code == CodeForbiddenAlias {
			t.Errorf("no alias in the workspace, yet: %+v", fi)
		}
	}
}

func TestResolveAuthorAndWriterRule(t *testing.T) {
	f := newFixture(t)
	ben := f.particular("Ben Fortuna", "ben")
	f.particular("Ben Smith", "ben") // a second ben: the name is ambiguous
	jane := f.particular("Jane")
	g, _ := f.w.Load()
	if p, cands := ResolveAuthor(g, "jane"); p == nil || p.ID != jane.ID || cands != nil {
		t.Errorf("unique name resolves: %v %v", p, cands)
	}
	if p, cands := ResolveAuthor(g, "ben"); p != nil || len(cands) != 2 {
		t.Errorf("ambiguous name resolves to none, with candidates: %v %v", p, cands)
	}
	if p, _ := ResolveAuthor(g, ben.URI); p == nil || p.ID != ben.ID {
		t.Error("uri resolves to its particular")
	}
	if w, _, err := ResolveAuthorForWrite(g, "jane", true, "--author"); err != nil || w != jane.URI {
		t.Errorf("a unique name writes the uri: %q %v", w, err)
	}
	if _, _, err := ResolveAuthorForWrite(g, "ben", true, "--author"); err == nil {
		t.Error("an explicitly given ambiguous name must be refused")
	}
	if w, amb, err := ResolveAuthorForWrite(g, "ben", false, "--author"); err != nil || w != "ben" || len(amb) != 2 {
		t.Errorf("a default ambiguous name is written unchanged with candidates: %q %v %v", w, amb, err)
	}
	if _, _, err := ResolveAuthorForWrite(g, "par_01a00000-0000-7000-8000-000000000001", true, "--author"); err == nil {
		t.Error("an id naming no particular must be refused")
	}
	orcid := "https://orcid.org/0000-0002-1825-0097"
	if w, _, err := ResolveAuthorForWrite(g, orcid, true, "--author"); err != nil || w != orcid {
		t.Errorf("an unknown uri is written unchanged: %q %v", w, err)
	}
}

func TestRecallByAuthorRelations(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	ben := f.particular("Ben", "ben")
	jane := f.particular("Jane", "jane")
	a := f.claim(p.ID, "asserted by ben") // fixture claims carry author: ben
	r := f.docClaim(p.ID, "reported from jane", dkf.Document{Ref: "conversation with Jane", Author: "jane"}, "")
	self := f.docClaim(p.ID, "ben quoting ben", dkf.Document{Ref: "notes", Author: ben.URI}, "")
	f.merge(ben.URI, "urn:test:ben")
	viaURN := f.docClaim(p.ID, "attributed through a merge", dkf.Document{Ref: "x", Author: "urn:test:ben"}, "")
	g, _ := f.w.Load()

	rels := map[string][]string{}
	for _, e := range Recall(g, RecallOptions{Author: ben.ID}) {
		rels[e.ID] = e.Relations
	}
	if len(rels) != 4 {
		t.Fatalf("recall by ben: %+v", rels)
	}
	want := map[string][]string{
		a.ID:      {"asserted"},
		r.ID:      {"asserted"},
		self.ID:   {"asserted", "reported"},
		viaURN.ID: {"asserted", "reported"}, // the URN reaches ben through the merge
	}
	for id, w := range want {
		if got := rels[id]; strings.Join(got, ",") != strings.Join(w, ",") {
			t.Errorf("%s relations = %v, want %v", id, got, w)
		}
	}
	janes := Recall(g, RecallOptions{Author: jane.ID})
	if len(janes) != 1 || janes[0].ID != r.ID || strings.Join(janes[0].Relations, ",") != "reported" {
		t.Errorf("recall by jane: %+v", janes)
	}
	// Without the filter, entries carry no relations.
	for _, e := range Recall(g, RecallOptions{Subject: p.ID}) {
		if e.Relations != nil {
			t.Errorf("%s carries relations without an author filter", e.ID)
		}
	}
}

func TestURLInContentNote(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	f.particular("Ben", "ben")
	catalogue := f.claim(p.ID, "Article: warmest winter — https://example.com/a")
	proper := f.docClaim(p.ID, "Sydney recorded its warmest winter", dkf.Document{Ref: "https://example.com/a"}, "")
	f.synthesis(p.ID, "reconciled; see https://example.com/a", in(catalogue.ID, dkf.RoleThesis))
	_, _ = f.w.RebuildIndex()
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := f.w.Load()
	var hits []string
	for _, fi := range fs {
		if fi.Code == CodeURLInContent {
			if fi.Severity != SeverityInfo {
				t.Errorf("a note, never a problem: %+v", fi)
			}
			hits = append(hits, fi.Path)
		}
	}
	if len(hits) != 1 || hits[0] != g.Files[catalogue.ID] {
		t.Errorf("only the catalogue-shaped claim is noted (not the documented fact, not the synthesis): %v", hits)
	}
	_ = proper
	if !IsCorpusFact(CodeURLInContent) {
		t.Error("url_in_content aggregates as a corpus fact")
	}
	if fs.HasErrors() {
		t.Error("the note must not fail validation")
	}
}

func (f *fixture) synthesisOpen(subject, unresolved string, at time.Time, inputs ...dkf.Input) *dkf.Synthesis {
	f.t.Helper()
	s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: subject, Content: "synthesis", Inputs: inputs, Unresolved: unresolved, Source: dkf.Source{Harness: "test"}, Method: dkf.DefaultMethod, Timestamp: at, Context: dkf.Context{Scope: dkf.ScopePersonal}}
	if err := f.w.Create(s); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func TestUnresolved(t *testing.T) {
	f := newFixture(t)
	x := f.particular("Project X")
	y := f.particular("Project Y")
	w := f.particular("Project W")
	n := f.particular("Project N")
	e := f.particular("Project E")
	ma := f.particular("Merged A")
	mb := f.particular("Merged B")
	r := f.particular("Project R")
	day := 24 * time.Hour

	// X: two syntheses in a chain, then a claim after the current one.
	xa := f.claim(x.ID, "A")
	xb := f.claim(x.ID, "B")
	x1 := f.synthesisOpen(x.ID, "A is open", ts, in(xa.ID, dkf.RoleThesis), in(xb.ID, dkf.RoleAntithesis))
	x2 := f.synthesisOpen(x.ID, "B is open", ts.Add(day), in(x1.ID, dkf.RoleThesis))
	f.claim(x.ID, "D after synthesis")
	// Y: claims only.
	f.claim(y.ID, "one")
	f.claim(y.ID, "two")
	f.claim(y.ID, "three")
	// W: the oldest current synthesis.
	wa := f.claim(w.ID, "A")
	ws := f.synthesisOpen(w.ID, "W open", ts.Add(-day), in(wa.ID, dkf.RoleThesis))
	// N: the conventional empty value.
	na := f.claim(n.ID, "A")
	ns := f.synthesisOpen(n.ID, NoneIdentified, ts, in(na.ID, dkf.RoleThesis))
	// E: a non-conventional near-miss is still listed.
	ea := f.claim(e.ID, "A")
	es := f.synthesisOpen(e.ID, "nothing", ts.Add(2*day), in(ea.ID, dkf.RoleThesis))
	// Merged class with the synthesis on the higher member.
	mba := f.claim(mb.ID, "A")
	ms := f.synthesisOpen(mb.ID, "merged open", ts.Add(3*day), in(mba.ID, dkf.RoleThesis))
	f.merge(ma.URI, mb.URI)
	// R: the newest synthesis retracted; its predecessor is current.
	ra := f.claim(r.ID, "A")
	r1 := f.synthesisOpen(r.ID, "r1 open", ts.Add(4*day), in(ra.ID, dkf.RoleThesis))
	r2 := f.synthesisOpen(r.ID, "r2 open", ts.Add(5*day), in(r1.ID, dkf.RoleThesis))
	f.retract(r2.ID)

	g := f.graph()
	entries := Unresolved(g, UnresolvedOptions{})
	byP := map[string]UnresolvedEntry{}
	for _, en := range entries {
		byP[en.Particular] = en
	}
	if _, ok := byP[y.ID]; ok {
		t.Error("a class without a synthesis should not be listed")
	}
	if _, ok := byP[n.ID]; ok {
		t.Error("None identified should be hidden by default")
	}
	ex := byP[x.ID]
	if ex.Synthesis != x2.ID || ex.Unresolved != "B is open" || ex.Unsynthesised != 1 || ex.Members != nil {
		t.Errorf("X entry: %+v", ex)
	}
	if ee := byP[e.ID]; ee.Synthesis != es.ID || ee.Unresolved != "nothing" {
		t.Errorf("E entry: %+v", ee)
	}
	if er := byP[r.ID]; er.Synthesis != r1.ID || er.Unresolved != "r1 open" {
		t.Errorf("retracted current should fall back: %+v", er)
	}
	lower, higher := ma.ID, mb.ID
	if higher < lower {
		lower, higher = higher, lower
	}
	if _, ok := byP[higher]; ok {
		t.Error("merged class should be keyed by its lowest member only")
	}
	if em := byP[lower]; em.Synthesis != ms.ID || len(em.Members) != 2 {
		t.Errorf("merged entry: %+v", em)
	}
	if len(entries) != 5 || entries[0].Synthesis != ws.ID || entries[1].Synthesis != x2.ID || entries[4].Synthesis != r1.ID {
		t.Errorf("ordering (oldest current first): %+v", entries)
	}

	withNone := Unresolved(g, UnresolvedOptions{IncludeNone: true})
	if len(withNone) != 6 {
		t.Errorf("include none: %d entries", len(withNone))
	}
	found := false
	for _, en := range withNone {
		if en.Synthesis == ns.ID && en.Unresolved == NoneIdentified {
			found = true
		}
	}
	if !found {
		t.Error("include none should list the conventional entry")
	}
	if one := Unresolved(g, UnresolvedOptions{Subject: x.ID}); len(one) != 1 || one[0].Synthesis != x2.ID {
		t.Errorf("single subject: %+v", one)
	}
	if none := Unresolved(g, UnresolvedOptions{Subject: y.ID}); len(none) != 0 {
		t.Errorf("subject without synthesis: %+v", none)
	}

	// Scope follows the current synthesis's effective scope.
	f.promote(dkf.ScopeOrganisation, x2.ID)
	g = f.graph()
	if org := Unresolved(g, UnresolvedOptions{Scope: dkf.ScopeOrganisation}); len(org) != 1 || org[0].Synthesis != x2.ID {
		t.Errorf("promoted scope: %+v", org)
	}
	for _, en := range Unresolved(g, UnresolvedOptions{Scope: dkf.ScopePersonal}) {
		if en.Synthesis == x2.ID {
			t.Error("promoted synthesis should not be listed at its asserted scope")
		}
	}
}
