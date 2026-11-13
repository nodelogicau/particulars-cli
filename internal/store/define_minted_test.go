package store

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

func particularCount(t *testing.T, w *Workspace) int {
	t.Helper()
	g, err := w.Load()
	if err != nil {
		t.Fatal(err)
	}
	return len(g.Particulars)
}

func mustDefineMinted(t *testing.T, w *Workspace, uri, label string, aliases ...string) (*dkf.Particular, bool) {
	t.Helper()
	p, created, err := w.DefineMinted(uri, label, aliases)
	if err != nil {
		t.Fatalf("DefineMinted(%q, %q): %v", uri, label, err)
	}
	return p, created
}

func wantCollision(t *testing.T, err error, existing *dkf.Particular) {
	t.Helper()
	var mc *MintedCollisionError
	if !errors.Is(err, ErrMintedCollision) || !errors.As(err, &mc) {
		t.Fatalf("expected a minted collision, got %v", err)
	}
	if mc.Existing.ID != existing.ID {
		t.Errorf("refusal names %s, want %s", mc.Existing.ID, existing.ID)
	}
}

func TestDefineMintedSameSlugDifferentThing(t *testing.T) {
	w := newWS(t)
	c, created := mustDefineMinted(t, w, "urn:dkf:W:c", "C")
	if !created {
		t.Fatal("first define should create")
	}
	path, _ := w.Path(c.ID)
	before, _ := os.ReadFile(path)
	for _, tc := range []struct {
		label   string
		aliases []string
	}{{"C#", nil}, {"C++", nil}, {"C#", []string{"C"}}} {
		_, _, err := w.DefineMinted("urn:dkf:W:c", tc.label, tc.aliases)
		wantCollision(t, err, c)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) || particularCount(t, w) != 1 {
		t.Errorf("a refusal wrote something:\n%s", after)
	}
}

func TestDefineMintedDiacritics(t *testing.T) {
	w := newWS(t)
	cafe, _ := mustDefineMinted(t, w, "urn:dkf:W:cafe", "Café")
	_, _, err := w.DefineMinted("urn:dkf:W:cafe", "Cafe", nil)
	wantCollision(t, err, cafe)
}

func TestDefineMintedCaseOnlyKeepsLabel(t *testing.T) {
	w := newWS(t)
	p1, _ := mustDefineMinted(t, w, "urn:dkf:W:project-x", "Project X", "ProjectX")
	path, _ := w.Path(p1.ID)
	before, _ := os.ReadFile(path)

	// A repeat define with nothing new writes nothing.
	p2, created := mustDefineMinted(t, w, "urn:dkf:W:project-x", "project x", "projectx")
	if created || p2.ID != p1.ID || p2.Label != "Project X" {
		t.Fatalf("got %+v created=%v", p2, created)
	}
	if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
		t.Errorf("repeat define rewrote the file:\n%s", after)
	}

	// New aliases are added, never removed, and the label stays.
	p3, _ := mustDefineMinted(t, w, "urn:dkf:W:project-x", "project x", "project_x")
	if p3.Label != "Project X" || strings.Join(p3.Aliases, ",") != "ProjectX,project_x" {
		t.Errorf("got label %q aliases %v", p3.Label, p3.Aliases)
	}
	g, _ := w.Load()
	if got := g.Particular(p1.ID); got.Label != "Project X" || strings.Join(got.Aliases, ",") != "ProjectX,project_x" {
		t.Errorf("on disk: %+v", got)
	}
}

func TestDefineMintedMatchesAlias(t *testing.T) {
	w := newWS(t)
	p, _ := mustDefineMinted(t, w, "urn:dkf:W:c-lang", "C-lang")
	if _, _, err := w.UpsertParticular("urn:dkf:W:c-lang", "C", nil); err != nil {
		t.Fatal(err)
	}
	got, created := mustDefineMinted(t, w, "urn:dkf:W:c-lang", "c-lang")
	if created || got.ID != p.ID || got.Label != "C" {
		t.Errorf("got %+v created=%v", got, created)
	}
}

func TestDefineMintedThroughMerge(t *testing.T) {
	w := newWS(t)
	ada, _, err := w.UpsertParticular("https://orcid.org/0000-0001", "Ada Lovelace", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.CreateMerge("urn:dkf:W:ada-lovelace", ada.URI, "", dkf.Source{Author: "ben"}, ts); err != nil {
		t.Fatal(err)
	}
	got, created := mustDefineMinted(t, w, "urn:dkf:W:ada-lovelace", "Ada Lovelace")
	if created || got.ID != ada.ID || particularCount(t, w) != 1 {
		t.Errorf("got %+v created=%v count=%d", got, created, particularCount(t, w))
	}

	if _, err := w.CreateMerge("urn:dkf:W:ada", ada.URI, "", dkf.Source{Author: "ben"}, ts); err != nil {
		t.Fatal(err)
	}
	_, _, err = w.DefineMinted("urn:dkf:W:ada", "ADA", nil)
	wantCollision(t, err, ada)
}

func TestDefineMintedPrefersCarrier(t *testing.T) {
	w := newWS(t)
	other, _, _ := w.UpsertParticular("https://example.com/x", "X", nil)
	carrier, _ := mustDefineMinted(t, w, "urn:dkf:W:x", "X")
	if _, err := w.CreateMerge("urn:dkf:W:x", other.URI, "", dkf.Source{Author: "ben"}, ts); err != nil {
		t.Fatal(err)
	}
	got, _ := mustDefineMinted(t, w, "urn:dkf:W:x", "x")
	if got.ID != carrier.ID {
		t.Errorf("resolved to %s, want the carrier %s", got.ID, carrier.ID)
	}
	_, _, err := w.DefineMinted("urn:dkf:W:x", "X!", nil)
	wantCollision(t, err, carrier)
}
