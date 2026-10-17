package query

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

func TestParseQuery(t *testing.T) {
	fold := newFolder()
	for _, tc := range []struct {
		in, whole string
		terms     []string
	}{
		{"timeout?", "timeout", []string{"timeout"}},
		{"the ticket was PROJ-1234.", "the ticket was proj-1234", []string{"the", "ticket", "was", "proj-1234"}},
		{"v1.2.3?", "v1.2.3", []string{"v1.2.3"}},
		{`"Project X"`, "project x", []string{"project", "x"}},
		{"?!", "", nil},
		{"Auth auth AUTH", "auth auth auth", []string{"auth"}},
		{"STRASSE Straße", "strasse strasse", []string{"strasse"}},
		// Edge whitespace is not trimmed until nodelogicau/particulars#37
		// settles it; terms never carry it.
		{" Project X", " project x", []string{"project", "x"}},
	} {
		q := parseQuery(tc.in, fold)
		if q.whole != tc.whole || strings.Join(q.terms, "|") != strings.Join(tc.terms, "|") {
			t.Errorf("parseQuery(%q) = %q %q; want %q %q", tc.in, q.whole, q.terms, tc.whole, tc.terms)
		}
	}
	if !parseQuery("?!", fold).empty() {
		t.Error("punctuation alone must be empty")
	}
}

func TestExactIn(t *testing.T) {
	fold := newFolder()
	for _, tc := range []struct {
		query, name string
		want        bool
	}{
		{"C#?", "C#", true},
		{"C", "C#", false},
		{"Project X?", "Project X", true},
		{`"Project X"`, "Project X", true},
		{"the Project X", "Project X", false},
		{"dkf", "DKF", true},
		{"aa?", "a", false},
		{"?a", "a", true},
		{"x", "", false},
		// Edge whitespace is not an edge yet (nodelogicau/particulars#37).
		{" Project X", "Project X", false},
	} {
		if got := exactIn(fold(tc.query), fold(tc.name)); got != tc.want {
			t.Errorf("exactIn(%q, %q) = %v; want %v", tc.query, tc.name, got, tc.want)
		}
	}
}

func recallQuery(t *testing.T, f *fixture, text string, opts RecallOptions) QueryResult {
	t.Helper()
	r, err := RecallQuery(f.graph(), text, opts)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func parts(r QueryResult) string {
	out := make([]string, len(r.Particulars))
	for i, p := range r.Particulars {
		out[i] = p.Label + ":" + string(p.Resolution)
	}
	return strings.Join(out, " ")
}

func TestRecallQueryBaseline(t *testing.T) {
	f := newFixture(t)
	pt := f.particular("Project T")
	c1 := f.claim(pt.ID, "The auth timeout is 30 minutes.")
	s1 := f.synthesis(pt.ID, "Sessions expire after half an hour", in(c1.ID, dkf.RoleThesis))
	sess := f.particular("Session")
	cs := f.claim(sess.ID, "The session timeout is 30 minutes.")
	tk := f.particular("Tracker")
	f.claim(tk.ID, "Filed as PROJ-1234 yesterday")
	ver := f.particular("Release")
	f.claim(ver.ID, "Shipped v1.2 to staging")
	st := f.particular("Street")
	f.claim(st.ID, "Lives on Hauptstraße")

	// A superseded claim matches; current is still the synthesis.
	r := recallQuery(t, f, "auth timeout", RecallOptions{})
	if len(r.Particulars) == 0 || r.Particulars[0].Particular != pt.ID {
		t.Fatalf("auth timeout: %s", parts(r))
	}
	es := r.Particulars[0].Entries
	if ids(es) != c1.ID+" "+s1.ID || !es[0].Matched || es[1].Matched || !es[1].Current {
		t.Errorf("superseded match: %+v", es)
	}

	// A question mark does not hide a term.
	r = recallQuery(t, f, "timeout?", RecallOptions{})
	var found bool
	for _, p := range r.Particulars {
		if p.Particular == sess.ID {
			found = p.Resolution == ResolutionLexical && p.Entries[0].ID == cs.ID && p.Entries[0].Matched
		}
	}
	if !found {
		t.Errorf("timeout?: %s", parts(r))
	}

	if r = recallQuery(t, f, "the ticket was PROJ-1234.", RecallOptions{}); !strings.Contains(parts(r), "Tracker:lexical") {
		t.Errorf("identifier at sentence end: %s", parts(r))
	}
	if r = recallQuery(t, f, "v1.2.3?", RecallOptions{}); strings.Contains(parts(r), "Release") {
		t.Errorf("interior punctuation kept: %s", parts(r))
	}
	if r = recallQuery(t, f, "HAUPTSTRASSE", RecallOptions{}); parts(r) != "Street:lexical" {
		t.Errorf("full case folding: %s", parts(r))
	}
}

func TestRecallQueryResolution(t *testing.T) {
	f := newFixture(t)
	px := f.particular("Project X")
	f.claim(px.ID, "Uses Postgres")
	cs := f.particular("C#")
	f.claim(cs.ID, "A language with C in its name")
	dk := f.particular("Knowledge Format", "DKF")
	f.claim(dk.ID, "Dialectical")
	q := f.particular("?")
	f.claim(q.ID, "A punctuation-only label")

	for query, want := range map[string]string{
		"Project X?": "Project X:label", `"Project X"`: "Project X:label",
		"C#?": "C#:label", "dkf": "Knowledge Format:alias",
	} {
		if r := recallQuery(t, f, query, RecallOptions{}); len(r.Particulars) == 0 || parts(r)[:len(want)] != want {
			t.Errorf("%q: %s", query, parts(r))
		}
	}
	if r := recallQuery(t, f, "C", RecallOptions{}); !strings.Contains(parts(r), "C#:lexical") {
		t.Errorf("a prefix is not exact: %s", parts(r))
	}
	r := recallQuery(t, f, "?!", RecallOptions{})
	if len(r.Particulars) != 0 || r.Count != 0 || r.Particulars == nil {
		t.Errorf("punctuation alone: %+v", r)
	}
}

func TestRecallQueryFiltersAndClasses(t *testing.T) {
	f := newFixture(t)
	a := f.particular("Alpha")
	keep := f.claim(a.ID, "Unrelated note")
	gone := f.claim(a.ID, "The timeout was raised")
	f.retract(gone.ID)
	p := f.particular("Private")
	f.claim(p.ID, "Personal timeout preference")
	m1 := f.particular("Mars")
	m2 := f.particular("Red Planet")
	f.claim(m1.ID, "Mars has a timeout")
	f.claim(m2.ID, "Red Planet timeout too")
	f.merge(m1.URI, m2.URI)

	r := recallQuery(t, f, "timeout", RecallOptions{})
	byID := map[string]QueryParticular{}
	for _, qp := range r.Particulars {
		byID[qp.Particular] = qp
	}
	// Resolved through a retracted claim; returned with what is visible.
	if qa, ok := byID[a.ID]; !ok || ids(qa.Entries) != keep.ID || qa.Entries[0].Matched {
		t.Errorf("retracted match: %+v", byID[a.ID])
	}
	// One class, under the lowest id, with both members' claims.
	root := min(m1.ID, m2.ID)
	if qm, ok := byID[root]; !ok || len(qm.Class) != 2 || len(qm.Entries) != 2 || len(byID) != 3 {
		t.Errorf("merge class once: %s", parts(r))
	}
	// What the caller cannot see does not rank: Alpha matches no visible
	// text, so it sorts after particulars that do.
	if last := r.Particulars[len(r.Particulars)-1]; last.Particular != a.ID {
		t.Errorf("retracted match ranks last: %s", parts(r))
	}

	// Scope drops a class without counting it.
	if r = recallQuery(t, f, "timeout", RecallOptions{Scope: dkf.ScopePublic}); len(r.Particulars) != 0 || r.Omitted != 0 {
		t.Errorf("nothing in scope: %+v", r)
	}
	// Filters combine with a query.
	f.claim(p.ID, "Session timeout", "auth")
	r = recallQuery(t, f, "timeout", RecallOptions{Topics: []string{"auth"}})
	if parts(r) != "Private:lexical" || len(r.Particulars[0].Entries) != 1 {
		t.Errorf("topic filter: %s", parts(r))
	}
}

func TestRecallQueryOrderAndLimit(t *testing.T) {
	f := newFixture(t)
	pa := f.particular("Alpha")
	f.claim(pa.ID, "auth is required")
	f.claim(pa.ID, "the timeout is short")
	pb := f.particular("Beta")
	f.claim(pb.ID, "auth only")
	pc := f.particular("Gamma")
	f.claim(pc.ID, "the auth timeout is 30m")

	r := recallQuery(t, f, "auth timeout", RecallOptions{})
	if parts(r) != "Gamma:lexical Alpha:lexical Beta:lexical" {
		t.Errorf("whole query, then terms: %s", parts(r))
	}
	r = recallQuery(t, f, "auth timeout", RecallOptions{Limit: 2})
	if parts(r) != "Gamma:lexical Alpha:lexical" || r.Omitted != 1 || r.Count != 2 {
		t.Errorf("limit: %s omitted %d", parts(r), r.Omitted)
	}
	// Ties break by id.
	r = recallQuery(t, f, "auth", RecallOptions{})
	got := []string{r.Particulars[0].Particular, r.Particulars[1].Particular, r.Particulars[2].Particular}
	if got[0] >= got[1] || got[1] >= got[2] {
		t.Errorf("ties by id: %v", got)
	}
	if _, err := RecallQuery(f.graph(), "auth", RecallOptions{Limit: -1}); err == nil {
		t.Error("negative limit must be a usage error")
	}

	// Exact before lexical under the default limit.
	g := newFixture(t)
	bill := g.particular("Billing")
	g.claim(bill.ID, "Owns invoices")
	for i := range 12 {
		p := g.particular(fmt.Sprintf("Service %02d", i))
		g.claim(p.ID, "Sends billing events")
	}
	r = recallQuery(t, g, "billing", RecallOptions{})
	if r.Count != DefaultQueryLimit || r.Omitted != 3 || r.Particulars[0].Particular != bill.ID || r.Particulars[0].Resolution != ResolutionLabel {
		t.Errorf("exact before lexical: %s omitted %d", parts(r), r.Omitted)
	}
}
