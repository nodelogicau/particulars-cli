package store

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

var ts = time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)

func newWS(t *testing.T) *Workspace {
	t.Helper()
	cfg := NewConfig()
	cfg.Defaults.Source.Author = "ben"
	w, err := Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func mkParticular(t *testing.T, w *Workspace, label string) *dkf.Particular {
	t.Helper()
	p, _, err := w.UpsertParticular(dkf.MintURI("", w.Config.Workspace.ID, dkf.Slugify(label)), label, nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func mkClaim(t *testing.T, w *Workspace, subject, content string) *dkf.Claim {
	t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content, Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	if err := w.Create(c); err != nil {
		t.Fatal(err)
	}
	if err := w.UpsertIndex(c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestInitAndOpen(t *testing.T) {
	w := newWS(t)
	for _, d := range []string{"particulars", "claims", "syntheses"} {
		if fi, err := os.Stat(filepath.Join(w.Root, d)); err != nil || !fi.IsDir() {
			t.Errorf("missing dir %s", d)
		}
	}
	if _, err := os.Stat(w.IndexPath()); err != nil {
		t.Error("index.yaml not created")
	}
	cfgBytes, _ := os.ReadFile(filepath.Join(w.Root, ConfigFile))
	if !strings.Contains(string(cfgBytes), "format: dkf/0.1") || !strings.Contains(string(cfgBytes), "author: ben") {
		t.Errorf("unexpected dkf.yaml:\n%s", cfgBytes)
	}
	w2, err := Open(w.Root)
	if err != nil || w2.Config.Workspace.ID != w.Config.Workspace.ID {
		t.Fatalf("reopen: %v", err)
	}
	if _, err := Init(w.Root, NewConfig()); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("re-init should fail with ErrAlreadyExists, got %v", err)
	}
}

func TestDiscoverPrecedence(t *testing.T) {
	outer := newWS(t)
	inner, err := Init(filepath.Join(outer.Root, "inner"), NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(inner.Root, "claims")
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvWorkspace, "")

	got, err := Discover("")
	if err != nil {
		t.Fatal(err)
	}
	if realpath(got.Root) != realpath(inner.Root) {
		t.Errorf("nearest should win: got %s want %s", got.Root, inner.Root)
	}
	t.Setenv(EnvWorkspace, outer.Root)
	got, _ = Discover("")
	if realpath(got.Root) != realpath(outer.Root) {
		t.Errorf("env should beat walk-up: got %s", got.Root)
	}
	got, _ = Discover(inner.Root)
	if realpath(got.Root) != realpath(inner.Root) {
		t.Errorf("flag should beat env: got %s", got.Root)
	}
	empty := t.TempDir()
	t.Setenv(EnvWorkspace, "")
	if err := os.Chdir(empty); err != nil {
		t.Fatal(err)
	}
	if _, err := Discover(""); !errors.Is(err, ErrNoWorkspace) {
		t.Errorf("expected ErrNoWorkspace, got %v", err)
	}
}

func realpath(p string) string {
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		return p
	}
	return r
}

func TestCreateExclusive(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	c := mkClaim(t, w, p.ID, "hello")
	if err := w.Create(c); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("second create should fail with ErrAlreadyExists, got %v", err)
	}
	bad := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: "x", Context: dkf.Context{Scope: "team"}, Timestamp: ts}
	if err := w.Create(bad); err == nil {
		t.Error("invalid object should not be written")
	}
	if w.Exists(bad.ID) {
		t.Error("invalid object file exists")
	}
}

func TestRetractAppendAndRestore(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	c := mkClaim(t, w, p.ID, "hello")
	path, _ := w.Path(c.ID)
	before, _ := os.ReadFile(path)

	a, err := w.Retract(c.ID, &dkf.Retracted{Timestamp: ts.Add(time.Hour), Reason: "wrong", Source: dkf.Source{Author: "ben"}})
	if err != nil {
		t.Fatal(err)
	}
	if a.GetRetracted() == nil {
		t.Fatal("returned assertion not retracted")
	}
	after, _ := os.ReadFile(path)
	if !bytes.HasPrefix(after, before) {
		t.Errorf("original bytes altered:\n--- before\n%s\n--- after\n%s", before, after)
	}
	if !strings.Contains(string(after), "retracted:\n  timestamp: 2026-08-20T10:00:00Z\n  reason: wrong") {
		t.Errorf("retracted block malformed:\n%s", after)
	}
	if _, err := w.Retract(c.ID, &dkf.Retracted{Timestamp: ts, Reason: "again", Source: dkf.Source{Author: "ben"}}); !errors.Is(err, ErrAlreadyRetracted) {
		t.Errorf("double retract: got %v", err)
	}
	if _, err := w.Retract("clm_0000", &dkf.Retracted{Timestamp: ts, Reason: "x", Source: dkf.Source{Author: "ben"}}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing: got %v", err)
	}
	if _, err := w.Retract(p.ID, &dkf.Retracted{Timestamp: ts, Reason: "x", Source: dkf.Source{Author: "ben"}}); err == nil {
		t.Error("particular should not be retractable")
	}

	// Restore path: a flow-style file cannot take an appended block.
	flow := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: "flow", Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	flowPath, _ := w.Path(flow.ID)
	flowBytes := []byte("{id: " + flow.ID + ", type: claim, subject: " + p.ID + ", content: flow, source: {author: ben}, context: {scope: personal}, timestamp: 2026-08-20T09:00:00Z}\n")
	if err := os.WriteFile(flowPath, flowBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Retract(flow.ID, &dkf.Retracted{Timestamp: ts, Reason: "x", Source: dkf.Source{Author: "ben"}}); err == nil {
		t.Fatal("expected append to a flow mapping to fail")
	}
	restored, _ := os.ReadFile(flowPath)
	if !bytes.Equal(restored, flowBytes) {
		t.Errorf("file not restored after failed retract:\n%s", restored)
	}
}

func TestUpsertParticular(t *testing.T) {
	w := newWS(t)
	uri := "https://example.com/p/project-x"
	p1, created, err := w.UpsertParticular(uri, "ProjectX", []string{"px"})
	if err != nil || !created {
		t.Fatalf("first upsert: %v created=%v", err, created)
	}
	p2, created, err := w.UpsertParticular(uri, "Project X", []string{"PX", "proj-x"})
	if err != nil || created {
		t.Fatalf("second upsert: %v created=%v", err, created)
	}
	if p2.ID != p1.ID || p2.Label != "Project X" {
		t.Errorf("expected same id and new label, got %+v", p2)
	}
	if strings.Join(p2.Aliases, ",") != "px,ProjectX,proj-x" {
		t.Errorf("aliases = %v", p2.Aliases)
	}
	g, _ := w.Load()
	if len(g.Particulars) != 1 {
		t.Errorf("expected 1 particular, got %d", len(g.Particulars))
	}
}

func TestIndexLifecycle(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	c := mkClaim(t, w, p.ID, "hello")
	d, err := w.CheckIndex()
	if err != nil || !d.Clean() {
		t.Fatalf("index should be clean after incremental upserts: %+v %v", d, err)
	}
	idx, _, _ := w.ReadIndex()
	if len(idx.Entries) != 2 || idx.Entries[0].ID != c.ID || idx.Entries[1].ID != p.ID {
		t.Errorf("entries not sorted by id: %+v", idx.Entries)
	}

	// Hand-added file → drift.
	extra := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: p.ID, Content: "by hand", Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	if err := w.Create(extra); err != nil {
		t.Fatal(err)
	}
	d, _ = w.CheckIndex()
	if d.Clean() || len(d.Missing) != 1 || d.Missing[0] != extra.ID {
		t.Errorf("expected drift naming %s, got %+v", extra.ID, d)
	}

	// Conflict-marker garbage → rebuild recovers.
	if err := os.WriteFile(w.IndexPath(), []byte("<<<<<<< HEAD\nformat: dkf/0.1\n=======\ngarbage\n>>>>>>> x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}
	d, _ = w.CheckIndex()
	if !d.Clean() {
		t.Errorf("rebuild should be clean: %+v", d)
	}

	// Missing index → upsert rebuilds; check reports missing.
	_ = os.Remove(w.IndexPath())
	d, _ = w.CheckIndex()
	if d.Clean() || len(d.Missing) != 3 {
		t.Errorf("missing index should report all entries missing: %+v", d)
	}
	if err := w.UpsertIndex(c); err != nil {
		t.Fatal(err)
	}
	d, _ = w.CheckIndex()
	if !d.Clean() {
		t.Errorf("upsert on missing index should rebuild: %+v", d)
	}

	// Retraction reflected.
	if _, err := w.Retract(c.ID, &dkf.Retracted{Timestamp: ts, Reason: "r", Source: dkf.Source{Author: "ben"}}); err != nil {
		t.Fatal(err)
	}
	a, _ := w.ReadAssertion(c.ID)
	if err := w.UpsertIndex(a); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(w.IndexPath())
	if !strings.Contains(string(data), "retracted: true") {
		t.Errorf("index should mark retraction:\n%s", data)
	}
}

func TestLoadProblems(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	_ = mkClaim(t, w, p.ID, "ok")
	_ = os.WriteFile(filepath.Join(w.Dir(dkf.TypeClaim), "clm_bad.yaml"), []byte("id: clm_other\ntype: claim\n"), 0o644)
	_ = os.WriteFile(filepath.Join(w.Dir(dkf.TypeClaim), "clm_junk.yaml"), []byte(": : :\n"), 0o644)
	_ = os.WriteFile(filepath.Join(w.Dir(dkf.TypeClaim), "syn_wrongdir.yaml"), []byte("id: syn_wrongdir\ntype: synthesis\n"), 0o644)
	g, err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Assertions) != 1 {
		t.Errorf("expected 1 good assertion, got %d", len(g.Assertions))
	}
	codes := map[string]int{}
	for _, pr := range g.Problems {
		codes[pr.Code]++
	}
	if codes["id_mismatch"] != 1 || codes["parse_error"] != 1 || codes["type_mismatch"] != 1 {
		t.Errorf("problems = %+v", g.Problems)
	}
	if g.Err() == nil {
		t.Error("Err() should be non-nil")
	}
}

func TestConfigBaseURI(t *testing.T) {
	cfg := NewConfig()
	cfg.Workspace.BaseURI = "https://example.com/particulars"
	if _, err := Init(filepath.Join(t.TempDir(), "kb"), cfg); !errors.Is(err, ErrInvalidBaseURI) {
		t.Errorf("init without trailing slash should fail with ErrInvalidBaseURI, got %v", err)
	}
	cfg.Workspace.BaseURI = "https://example.com/particulars/"
	w, err := Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(filepath.Join(w.Root, "merges")); err != nil || !fi.IsDir() {
		t.Error("init should create merges/")
	}
	_ = os.WriteFile(filepath.Join(w.Root, ConfigFile), []byte("format: dkf/0.1\nworkspace:\n  id: x\n  base-uri: https://example.com/p\n"), 0o644)
	if _, err := Open(w.Root); !errors.Is(err, ErrInvalidBaseURI) {
		t.Errorf("open with bad base-uri: %v", err)
	}
}

func TestMergesAndClasses(t *testing.T) {
	w := newWS(t)
	a := mkParticular(t, w, "A")
	b := mkParticular(t, w, "B")
	c := mkParticular(t, w, "C")
	d := mkParticular(t, w, "D")
	src := dkf.Source{Author: "ben"}
	m1, err := w.CreateMerge(b.URI, a.URI, "same", src, ts)
	if err != nil {
		t.Fatal(err)
	}
	if m1.URIs[0] != a.URI || m1.URIs[1] != b.URI {
		t.Errorf("uris should be sorted: %v", m1.URIs)
	}
	if _, err := w.CreateMerge(a.URI, b.URI, "", src, ts); err == nil {
		t.Error("duplicate merge should fail")
	}
	if _, err := w.CreateMerge(a.URI, a.URI, "", src, ts); err == nil {
		t.Error("self merge should fail")
	}
	// Bridge through a foreign URI: B <-> U <-> C.
	foreign := "https://www.wikidata.org/entity/Q1"
	if _, err := w.CreateMerge(b.URI, foreign, "", src, ts); err != nil {
		t.Fatal(err)
	}
	m3, err := w.CreateMerge(foreign, c.URI, "", src, ts)
	if err != nil {
		t.Fatal(err)
	}
	g, _ := w.Load()
	if got := strings.Join(g.ClassOf(a.ID), ","); got != strings.Join([]string{a.ID, b.ID, c.ID}, ",") && len(g.ClassOf(a.ID)) != 3 {
		t.Errorf("class of A = %v", g.ClassOf(a.ID))
	}
	if len(g.ClassOf(a.ID)) != 3 || len(g.ClassOf(d.ID)) != 1 || g.ClassOf(d.ID)[0] != d.ID {
		t.Errorf("classes: A=%v D=%v", g.ClassOf(a.ID), g.ClassOf(d.ID))
	}
	if g.MergeBetween(foreign, b.URI) == nil {
		t.Error("MergeBetween should find the bridge in either order")
	}
	// Retract the C edge → C leaves the class; merge file stays.
	if _, err := w.Retract(m3.ID, &dkf.Retracted{Timestamp: ts, Reason: "wrong", Source: src, SupersededBy: "clm_01x"}); err == nil {
		t.Error("superseded-by on a merge should be rejected")
	}
	r, err := w.Retract(m3.ID, &dkf.Retracted{Timestamp: ts, Reason: "wrong", Source: src})
	if err != nil || r.GetRetracted() == nil {
		t.Fatalf("retract merge: %v", err)
	}
	if err := w.UpsertIndex(r); err != nil {
		t.Fatal(err)
	}
	g, _ = w.Load()
	if len(g.ClassOf(a.ID)) != 2 || len(g.ClassOf(c.ID)) != 1 {
		t.Errorf("after retract: A=%v C=%v", g.ClassOf(a.ID), g.ClassOf(c.ID))
	}
	if _, err := os.Stat(filepath.Join(w.Dir(dkf.TypeMerge), m3.ID+".yaml")); err != nil {
		t.Error("retracted merge file must remain")
	}
	idx, _, _ := w.ReadIndex()
	found := false
	for _, e := range idx.Entries {
		if e.ID == m3.ID {
			found = true
			if e.Type != dkf.TypeMerge || len(e.URIs) != 2 || !e.Retracted {
				t.Errorf("merge index entry: %+v", e)
			}
		}
	}
	if !found {
		t.Error("merge missing from index")
	}
	if d, _ := w.CheckIndex(); !d.Clean() {
		t.Errorf("index should be clean: %+v", d)
	}
}

func TestPointerDiscovery(t *testing.T) {
	repo := t.TempDir()
	ws, err := Init(filepath.Join(repo, "knowledge"), NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WritePointer(repo, "knowledge"); err != nil {
		t.Fatal(err)
	}
	if _, err := WritePointer(repo, "knowledge"); err != nil {
		t.Errorf("identical rewrite should succeed: %v", err)
	}
	if _, err := WritePointer(repo, "elsewhere"); !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("differing pointer should be refused: %v", err)
	}
	sub := filepath.Join(repo, "src", "deep")
	_ = os.MkdirAll(sub, 0o755)
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	t.Setenv(EnvWorkspace, "")
	_ = os.Chdir(sub)
	w, res, err := DiscoverWith("")
	if err != nil || realpath(w.Root) != realpath(ws.Root) || res.Via != "pointer" || realpath(res.Pointer) != realpath(filepath.Join(repo, PointerFile)) {
		t.Errorf("pointer discovery: %v %+v", err, res)
	}
	// dkf.yaml beats a pointer in the same directory.
	both := t.TempDir()
	_, _ = Init(both, NewConfig())
	_, _ = WritePointer(both, "nowhere")
	_ = os.Chdir(both)
	if _, res, err := DiscoverWith(""); err != nil || res.Via != "dkf.yaml" {
		t.Errorf("dkf.yaml should win: %v %+v", err, res)
	}
	// Dangling pointer names both paths; comments and blank lines are skipped; absolute targets work.
	dangling := t.TempDir()
	_ = os.WriteFile(filepath.Join(dangling, PointerFile), []byte("# comment\n\nmissing\n"), 0o644)
	_ = os.Chdir(dangling)
	_, _, err = DiscoverWith("")
	if !errors.Is(err, ErrNoWorkspace) || !strings.Contains(err.Error(), PointerFile) || !strings.Contains(err.Error(), "missing") {
		t.Errorf("dangling: %v", err)
	}
	absrepo := t.TempDir()
	_ = os.WriteFile(filepath.Join(absrepo, PointerFile), []byte("# abs\n"+ws.Root+"\n"), 0o644)
	_ = os.Chdir(absrepo)
	if w, res, err := DiscoverWith(""); err != nil || realpath(w.Root) != realpath(ws.Root) || res.Via != "pointer" {
		t.Errorf("absolute pointer: %v %+v", err, res)
	}
	// No chaining: a pointer to a directory that only has another pointer is dangling.
	chain := t.TempDir()
	_ = os.MkdirAll(filepath.Join(chain, "mid"), 0o755)
	_ = os.WriteFile(filepath.Join(chain, PointerFile), []byte("mid\n"), 0o644)
	_ = os.WriteFile(filepath.Join(chain, "mid", PointerFile), []byte(ws.Root+"\n"), 0o644)
	_ = os.Chdir(chain)
	if _, _, err := DiscoverWith(""); !errors.Is(err, ErrNoWorkspace) {
		t.Errorf("chained pointer should not resolve: %v", err)
	}
	// Env and flag still win.
	t.Setenv(EnvWorkspace, ws.Root)
	if _, res, _ := DiscoverWith(""); res.Via != "env" {
		t.Errorf("env: %+v", res)
	}
	if _, res, _ := DiscoverWith(ws.Root); res.Via != "flag" {
		t.Errorf("flag: %+v", res)
	}
}

func mkScopedClaim(t *testing.T, w *Workspace, subject, content string, sc dkf.Scope) *dkf.Claim {
	t.Helper()
	c := &dkf.Claim{ID: dkf.NewID(dkf.TypeClaim), Subject: subject, Content: content,
		Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: sc}, Timestamp: ts}
	if err := w.Create(c); err != nil {
		t.Fatal(err)
	}
	if err := w.UpsertIndex(c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestEffectiveScope(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	priv := mkScopedClaim(t, w, p.ID, "a", dkf.ScopePersonal)
	other := mkScopedClaim(t, w, p.ID, "b", dkf.ScopePersonal)
	src := dkf.Source{Author: "ben"}
	now := time.Now().UTC()

	// No promotions: effective == asserted, everywhere.
	g, err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := g.EffectiveScope(priv.ID); got != dkf.ScopePersonal {
		t.Errorf("unpromoted claim: %q", got)
	}
	if len(g.Promotions) != 0 || len(g.PromotionsFor(priv.ID)) != 0 {
		t.Error("a workspace with no publishes/ should load no promotions")
	}

	// The widest promotion wins, regardless of the order written.
	if _, err := w.CreatePromotion([]string{priv.ID}, dkf.ScopePublic, "", src, now); err != nil {
		t.Fatal(err)
	}
	second, err := w.CreatePromotion([]string{priv.ID}, dkf.ScopeOrganisation, "redundant but valid", src, now)
	if err != nil {
		t.Fatalf("a narrower-than-existing but not narrowing promotion must be allowed: %v", err)
	}
	g, _ = w.Load()
	if got := g.EffectiveScope(priv.ID); got != dkf.ScopePublic {
		t.Errorf("widest promotion should win: %q", got)
	}
	if got := g.EffectiveScope(other.ID); got != dkf.ScopePersonal {
		t.Errorf("promotion must not leak to an uncovered object: %q", got)
	}
	if n := len(g.PromotionsFor(priv.ID)); n != 2 {
		t.Errorf("PromotionsFor: want 2, got %d", n)
	}

	// Retracting a promotion reverts what it granted.
	if _, err := w.Retract(second.ID, &dkf.Retracted{Timestamp: now, Reason: "undo", Source: src}); err != nil {
		t.Fatal(err)
	}
	g, _ = w.Load()
	if got := g.EffectiveScope(priv.ID); got != dkf.ScopePublic {
		t.Errorf("the wider promotion still stands: %q", got)
	}
	for _, pr := range g.SortedPromotions() {
		if pr.Retracted == nil {
			if _, err := w.Retract(pr.ID, &dkf.Retracted{Timestamp: now, Reason: "undo", Source: src}); err != nil {
				t.Fatal(err)
			}
		}
	}
	g, _ = w.Load()
	if got := g.EffectiveScope(priv.ID); got != dkf.ScopePersonal {
		t.Errorf("with every promotion retracted the claim reverts: %q", got)
	}
}

func TestPromotionMayOnlyWiden(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	pub := mkScopedClaim(t, w, p.ID, "already public", dkf.ScopePublic)
	priv := mkScopedClaim(t, w, p.ID, "private", dkf.ScopePersonal)
	src := dkf.Source{Author: "ben"}
	now := time.Now().UTC()

	if _, err := w.CreatePromotion([]string{pub.ID}, dkf.ScopeOrganisation, "", src, now); err == nil {
		t.Error("narrowing a public claim to organisation must be refused")
	} else if !strings.Contains(err.Error(), "only widen") {
		t.Errorf("the error should say why: %v", err)
	}
	// Refused before writing: nothing lands on disk.
	g, _ := w.Load()
	if len(g.Promotions) != 0 {
		t.Error("a refused promotion must not be written")
	}
	// Widening a mixed set is fine as long as no member is narrowed.
	if _, err := w.CreatePromotion([]string{priv.ID}, dkf.ScopePublic, "", src, now); err != nil {
		t.Fatal(err)
	}
	if _, err := w.CreatePromotion([]string{priv.ID, pub.ID}, dkf.ScopePublic, "", src, now); err != nil {
		t.Errorf("promoting both to public is not narrowing either: %v", err)
	}
	// Only assertions can be promoted.
	if _, err := w.CreatePromotion([]string{p.ID}, dkf.ScopePublic, "", src, now); err == nil {
		t.Error("a particular carries no scope and must be refused")
	}
	if _, err := w.CreatePromotion([]string{"clm_01916f03-b680-71a3-974f-9401ba374e1f"}, dkf.ScopePublic, "", src, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("an unknown id should be ErrNotFound, got %v", err)
	}
	if _, err := w.CreatePromotion(nil, dkf.ScopePublic, "", src, now); err == nil {
		t.Error("an empty promotion must be refused")
	}
}

func TestPromotionIndexEntry(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	a := mkScopedClaim(t, w, p.ID, "one", dkf.ScopePersonal)
	b := mkScopedClaim(t, w, p.ID, "two", dkf.ScopePersonal)
	pr, err := w.CreatePromotion([]string{a.ID, b.ID}, dkf.ScopeOrganisation, "", dkf.Source{Author: "ben"}, ts)
	if err != nil {
		t.Fatal(err)
	}
	idx, _, err := w.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	var got *Entry
	for i := range idx.Entries {
		if idx.Entries[i].ID == pr.ID {
			got = &idx.Entries[i]
		}
	}
	if got == nil {
		t.Fatal("the promotion should be indexed")
	}
	if got.Type != dkf.TypePublish || got.Scope != string(dkf.ScopeOrganisation) || len(got.Claims) != 2 {
		t.Fatalf("promotion entry: %+v", got)
	}

	// Effective scope must be computable from index.yaml alone: a consumer
	// that never opens an object file gets the same answer as the graph.
	fromIndex := map[string]dkf.Scope{}
	for _, e := range idx.Entries {
		if e.Retracted || e.Scope == "" {
			continue
		}
		if e.Type == dkf.TypePublish {
			for _, id := range e.Claims {
				if dkf.ScopeRank(dkf.Scope(e.Scope)) > dkf.ScopeRank(fromIndex[id]) {
					fromIndex[id] = dkf.Scope(e.Scope)
				}
			}
			continue
		}
		if dkf.ScopeRank(dkf.Scope(e.Scope)) > dkf.ScopeRank(fromIndex[e.ID]) {
			fromIndex[e.ID] = dkf.Scope(e.Scope)
		}
	}
	g, err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{a.ID, b.ID} {
		if fromIndex[id] != g.EffectiveScope(id) {
			t.Errorf("%s: index says %q, graph says %q", id, fromIndex[id], g.EffectiveScope(id))
		}
	}
	// The index rebuild is stable with promotions present.
	if _, err := w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}
	if diff, err := w.CheckIndex(); err != nil || !diff.Clean() {
		t.Errorf("index should be current after rebuild: %+v %v", diff, err)
	}
}

func TestUnknownIndexEntriesSurvive(t *testing.T) {
	w := newWS(t)
	p := mkParticular(t, w, "Project X")
	mkClaim(t, w, p.ID, "hello")

	// A newer conforming writer adds an entry type this implementation has
	// never heard of. Inject it as that writer would: in ascending-id order.
	future := "  - id: sig_01b00000-0000-7000-8000-000000000001\n    type: signature\n    signer: did:example:ben\n    payload: canonical-yaml\n"
	data, err := os.ReadFile(w.IndexPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(w.IndexPath(), append(data, []byte(future)...), 0o644); err != nil {
		t.Fatal(err)
	}

	// Reading ignores it: no consumer sees it among the typed entries.
	idx, _, err := w.ReadIndex()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range idx.Entries {
		if e.ID == "sig_01b00000-0000-7000-8000-000000000001" {
			t.Fatal("an unknown type must not surface as a typed entry")
		}
	}
	if len(idx.Unknown) != 1 {
		t.Fatalf("unknown rows preserved: got %d", len(idx.Unknown))
	}

	// The check does not fail on evidence of a newer writer.
	d, err := w.CheckIndex()
	if err != nil || !d.Clean() {
		t.Fatalf("unknown entries are never drift: %+v %v", d, err)
	}

	// A rebuild preserves it, unchanged, in ascending-id order.
	if _, err := w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := os.ReadFile(w.IndexPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rebuilt), "type: signature") || !strings.Contains(string(rebuilt), "signer: did:example:ben") {
		t.Fatalf("rebuild dropped or altered the unknown entry:\n%s", rebuilt)
	}
	sigPos := strings.Index(string(rebuilt), "id: sig_")
	parPos := strings.Index(string(rebuilt), "id: par_")
	if sigPos < parPos {
		t.Errorf("sig_ sorts after par_ in ascending-id order:\n%s", rebuilt)
	}

	// An incremental update carries it through too.
	c2 := mkClaim(t, w, p.ID, "another")
	after, _ := os.ReadFile(w.IndexPath())
	if !strings.Contains(string(after), "type: signature") || !strings.Contains(string(after), c2.ID) {
		t.Fatalf("upsert must add the new entry and keep the unknown one:\n%s", after)
	}
	// And a rebuild is idempotent over it.
	if _, err := w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(w.IndexPath())
	if !bytes.Equal(after, again) {
		t.Errorf("rebuild is not idempotent with an unknown entry present:\n%s\n---\n%s", after, again)
	}

	// Real drift still fails: delete the claim's entry by rebuilding from a
	// graph, then hand-removing a row.
	trimmed := strings.Replace(string(again), "  - id: "+c2.ID+"\n", "  - id: "+c2.ID+"-gone\n", 1)
	if err := os.WriteFile(w.IndexPath(), []byte(trimmed), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, _ := w.CheckIndex(); d.Clean() {
		t.Error("a genuinely stale index must still fail the check")
	}
}
