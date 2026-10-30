package query

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// inferred asserts an inferred claim with the given basis through the normal
// write path.
func (f *fixture) inferred(subject, content string, basis ...string) *dkf.Claim {
	f.t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content, Evidential: dkf.EvidentialInferred,
		Basis: basis, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	if len(basis) == 0 {
		c.Basis = nil
	}
	if err := f.w.Create(c); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

// rawClaim writes c as canonical bytes without the write path's checks, for
// files a writer would refuse but a reader must still read.
func (f *fixture) rawClaim(c *dkf.Claim) {
	f.t.Helper()
	data, err := dkf.Encode(c)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.w.Dir(dkf.TypeClaim), c.ID+".yaml"), data, 0o644); err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.w.RebuildIndex(); err != nil {
		f.t.Fatal(err)
	}
}

func rawInferred(subject string, ev dkf.Evidential, basis []string) *dkf.Claim {
	return &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: "rule", Evidential: ev, Basis: basis,
		Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
}

func findingFor(fs Findings, code, pathPart string) bool {
	for _, f := range fs {
		if f.Code == code && strings.Contains(f.Path, pathPart) {
			return true
		}
	}
	return false
}

func TestValidateBasisClean(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	f.particular("Ben", "ben")
	a := f.claim(p.ID, "A")
	f.inferred(p.ID, "rule", a.ID)
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("an inferred claim with a basis should have no findings: %+v", fs)
	}
}

func TestValidateBasisReferences(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")

	dangling := rawInferred(p.ID, dkf.EvidentialInferred, []string{dkf.NewID(dkf.TypeClaim)})
	f.rawClaim(dangling)
	wrongType := rawInferred(p.ID, dkf.EvidentialInferred, []string{p.ID})
	f.rawClaim(wrongType)
	self := rawInferred(p.ID, dkf.EvidentialInferred, nil)
	self.Basis = []string{self.ID}
	f.rawClaim(self)
	empty := rawInferred(p.ID, dkf.EvidentialInferred, []string{})
	f.rawClaim(empty)
	dup := rawInferred(p.ID, dkf.EvidentialInferred, []string{a.ID, a.ID})
	f.rawClaim(dup)

	// A cycle through a basis and an input: the claim cites the synthesis,
	// which cites the claim.
	synID := dkf.NewID(dkf.TypeSynthesis)
	loop := f.inferred(p.ID, "loop", synID)
	syn := &dkf.Synthesis{ID: synID, Subject: p.ID, Content: "S", Inputs: []dkf.Input{in(loop.ID, dkf.RoleThesis)},
		Unresolved: "n", Source: dkf.Source{Harness: "t"}, Method: dkf.DefaultMethod, Timestamp: ts, Context: dkf.Context{Scope: dkf.ScopePersonal}}
	if err := f.w.Create(syn); err != nil {
		t.Fatal(err)
	}
	if _, err := f.w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}

	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ code, id, sev string }{
		{CodeDanglingReference, dangling.ID, SeverityError},
		{dkf.CodeInvalidID, wrongType.ID, SeverityError},
		{dkf.CodeSelfBasis, self.ID, SeverityError},
		{dkf.CodeEmptyBasis, empty.ID, SeverityError},
		{CodeDuplicateBasis, dup.ID, SeverityWarning},
		{CodeCycle, loop.ID, SeverityError},
		{CodeCycle, syn.ID, SeverityError},
	} {
		found := false
		for _, fd := range fs {
			if fd.Code == c.code && strings.Contains(fd.Path, c.id) {
				found = true
				if fd.Severity != c.sev {
					t.Errorf("%s on %s: severity %s, want %s", c.code, c.id, fd.Severity, c.sev)
				}
			}
		}
		if !found {
			t.Errorf("want %s against %s; got %+v", c.code, c.id, fs)
		}
	}
	if findingFor(fs, CodeCycle, self.ID) {
		t.Error("a self-citing basis is self_basis, not cycle")
	}
	// Every file is still read.
	g := f.graph()
	for _, id := range []string{dangling.ID, wrongType.ID, self.ID, empty.ID, dup.ID} {
		if g.Assertion(id) == nil {
			t.Errorf("%s must still be read", id)
		}
	}
}

func TestValidateBasisNotInferred(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	held := rawInferred(p.ID, dkf.EvidentialHeld, []string{a.ID})
	f.rawClaim(held)
	s := f.synthesis(p.ID, "S", in(a.ID, dkf.RoleThesis))
	path := filepath.Join(f.w.Dir(dkf.TypeSynthesis), s.ID+".yaml")
	data, _ := os.ReadFile(path)
	_ = os.WriteFile(path, append(data, []byte("basis:\n  - "+a.ID+"\n")...), 0o644)

	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	if !fs.HasErrors() || !findingFor(fs, dkf.CodeBasisNotInferred, held.ID) || !findingFor(fs, dkf.CodeBasisNotInferred, s.ID) {
		t.Errorf("want basis_not_inferred errors on the held claim and the synthesis: %+v", fs)
	}
	got := ids(Recall(f.graph(), RecallOptions{Subject: p.ID}))
	if !strings.Contains(got, held.ID) || !strings.Contains(got, s.ID) {
		t.Errorf("both files must still be recalled, got %s", got)
	}
}

func TestInferredWithoutBasisIsAggregateCorpusFact(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	for i := 0; i < 40; i++ {
		f.inferred(p.ID, "rule")
	}
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	if n := codes(fs)[SeverityInfo+":"+CodeInferredWithoutBasis]; n != 40 {
		t.Errorf("want 40 inferred_without_basis notes, got %d", n)
	}
	if fs.HasErrors() {
		t.Errorf("inferred without basis is valid: %+v", fs)
	}
	if !IsCorpusFact(CodeInferredWithoutBasis) {
		t.Error("inferred_without_basis is a fact about the corpus, reported in aggregate")
	}
}

func TestValidateStaleClaim(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	rule := f.inferred(p.ID, "rule", a.ID)
	f.retract(a.ID)
	fs, err := Validate(f.w)
	if err != nil {
		t.Fatal(err)
	}
	if !findingFor(fs, CodeStaleClaim, rule.ID) {
		t.Errorf("want stale_claim on the rule: %+v", fs)
	}
	if fs.HasErrors() {
		t.Errorf("a stale claim is a warning: %+v", fs)
	}
}

func TestConflictsStaleThroughBasis(t *testing.T) {
	f := newFixture(t)
	x := f.particular("Project X")
	y := f.particular("Library Y")
	a := f.claim(x.ID, "A")
	b := f.claim(x.ID, "B")
	lib := f.claim(y.ID, "lib")
	rule := f.inferred(x.ID, "rule", a.ID, b.ID)
	crossRule := f.inferred(x.ID, "cross", lib.ID)
	syn := f.synthesis(x.ID, "S", in(rule.ID, dkf.RoleThesis))

	r := Analyse(f.graph(), x)
	if len(r.Stale) != 0 {
		t.Fatalf("nothing is retracted yet: %+v", r)
	}
	// The synthesis reconciles the rule, not the instances behind it.
	if want := strings.Join(sortedIDs(a.ID, b.ID, crossRule.ID), " "); strings.Join(r.Unsynthesised, " ") != want {
		t.Errorf("unsynthesised = %v, want %s", r.Unsynthesised, want)
	}

	f.retract(a.ID)
	f.retract(lib.ID)
	r = Analyse(f.graph(), x)
	if want := strings.Join(sortedIDs(rule.ID, crossRule.ID, syn.ID), " "); strings.Join(r.Stale, " ") != want {
		t.Errorf("stale = %v, want %s (the rule, the cross-particular rule, and the synthesis over the rule)", r.Stale, want)
	}

	// Leaving the stale set: retract the rule, superseded by a revised one.
	revised := f.inferred(x.ID, "revised rule", b.ID)
	ret, err := f.w.Retract(rule.ID, &dkf.Retracted{Timestamp: ts, Reason: "a was wrong", Source: dkf.Source{Author: "ben"}, SupersededBy: revised.ID})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.w.UpsertIndex(ret); err != nil {
		t.Fatal(err)
	}
	r = Analyse(f.graph(), x)
	for _, id := range r.Stale {
		if id == rule.ID {
			t.Error("a retracted rule leaves the stale set")
		}
	}
	if !contains(r.Unsynthesised, revised.ID) {
		t.Errorf("the replacement is unsynthesised like any other: %v", r.Unsynthesised)
	}
}

func TestBasisDoesNotSynthesise(t *testing.T) {
	f := newFixture(t)
	x := f.particular("Project X")
	a := f.claim(x.ID, "A")
	b := f.claim(x.ID, "B")
	rule := f.inferred(x.ID, "rule", a.ID, b.ID)
	r := Analyse(f.graph(), x)
	if want := strings.Join(sortedIDs(a.ID, b.ID, rule.ID), " "); strings.Join(r.Unsynthesised, " ") != want {
		t.Errorf("unsynthesised = %v, want all three", r.Unsynthesised)
	}
}

func TestPriorityCountsStaleRuleTwice(t *testing.T) {
	f := newFixture(t)
	x := f.particular("Project X")
	y := f.particular("Library Y")
	lib := f.claim(y.ID, "lib")
	f.inferred(x.ID, "rule", lib.ID)
	f.claim(x.ID, "other")
	f.retract(lib.ID)
	r := Analyse(f.graph(), x)
	if r.Priority != 3 {
		t.Errorf("priority = %d, want 3 (two unsynthesised, one stale): %+v", r.Priority, r)
	}
}

func TestAnalyseSetFollowsBasisOutsideTheSet(t *testing.T) {
	f := newFixture(t)
	x := f.particular("Project X")
	a := f.claim(x.ID, "A")
	b := f.claim(x.ID, "B")
	rule := f.inferred(x.ID, "rule", a.ID)
	f.retract(a.ID)
	r, err := AnalyseSet(f.graph(), []string{rule.ID, b.ID})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.Stale, " ") != rule.ID {
		t.Errorf("stale = %v, want the rule", r.Stale)
	}
}

func sortedIDs(ids ...string) []string {
	out := append([]string(nil), ids...)
	sort.Strings(out)
	return out
}

func TestScopeWiderThanBasis(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	priv := f.claim(p.ID, "a personal instance")
	rule := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: "rule", Evidential: dkf.EvidentialInferred,
		Basis: []string{priv.ID}, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopeOrganisation}, Timestamp: ts}
	if err := f.w.Create(rule); err != nil {
		t.Fatal(err)
	}
	_, _ = f.w.RebuildIndex()

	g := f.graph()
	msg := ScopeWiderThanInputs(g, g.Assertion(rule.ID))
	if !strings.Contains(msg, "inferred claim "+rule.ID) || !strings.Contains(msg, "basis member(s) "+priv.ID) {
		t.Fatalf("an organisation rule over a personal instance should warn, naming the member: %q", msg)
	}
	fs, _ := Validate(f.w)
	if !findingFor(fs, CodeScopeWiderThanInputs, rule.ID) {
		t.Errorf("validate should report it: %+v", fs)
	}

	// Promoting the basis member clears it on a claim the promotion did not name.
	pr := f.promote(dkf.ScopeOrganisation, priv.ID)
	g = f.graph()
	for _, m := range ScopeFindingsForPromotion(g, pr) {
		if strings.Contains(m, rule.ID) {
			t.Errorf("the promotion cleared the condition, so it must not report it: %s", m)
		}
	}
	if msg := ScopeWiderThanInputs(g, g.Assertion(rule.ID)); msg != "" {
		t.Errorf("promoting the member should clear it: %s", msg)
	}

	// Promoting the rule past its basis creates it again; the basis stays put.
	pr = f.promote(dkf.ScopePublic, rule.ID)
	g = f.graph()
	found := false
	for _, m := range ScopeFindingsForPromotion(g, pr) {
		found = found || strings.Contains(m, rule.ID)
	}
	if !found {
		t.Error("promoting the rule past its basis should be reported")
	}
	if got := g.EffectiveScope(priv.ID); got != dkf.ScopeOrganisation {
		t.Errorf("promotion never cascades to a basis: member is %s", got)
	}
}

func TestLineageFollowsBasis(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	b := f.claim(p.ID, "B")
	c := f.claim(p.ID, "C")
	rule := f.inferred(p.ID, "rule", a.ID, b.ID, c.ID)
	syn := f.synthesis(p.ID, "S", in(rule.ID, dkf.RoleThesis))
	g := f.graph()

	tree, err := Lineage(g, rule.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Inputs) != 0 || len(tree.Basis) != 3 || tree.Basis[0].ID != a.ID || tree.Basis[0].Role != "" {
		t.Fatalf("a rule's instances are its basis children, with no role: %+v", tree)
	}

	tree, _ = Lineage(g, syn.ID, 0)
	r := tree.Inputs[0]
	if r.ID != rule.ID || r.Role != dkf.RoleThesis || len(r.Basis) != 3 {
		t.Fatalf("through the synthesis to the rule's instances: %+v", r)
	}

	// A basis edge counts as a level.
	shallow, _ := Lineage(g, syn.ID, 2)
	if len(shallow.Inputs[0].Basis) != 0 || !shallow.Inputs[0].Truncated {
		t.Errorf("depth 2 should stop at the rule: %+v", shallow.Inputs[0])
	}
	if leaf, _ := Lineage(g, a.ID, 0); leaf.Basis == nil || leaf.Inputs == nil {
		t.Error("every node carries both lists, empty when there is nothing to list")
	}
}

func TestLineageExpandsEachObjectOnce(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "A")
	b := f.claim(p.ID, "B")
	c := f.synthesis(p.ID, "C", in(a.ID, dkf.RoleThesis), in(b.ID, dkf.RoleAntithesis))
	d := f.synthesis(p.ID, "D", in(c.ID, dkf.RoleThesis), in(a.ID, dkf.RoleAntithesis))
	tree, err := Lineage(f.graph(), d.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	first := tree.Inputs[0].Inputs[0]
	again := tree.Inputs[1]
	if first.ID != a.ID || first.Repeated || first.Content != "A" {
		t.Errorf("the first occurrence is expanded: %+v", first)
	}
	if again.ID != a.ID || !again.Repeated || again.Content != "" || again.Role != dkf.RoleAntithesis || again.Type != dkf.TypeClaim {
		t.Errorf("a later occurrence is named, with its edge, and not expanded: %+v", again)
	}
}

func TestRecallCarriesBasisInLineageOrder(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	// The rule's id sorts before its instance's, so only the basis edge
	// puts the instance first.
	ruleID := dkf.NewID(dkf.TypeClaim)
	a := f.claim(p.ID, "A")
	rule := &dkf.Claim{ID: ruleID, Subject: p.ID, Content: "rule", Evidential: dkf.EvidentialInferred, Basis: []string{a.ID},
		Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	if err := f.w.Create(rule); err != nil {
		t.Fatal(err)
	}
	_, _ = f.w.RebuildIndex()
	es := Recall(f.graph(), RecallOptions{Subject: p.ID})
	if ids(es) != a.ID+" "+rule.ID {
		t.Errorf("order = %s, want the instance before the rule", ids(es))
	}
	for _, e := range es {
		if e.ID == rule.ID && (len(e.Basis) != 1 || e.Basis[0] != a.ID) {
			t.Errorf("rule entry basis = %v", e.Basis)
		}
		if e.ID == a.ID && e.Basis != nil {
			t.Errorf("an entry without a basis carries none: %v", e.Basis)
		}
	}
}
