package store

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// idAt mints an id of type t whose UUIDv7 instant is at, so a test can place
// objects in chosen months.
func idAt(t dkf.Type, at time.Time) string {
	var r [8]byte
	_, _ = rand.Read(r[:])
	ms := at.UnixMilli()
	return fmt.Sprintf("%s_%08x-%04x-7%03x-%x%03x-%012x", dkf.Prefix(t), ms>>16, ms&0xffff,
		int(r[0])<<4|int(r[1])>>4&0xfff, 8|r[2]&3, int(r[3])<<4&0xfff|int(r[4])>>4, int64(r[5])<<32|int64(r[6])<<16|int64(r[7]))
}

func month(y int, m time.Month) time.Time { return time.Date(y, m, 10, 12, 0, 0, 0, time.UTC) }

func newShardedWS(t *testing.T) *Workspace {
	t.Helper()
	cfg := NewConfig()
	cfg.Format = dkf.Format02
	cfg.Defaults.Source.Author = "ben"
	w, err := Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// shardedFixture mints and retracts through the store as the verbs do:
// Create or Retract, then UpsertIndex.
type shardedFixture struct {
	t       *testing.T
	w       *Workspace
	subject *dkf.Particular
}

func (f *shardedFixture) particular(at time.Time) {
	f.t.Helper()
	p := &dkf.Particular{ID: idAt(dkf.TypeParticular, at), URI: dkf.MintURI("", f.w.Config.Workspace.ID, "subject"), Label: "Subject"}
	f.mustCreate(p)
	f.subject = p
}

func (f *shardedFixture) claim(at time.Time) string {
	f.t.Helper()
	return f.claimID(idAt(dkf.TypeClaim, at), at)
}

func (f *shardedFixture) claimID(id string, at time.Time) string {
	f.t.Helper()
	c := &dkf.Claim{ID: id, Subject: f.subject.ID, Content: "c " + id, Evidential: dkf.EvidentialObserved,
		Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: at}
	f.mustCreate(c)
	return id
}

func (f *shardedFixture) mustCreate(obj dkf.Object) {
	f.t.Helper()
	if err := f.w.Create(obj); err != nil {
		f.t.Fatal(err)
	}
	if err := f.w.UpsertIndex(obj); err != nil {
		f.t.Fatal(err)
	}
}

func (f *shardedFixture) retract(id string, at time.Time) error {
	r := &dkf.Retracted{Timestamp: at, Reason: "wrong", Source: dkf.Source{Author: "ben"}}
	updated, err := f.w.Retract(id, r)
	if err != nil {
		return err
	}
	return f.w.UpsertIndex(updated)
}

// indexFiles reads every index document on disk, keyed by root-relative path.
func indexFiles(t *testing.T, w *Workspace) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	if b, err := os.ReadFile(w.IndexPath()); err == nil {
		out[IndexFile] = b
	}
	entries, _ := os.ReadDir(filepath.Join(w.Root, "index"))
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(w.Root, "index", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out["index/"+e.Name()] = b
	}
	return out
}

// rebuilt is what a rebuild of the files would write, preserving whatever
// the committed documents carry that this implementation does not know.
func rebuilt(t *testing.T, w *Workspace) map[string][]byte {
	t.Helper()
	g, err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := BuildSharded(g, w.Config.Format)
	var committed []*Index
	if root, _, err := w.readRoot(); err == nil {
		committed = append(committed, root)
		for _, seg := range root.Segments {
			if d, _, err := w.readDoc(seg.Path); err == nil {
				committed = append(committed, d)
			}
		}
	}
	s.preserve(committed)
	out := map[string][]byte{}
	for rel, d := range s.Docs {
		b, err := EncodeIndex(d)
		if err != nil {
			t.Fatal(err)
		}
		out[rel] = b
	}
	b, err := EncodeIndex(s.Root)
	if err != nil {
		t.Fatal(err)
	}
	out[IndexFile] = b
	return out
}

// The upstream shape, the rollover rules, and the bounded-write promise, by
// stepping one workspace through nine months. After every step the index on
// disk equals a rebuild, no sealed segment has changed, and the only files
// written are the root, the legacy document, and segments for months that
// just sealed.
func TestShardedIndexSequence(t *testing.T) {
	w := newShardedWS(t)
	f := &shardedFixture{t: t, w: w}
	var a, b, c string
	steps := []struct {
		name string
		do   func()
		// may lists index documents the step is allowed to write.
		may []string
	}{
		{"August mints", func() { f.particular(month(2026, 8)); a = f.claim(month(2026, 8)); b = f.claim(month(2026, 8)) }, []string{IndexFile}},
		{"September mint and a retraction of an August claim", func() {
			c = f.claim(month(2026, 9))
			if err := f.retract(a, month(2026, 9)); err != nil {
				t.Fatal(err)
			}
		}, []string{IndexFile}},
		{"October mint seals August", func() { f.claim(month(2026, 10)) }, []string{IndexFile, "index/2026-08.yaml"}},
		{"retracting a sealed August claim touches only the root", func() {
			if err := f.retract(b, month(2026, 10)); err != nil {
				t.Fatal(err)
			}
		}, []string{IndexFile}},
		{"a preserved row of an unknown type", func() { injectUnknown(t, w, "rdx"+idAt("", month(2026, 10))) }, []string{IndexFile}},
		{"November mint seals September", func() { f.claim(month(2026, 11)) }, []string{IndexFile, "index/2026-09.yaml"}},
		{"a retraction dated ahead of the newest id", func() {
			if err := f.retract(c, month(2026, 12)); err != nil {
				t.Fatal(err)
			}
		}, []string{IndexFile}},
		{"an April 2027 mint seals five months at once", func() { f.claim(month(2027, 4)) },
			[]string{IndexFile, "index/2026-10.yaml", "index/2026-11.yaml", "index/2026-12.yaml"}},
		{"a legacy id", func() { f.claimID("clm_07m3zp9s2q1r4t8v", month(2027, 4)) }, []string{IndexFile, LegacyIndex}},
		{"retracting the legacy claim", func() {
			if err := f.retract("clm_07m3zp9s2q1r4t8v", month(2027, 4)); err != nil {
				t.Fatal(err)
			}
		}, []string{IndexFile}},
		{"updating a particular minted in a sealed month writes no index document", func() {
			if _, _, err := w.UpsertParticular(f.subject.URI, "Subject", []string{"subj"}); err != nil {
				t.Fatal(err)
			}
		}, nil},
	}
	for _, s := range steps {
		before := indexFiles(t, w)
		s.do()
		after := indexFiles(t, w)
		for rel, b := range after {
			if old, ok := before[rel]; (!ok || !bytes.Equal(old, b)) && !slices.Contains(s.may, rel) {
				t.Errorf("%s: wrote %s, which it may not", s.name, rel)
			}
		}
		for rel, b := range before {
			if rel != IndexFile && rel != LegacyIndex && !bytes.Equal(after[rel], b) {
				t.Errorf("%s: sealed segment %s changed", s.name, rel)
			}
		}
		want := rebuilt(t, w)
		for rel, b := range want {
			if !bytes.Equal(after[rel], b) {
				t.Errorf("%s: %s differs from a rebuild:\n--- on disk\n%s--- rebuild\n%s", s.name, rel, after[rel], b)
			}
		}
		for rel := range after {
			if _, ok := want[rel]; !ok {
				t.Errorf("%s: %s exists but a rebuild writes no such document", s.name, rel)
			}
		}
		if d, err := w.CheckIndex(); err != nil || !d.Clean() {
			t.Errorf("%s: check %+v %v", s.name, d, err)
		}
	}

	// The upstream shape at the end: legacy first, sealed months ascending,
	// a retraction-only month, and the unknown row counted where it sealed.
	root, _, _ := w.readRoot()
	var paths []string
	for _, s := range root.Segments {
		paths = append(paths, fmt.Sprintf("%s=%d", s.Path, s.Count))
	}
	if got, want := strings.Join(paths, " "), "index/legacy.yaml=1 index/2026-08.yaml=3 index/2026-09.yaml=1 index/2026-10.yaml=2 index/2026-11.yaml=1 index/2026-12.yaml=0"; got != want {
		t.Errorf("segments %s, want %s", got, want)
	}
	dec, _, _ := w.readDoc("index/2026-12.yaml")
	if len(dec.Entries) != 0 || !slices.Equal(dec.Retracted, []string{c}) {
		t.Errorf("December, retractions only: %+v", dec)
	}
	sep, _, _ := w.readDoc("index/2026-09.yaml")
	if !slices.Equal(sep.Retracted, []string{a}) {
		t.Errorf("September carries the August claim retracted in September: %v", sep.Retracted)
	}
	for _, e := range append(root.Entries, sep.Entries...) {
		if e.Retracted {
			t.Errorf("%s carries a retracted flag", e.ID)
		}
	}
	if !slices.Contains(root.Retracted, "clm_07m3zp9s2q1r4t8v") {
		t.Errorf("a legacy object's retraction belongs to the root of its month: %v", root.Retracted)
	}

	// The guards: April 2027 is current, so March is open and January sealed.
	if err := f.retract(f.claim(month(2027, 3)), month(2027, 1)); !errors.Is(err, ErrSealedMonth) {
		t.Errorf("a retraction dated into a sealed month: %v", err)
	}
	early := &dkf.Claim{ID: idAt(dkf.TypeClaim, month(2027, 1)), Subject: f.subject.ID, Content: "late", Evidential: dkf.EvidentialObserved,
		Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: month(2027, 1)}
	before := indexFiles(t, w)
	if err := w.Create(early); !errors.Is(err, ErrSealedMonth) || !strings.Contains(err.Error(), "2027-01") || !strings.Contains(err.Error(), "2027-04") {
		t.Errorf("a mint into a sealed month: %v", err)
	}
	if _, err := os.Stat(filepath.Join(w.Root, "claims", "2027-01", early.ID+".yaml")); err == nil {
		t.Error("a refused mint was written")
	}
	if after := indexFiles(t, w); !mapsEqual(before, after) {
		t.Error("a refused mint changed the index")
	}
}

func mapsEqual(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if !bytes.Equal(b[k], v) {
			return false
		}
	}
	return true
}

// injectUnknown adds an entry of a type this implementation does not know to
// the root, as a newer writer would.
func injectUnknown(t *testing.T, w *Workspace, id string) {
	t.Helper()
	root, _, err := w.readRoot()
	if err != nil {
		t.Fatal(err)
	}
	var n yaml.Node
	if err := yaml.Unmarshal([]byte("id: "+id+"\ntype: redaction\n"), &n); err != nil {
		t.Fatal(err)
	}
	root.Unknown = append(root.Unknown, n.Content[0])
	if err := w.writeDoc(IndexFile, root); err != nil {
		t.Fatal(err)
	}
}

// Rebuild writes what is absent and never rewrites a sealed segment; the
// check compares counts first, and validation is where a retraction dated
// into a sealed month is found.
func TestShardedRebuildAndCheck(t *testing.T) {
	w := newShardedWS(t)
	f := &shardedFixture{t: t, w: w}
	f.particular(month(2026, 8))
	aug := f.claim(month(2026, 8))
	f.claim(month(2026, 9))
	f.claim(month(2026, 10))
	sep := f.claim(month(2026, 11)) // seals August and September

	// From nothing.
	if err := os.RemoveAll(filepath.Join(w.Root, "index")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(w.IndexPath()); err != nil {
		t.Fatal(err)
	}
	if _, err := w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Fatalf("a rebuild from nothing: %+v %v", d, err)
	}
	_ = sep

	// A late merge: a file arrives in sealed August behind the index's back.
	augSeg := filepath.Join(w.Root, "index", "2026-08.yaml")
	augBytes, _ := os.ReadFile(augSeg)
	late := &dkf.Claim{ID: idAt(dkf.TypeClaim, month(2026, 8)), Subject: f.subject.ID, Content: "late", Evidential: dkf.EvidentialObserved,
		Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: month(2026, 8)}
	data, _ := dkf.Encode(late)
	latePath := filepath.Join(w.Root, "claims", "2026-08", late.ID+".yaml")
	if err := os.WriteFile(latePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if d, _ := w.CheckIndex(); !slices.Equal(d.LateMerge, []string{late.ID}) {
		t.Errorf("late merge not reported: %+v", d)
	}
	var sealed *SealedSegmentsError
	if _, err := w.RebuildIndex(); !errors.As(err, &sealed) || !slices.Equal(sealed.Paths, []string{"index/2026-08.yaml"}) {
		t.Errorf("rebuild over a late merge: %v", err)
	}
	if b, _ := os.ReadFile(augSeg); !bytes.Equal(b, augBytes) {
		t.Error("rebuild rewrote a sealed segment")
	}
	if d, _ := w.CheckIndex(); !slices.Equal(d.LateMerge, []string{late.ID}) {
		t.Errorf("after a refused rebuild the late merge must stay visible: %+v", d)
	}
	if err := os.Remove(latePath); err != nil {
		t.Fatal(err)
	}

	// A matching count is not parsed: a segment altered without changing its
	// count goes unnoticed by the check, by design.
	altered := strings.Replace(string(augBytes), "type: claim", "type: claim\n    scope: public", 1)
	if err := os.WriteFile(augSeg, []byte(altered), 0o644); err != nil {
		t.Fatal(err)
	}
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Errorf("a counted-equal segment was parsed: %+v %v", d, err)
	}
	if err := os.WriteFile(augSeg, augBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// A missing segment.
	sepSeg := filepath.Join(w.Root, "index", "2026-09.yaml")
	sepBytes, _ := os.ReadFile(sepSeg)
	if err := os.Remove(sepSeg); err != nil {
		t.Fatal(err)
	}
	if d, _ := w.CheckIndex(); !slices.Equal(d.SegmentsMissing, []string{"index/2026-09.yaml"}) {
		t.Errorf("missing segment: %+v", d)
	}
	if err := os.WriteFile(sepSeg, sepBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// A retraction written to a file but not to the index.
	open := f.claim(month(2026, 11))
	obj, _ := w.Read(open)
	r := obj.(dkf.Retractable)
	r.SetRetracted(&dkf.Retracted{Timestamp: month(2026, 11), Reason: "wrong", Source: dkf.Source{Author: "ben"}})
	data, _ = dkf.Encode(obj)
	openPath, _ := w.Path(open)
	if err := os.WriteFile(openPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if d, _ := w.CheckIndex(); !slices.Equal(d.RetractedMissing, []string{open}) {
		t.Errorf("retraction after commit: %+v", d)
	}
	if _, err := w.RebuildIndex(); err != nil {
		t.Fatal(err)
	}

	// A retraction dated into sealed August: the count is unchanged, so the
	// check passes, and validation's comparison finds it.
	obj, _ = w.Read(aug)
	obj.(dkf.Retractable).SetRetracted(&dkf.Retracted{Timestamp: month(2026, 8), Reason: "wrong", Source: dkf.Source{Author: "ben"}})
	data, _ = dkf.Encode(obj)
	augPath, _ := w.Path(aug)
	if err := os.WriteFile(augPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Errorf("the check should not see a sealed-month retraction: %+v %v", d, err)
	}
	g, err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
	ms, err := w.SealedRetractions(g)
	if err != nil || len(ms) != 1 || ms[0].ID != aug || ms[0].Segment != "index/2026-08.yaml" || !ms[0].InFiles {
		t.Errorf("sealed retraction: %+v %v", ms, err)
	}
}

// A workspace with only legacy ids has no sealed month: the root carries
// every retraction and index/legacy.yaml every entry.
func TestShardedLegacyOnly(t *testing.T) {
	w := newShardedWS(t)
	f := &shardedFixture{t: t, w: w}
	p := &dkf.Particular{ID: "par_07m3zp9s2q1r4t8a", URI: dkf.MintURI("", w.Config.Workspace.ID, "subject"), Label: "Subject"}
	f.mustCreate(p)
	f.subject = p
	f.claimID("clm_07m3zp9s2q1r4t8b", month(2020, 1))
	if err := f.retract("clm_07m3zp9s2q1r4t8b", month(2020, 2)); err != nil {
		t.Fatal(err)
	}
	root, _, _ := w.readRoot()
	if len(root.Entries) != 0 || !slices.Equal(root.Retracted, []string{"clm_07m3zp9s2q1r4t8b"}) || len(root.Segments) != 1 || root.Segments[0] != (SegmentRef{LegacyIndex, 2}) {
		t.Errorf("legacy-only root: %+v", root)
	}
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Errorf("check: %+v %v", d, err)
	}
	// The first UUIDv7 id gives the workspace a current month; the 2020
	// retraction then seals into its own segment.
	f.claim(month(2026, 11))
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Errorf("after the first minted id: %+v %v", d, err)
	}
	if _, err := os.Stat(filepath.Join(w.Root, "index", "2020-02.yaml")); err != nil {
		t.Errorf("the 2020 retraction did not seal: %v", err)
	}
}

// A clock two days slow across a month boundary mints into the open month
// before the newest id's, and nothing is refused.
func TestShardedGuardAcceptsAClockDaysSlow(t *testing.T) {
	w := newShardedWS(t)
	f := &shardedFixture{t: t, w: w}
	f.particular(time.Date(2026, 12, 2, 9, 0, 0, 0, time.UTC))
	id := f.claim(time.Date(2026, 11, 30, 9, 0, 0, 0, time.UTC))
	if p, _ := w.Path(id); !strings.Contains(filepath.ToSlash(p), "claims/2026-11/") {
		t.Errorf("minted under %s", p)
	}
	if err := f.retract(id, time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Errorf("a retraction two days slow: %v", err)
	}
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Errorf("check: %+v %v", d, err)
	}
}
