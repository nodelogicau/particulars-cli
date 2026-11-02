package query

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

// v7At is a UUIDv7 id of type t minted at the given instant; n keeps ids
// minted at one instant distinct.
func v7At(t dkf.Type, at time.Time, n int) string {
	ms := at.UnixMilli()
	return fmt.Sprintf("%s_%08x-%04x-7000-8000-%012x", dkf.Prefix(t), ms>>16, ms&0xffff, n)
}

// validate reports a file in a sealed month that its segment lacks as a late
// merge, and a retraction dated into a sealed month as sealed_retraction —
// the one the drift check, by design, does not see.
func TestShardedSealedChecks(t *testing.T) {
	cfg := store.NewConfig()
	cfg.Format = dkf.Format02
	cfg.Defaults.Source.Author = "ben"
	w, err := store.Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	at := func(m time.Month) time.Time { return time.Date(2026, m, 1, 0, 0, 0, 0, time.UTC) }
	create := func(obj dkf.Object) {
		t.Helper()
		if err := w.Create(obj); err != nil {
			t.Fatal(err)
		}
		if err := w.UpsertIndex(obj); err != nil {
			t.Fatal(err)
		}
	}
	claim := func(id, subject string, ts time.Time) *dkf.Claim {
		return &dkf.Claim{ID: id, Subject: subject, Content: "c " + id, Evidential: dkf.EvidentialObserved,
			Source: dkf.Source{Author: "ben"}, Context: dkf.Context{Scope: dkf.ScopePersonal}, Timestamp: ts}
	}
	p := &dkf.Particular{ID: v7At(dkf.TypeParticular, at(7), 1), URI: dkf.MintURI("", cfg.Workspace.ID, "subject"), Label: "Subject"}
	create(p)
	july := claim(v7At(dkf.TypeClaim, at(7), 2), p.ID, at(7))
	create(july)
	create(claim(v7At(dkf.TypeClaim, at(8), 3), p.ID, at(8)))
	create(claim(v7At(dkf.TypeClaim, at(9), 4), p.ID, at(9))) // seals July
	create(claim(v7At(dkf.TypeClaim, at(10), 5), p.ID, at(10)))

	codes := func() map[string][]Finding {
		t.Helper()
		fs, err := Validate(w)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string][]Finding{}
		for _, f := range fs {
			out[f.Code] = append(out[f.Code], f)
		}
		return out
	}
	if got := codes(); len(got[CodeIndexStale])+len(got[CodeLateMerge])+len(got[CodeSealedRetraction]) != 0 {
		t.Fatalf("a CLI-written sharded index has index findings: %v", got)
	}

	// A late merge: a July file arrives behind the index.
	late := claim(v7At(dkf.TypeClaim, at(7), 9), p.ID, at(7))
	data, _ := dkf.Encode(late)
	latePath := filepath.Join(w.Root, "claims", "2026-07", late.ID+".yaml")
	if err := os.WriteFile(latePath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got := codes()
	if lm := got[CodeLateMerge]; len(lm) != 1 || lm[0].Path != "claims/2026-07/"+late.ID+".yaml" || !strings.Contains(lm[0].Message, "index/2026-07.yaml") {
		t.Errorf("late_merge: %+v", lm)
	}
	if len(got[CodeIndexStale]) != 0 {
		t.Errorf("a late merge alone is reported once, as itself: %+v", got[CodeIndexStale])
	}
	if err := os.Remove(latePath); err != nil {
		t.Fatal(err)
	}

	// A retraction dated into sealed July, written by hand.
	july.SetRetracted(&dkf.Retracted{Timestamp: at(7).Add(time.Hour), Reason: "wrong", Source: dkf.Source{Author: "ben"}})
	data, _ = dkf.Encode(july)
	julyPath, _ := w.Path(july.ID)
	if err := os.WriteFile(julyPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if d, err := w.CheckIndex(); err != nil || !d.Clean() {
		t.Errorf("the drift check does not see a sealed-month retraction: %+v %v", d, err)
	}
	got = codes()
	if sr := got[CodeSealedRetraction]; len(sr) != 1 || !strings.Contains(sr[0].Message, july.ID) || !strings.Contains(sr[0].Message, "index/2026-07.yaml") {
		t.Errorf("sealed_retraction: %+v", sr)
	}
}
