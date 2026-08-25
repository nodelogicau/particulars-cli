package graph

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

var ts = time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

type fixture struct {
	t  *testing.T
	ws *store.Workspace
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Defaults.Source.Author = "ben"
	ws, err := store.Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{t: t, ws: ws}
}

func (f *fixture) particular(label string) *dkf.Particular {
	f.t.Helper()
	p, _, err := f.ws.UpsertParticular(dkf.MintURI("", f.ws.Config.Workspace.ID, dkf.Slugify(label)), label, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	return p
}

type claimOpts struct {
	scope      dkf.Scope
	topics     []string
	document   string
	confidence *float64
	at         time.Time
}

func (f *fixture) claim(subject, content string, o claimOpts) *dkf.Claim {
	f.t.Helper()
	if o.scope == "" {
		o.scope = dkf.ScopeOrganisation
	}
	if o.at.IsZero() {
		o.at = ts
	}
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content,
		Source:  dkf.Source{Author: "ben", Document: dkf.Document{Ref: o.document}},
		Context: dkf.Context{Scope: o.scope, Topics: o.topics}, Timestamp: o.at, Confidence: o.confidence}
	if err := f.ws.Create(c); err != nil {
		f.t.Fatal(err)
	}
	if err := f.ws.UpsertIndex(c); err != nil {
		f.t.Fatal(err)
	}
	return c
}

func (f *fixture) synthesis(subject, content, unresolved string, at time.Time, inputs ...dkf.Input) *dkf.Synthesis {
	f.t.Helper()
	s := &dkf.Synthesis{ID: dkf.NewID(dkf.TypeSynthesis), Subject: subject, Content: content, Inputs: inputs,
		Unresolved: unresolved, Source: dkf.Source{Author: "ben", Harness: "test"}, Method: dkf.DefaultMethod,
		Timestamp: at, Context: dkf.Context{Scope: dkf.ScopeOrganisation}}
	if err := f.ws.Create(s); err != nil {
		f.t.Fatal(err)
	}
	if err := f.ws.UpsertIndex(s); err != nil {
		f.t.Fatal(err)
	}
	return s
}

func (f *fixture) retract(id string) {
	f.t.Helper()
	a, err := f.ws.Retract(id, &dkf.Retracted{Timestamp: ts, Reason: "test", Source: dkf.Source{Author: "ben"}})
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.ws.UpsertIndex(a); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) build(o Options) []Line {
	f.t.Helper()
	g, err := f.ws.Load()
	if err != nil {
		f.t.Fatal(err)
	}
	return Build(g, f.ws, o)
}

func in(id string, role dkf.Role) dkf.Input {
	return dkf.Input{ID: id, Role: role, Weight: dkf.WeightPrimary}
}

func f64(v float64) *float64 { return &v }

func TestItemsForWorkspace(t *testing.T) {
	f := newFixture(t)
	a := f.particular("Alpha")
	b := f.particular("Beta")
	f.claim(a.ID, "alpha one", claimOpts{})
	f.claim(b.ID, "beta one", claimOpts{})
	lines := f.build(Options{})
	if len(lines) != 2 {
		t.Fatalf("expected 2 items, got %d", len(lines))
	}
	if lines[0].ID > lines[1].ID {
		t.Error("items should be ordered by particular id")
	}
	for _, l := range lines {
		if len(l.Item.ACL) != 1 || l.Item.ACL[0].Type != "everyone" || l.Item.ACL[0].AccessType != "grant" {
			t.Errorf("acl: %+v", l.Item.ACL)
		}
		if l.Item.Content.Type != "text" || l.Item.Content.Value == "" {
			t.Errorf("content: %+v", l.Item.Content)
		}
	}
}

func TestDeterministic(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Alpha")
	f.claim(p.ID, "one", claimOpts{topics: []string{"b", "a"}})
	f.claim(p.ID, "two", claimOpts{topics: []string{"a"}})
	one, _ := json.Marshal(f.build(Options{SourceURL: "https://example.com/blob/main/"}))
	two, _ := json.Marshal(f.build(Options{SourceURL: "https://example.com/blob/main/"}))
	if string(one) != string(two) {
		t.Error("export is not deterministic")
	}
}

func TestScopeGovernsExport(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Alpha")
	f.claim(p.ID, "public knowledge", claimOpts{scope: dkf.ScopeOrganisation})
	f.claim(p.ID, "SECRET personal note", claimOpts{scope: dkf.ScopePersonal})
	only := f.particular("Private Only")
	f.claim(only.ID, "SECRET all personal", claimOpts{scope: dkf.ScopePersonal})

	lines := f.build(Options{})
	if len(lines) != 1 || lines[0].ID != p.ID {
		t.Fatalf("wholly personal particular must be omitted: %+v", lines)
	}
	body, _ := json.Marshal(lines[0])
	if strings.Contains(string(body), "SECRET") {
		t.Errorf("personal content leaked:\n%s", body)
	}
	if lines[0].Item.Properties.ClaimCount != 1 {
		t.Errorf("claimCount should exclude personal: %d", lines[0].Item.Properties.ClaimCount)
	}
	// public wins over organisation for the scope property
	f.claim(p.ID, "published", claimOpts{scope: dkf.ScopePublic})
	if got := f.build(Options{})[0].Item.Properties.Scope; got != string(dkf.ScopePublic) {
		t.Errorf("scope = %s, want public", got)
	}
	// narrowing works
	if lines := f.build(Options{Scope: dkf.ScopePublic}); len(lines) != 1 || lines[0].Item.Properties.ClaimCount != 1 {
		t.Errorf("narrowed export: %+v", lines)
	}
}

func TestRetractedNeverExported(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Alpha")
	a := f.claim(p.ID, "WITHDRAWN claim", claimOpts{})
	standing := f.claim(p.ID, "standing claim", claimOpts{})
	before := f.build(Options{})[0].Item.Properties.ClaimCount
	f.retract(a.ID)
	line := f.build(Options{})[0]
	if strings.Contains(line.Item.Content.Value, "WITHDRAWN") {
		t.Errorf("retracted content exported:\n%s", line.Item.Content.Value)
	}
	if line.Item.Properties.ClaimCount != before-1 {
		t.Errorf("claimCount %d, want %d", line.Item.Properties.ClaimCount, before-1)
	}

	// A retracted synthesis must not be the belief; the older one stands.
	older := f.synthesis(p.ID, "older belief", "None identified", ts, in(standing.ID, dkf.RoleThesis))
	newer := f.synthesis(p.ID, "NEWER belief", "n", ts.Add(time.Hour), in(standing.ID, dkf.RoleThesis))
	f.retract(newer.ID)
	line = f.build(Options{})[0]
	if !strings.Contains(line.Item.Content.Value, "older belief") || strings.Contains(line.Item.Content.Value, "NEWER") {
		t.Errorf("belief after retraction:\n%s", line.Item.Content.Value)
	}
	if line.Item.Properties.CurrentSynthesis != older.ID {
		t.Errorf("currentSynthesis = %s, want %s", line.Item.Properties.CurrentSynthesis, older.ID)
	}
}

func TestBriefContent(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "Uses Postgres 16", claimOpts{document: "docs/db.md", confidence: f64(0.9), topics: []string{"db"}})
	b := f.claim(p.ID, "Used MySQL until 2024", claimOpts{document: "adr/1.md"})
	s := f.synthesis(p.ID, "Postgres since 2024", "Compliance basis unsourced.", ts.Add(time.Hour), in(a.ID, dkf.RoleThesis), in(b.ID, dkf.RoleAntithesis))
	later := f.claim(p.ID, "Billing split out in Q2", claimOpts{at: ts.Add(2 * time.Hour)})

	line := f.build(Options{})[0]
	v := line.Item.Content.Value
	for _, want := range []string{
		"Project X  (" + p.URI + ")",
		"CURRENT BELIEF (" + s.ID + ", 2026-08-20T10:00:00Z)",
		"Postgres since 2024",
		"NOT RECONCILED\nCompliance basis unsourced.",
		"- Uses Postgres 16, confidence 0.9 — evidence: docs/db.md",
		"- Used MySQL until 2024 — evidence: adr/1.md",
		"[unsynthesised]",
	} {
		if !strings.Contains(v, want) {
			t.Errorf("brief missing %q:\n%s", want, v)
		}
	}
	if strings.Contains(v, s.Content+"\n- ") { // synthesis must not repeat in SUPPORTING
		t.Error("current synthesis repeated in supporting list")
	}
	// only the later claim is unsynthesised
	if line.Item.Properties.OpenQuestions != 1 {
		t.Errorf("openQuestions = %d, want 1", line.Item.Properties.OpenQuestions)
	}
	if !strings.Contains(v, "Billing split out in Q2") {
		t.Error("later claim missing")
	}
	_ = later

	// "None identified" is still shown, so a reader sees it was considered.
	f2 := newFixture(t)
	q := f2.particular("Q")
	c := f2.claim(q.ID, "c", claimOpts{})
	f2.synthesis(q.ID, "settled", "None identified", ts.Add(time.Hour), in(c.ID, dkf.RoleThesis))
	if v := f2.build(Options{})[0].Item.Content.Value; !strings.Contains(v, "NOT RECONCILED\nNone identified") {
		t.Errorf("conventional unresolved not shown:\n%s", v)
	}
}

func TestBriefWithoutSynthesis(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Alpha")
	for _, c := range []string{"one", "two", "three"} {
		f.claim(p.ID, c, claimOpts{})
	}
	line := f.build(Options{})[0]
	v := line.Item.Content.Value
	if !strings.Contains(v, "NO SYNTHESIS YET — 3 assertions not yet reconciled") {
		t.Errorf("no-synthesis brief:\n%s", v)
	}
	if strings.Contains(v, "CURRENT BELIEF") || strings.Contains(v, "NOT RECONCILED") {
		t.Error("empty sections should be omitted")
	}
	if line.Item.Properties.CurrentSynthesis != "" || line.Item.Properties.OpenQuestions != 3 {
		t.Errorf("properties: %+v", line.Item.Properties)
	}
}

func TestProperties(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Project X")
	a := f.claim(p.ID, "one", claimOpts{topics: []string{"db", "architecture"}})
	s := f.synthesis(p.ID, "belief", "n", ts.Add(3*time.Hour), in(a.ID, dkf.RoleThesis))

	line := f.build(Options{SourceURL: "https://github.com/o/r/blob/main"})[0]
	pr := line.Item.Properties
	if pr.Title != "Project X" || pr.ParticularURI != p.URI || pr.LastModified != "2026-08-20T12:00:00Z" {
		t.Errorf("properties: %+v", pr)
	}
	if strings.Join(pr.Topics, ",") != "architecture,db" {
		t.Errorf("topics should be sorted and deduped: %v", pr.Topics)
	}
	if strings.Join(pr.Authors, ",") != "ben" {
		t.Errorf("authors: %v", pr.Authors)
	}
	if pr.URL != "https://github.com/o/r/blob/main/syntheses/"+s.ID+".yaml" {
		t.Errorf("url = %s", pr.URL)
	}
	// collection type specifiers and key order
	body, _ := json.Marshal(pr)
	js := string(body)
	if !strings.Contains(js, `"topics@odata.type":"Collection(String)"`) || !strings.Contains(js, `"authors@odata.type":"Collection(String)"`) {
		t.Errorf("collection specifiers missing: %s", js)
	}
	if !strings.HasPrefix(js, `{"title":`) {
		t.Errorf("property order: %s", js)
	}
	// without a source url there is no url property
	body, _ = json.Marshal(f.build(Options{})[0].Item.Properties)
	if strings.Contains(string(body), `"url"`) {
		t.Errorf("url should be absent: %s", body)
	}
}

func TestURLFallsBackToNewestClaim(t *testing.T) {
	f := newFixture(t)
	p := f.particular("Alpha")
	f.claim(p.ID, "one", claimOpts{})
	newest := f.claim(p.ID, "two", claimOpts{})
	got := f.build(Options{SourceURL: "https://example.com/blob/main/"})[0].Item.Properties.URL
	if got != "https://example.com/blob/main/claims/"+newest.ID+".yaml" {
		t.Errorf("url = %s", got)
	}
}

func TestSchemaConstraints(t *testing.T) {
	reg := NewRegistration("particulars", "", "")
	if reg.Connection.ID != "particulars" || reg.Connection.Name == "" || reg.Connection.Description == "" {
		t.Errorf("connection: %+v", reg.Connection)
	}
	if reg.Schema.BaseType != "microsoft.graph.externalItem" {
		t.Errorf("baseType: %s", reg.Schema.BaseType)
	}
	want := map[string]bool{"title": true, "url": true, "particularUri": true, "scope": true, "topics": true,
		"authors": true, "lastModifiedDateTime": true, "claimCount": true, "openQuestions": true, "currentSynthesis": true}
	labels := map[string]string{}
	for _, p := range reg.Schema.Properties {
		delete(want, p.Name)
		if p.IsSearchable && p.IsRefinable {
			t.Errorf("%s: searchable and refinable are mutually exclusive", p.Name)
		}
		if p.IsExactMatchRequired && p.IsSearchable {
			t.Errorf("%s: isExactMatchRequired requires a non-searchable property", p.Name)
		}
		for _, l := range p.Labels {
			if !p.IsRetrievable {
				t.Errorf("%s: labelled properties must be retrievable", p.Name)
			}
			if prev, dup := labels[l]; dup {
				t.Errorf("label %s used by both %s and %s", l, prev, p.Name)
			}
			labels[l] = p.Name
		}
	}
	if len(want) != 0 {
		t.Errorf("schema missing properties: %v", want)
	}
	if labels["title"] != "title" || labels["url"] != "url" || labels["lastModifiedDateTime"] != "lastModifiedDateTime" {
		t.Errorf("expected semantic labels: %v", labels)
	}
}
