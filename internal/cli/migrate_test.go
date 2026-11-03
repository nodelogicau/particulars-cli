package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
)

// treeHash fingerprints every file under dir, so a refusal can be shown to
// have written nothing.
func treeHash(t *testing.T, dir string) string {
	t.Helper()
	h := sha256.New()
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		h.Write([]byte(filepath.ToSlash(rel) + "\x00"))
		h.Write(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// objectBytes reads every object file under the type directories, keyed by
// id, wherever the layout keeps it.
func objectBytes(t *testing.T, dir string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	for _, td := range []string{"particulars", "claims", "syntheses", "merges", "publishes"} {
		_ = filepath.WalkDir(filepath.Join(dir, td), func(p string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() && strings.HasSuffix(p, ".yaml") {
				b, _ := os.ReadFile(p)
				out[strings.TrimSuffix(filepath.Base(p), ".yaml")] = b
			}
			return nil
		})
	}
	return out
}

// findingCodes is validate's findings as a sorted list of codes, so a
// migration can be shown to change nothing validate says but paths.
func findingCodes(t *testing.T) []string {
	t.Helper()
	r := run(t, "", "validate", "--json")
	var codes []string
	for _, f := range r.js["findings"].([]any) {
		codes = append(codes, f.(map[string]any)["code"].(string))
	}
	sort.Strings(codes)
	return codes
}

func mustRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// flatFixture is a dkf/0.1 workspace with every object type, a retraction,
// a claim minted three months ago, a legacy id, a preserved index row of an
// unknown type, and a dkf.yaml carrying a comment and an unknown key.
func flatFixture(t *testing.T) (dir, old, legacy, unknown string) {
	t.Helper()
	dir = initWS(t, "--format", "dkf/0.1")
	claim := populate(t)
	src, _ := os.ReadFile(objPath(t, dir, claim))
	ms := time.Now().UTC().AddDate(0, -3, 0).UnixMilli()
	old = fmt.Sprintf("clm_%08x-%04x-7000-8000-00000000abcd", ms>>16, ms&0xffff)
	writeObj(t, dir, old, bytes.Replace(src, []byte(claim), []byte(old), 1))
	legacy = "clm_07m3zp9s2q1r4t8v"
	writeObj(t, dir, legacy, bytes.Replace(src, []byte(claim), []byte(legacy), 1))
	if r := run(t, "", "index"); r.code != 0 {
		t.Fatalf("index: %s", r.stderr)
	}
	unknown = "rdx_" + strings.TrimPrefix(claim, "clm_")
	idx := filepath.Join(dir, "index.yaml")
	b, _ := os.ReadFile(idx)
	b = append(b, []byte("  - id: "+unknown+"\n    type: redaction\n")...)
	if err := os.WriteFile(idx, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(t, "", "index"); r.code != 0 || !strings.Contains(mustRead(t, idx), unknown) {
		t.Fatalf("a flat rebuild should keep the unknown row in place: %s", r.stderr)
	}
	cfg := filepath.Join(dir, "dkf.yaml")
	c, _ := os.ReadFile(cfg)
	c = bytes.Replace(c, []byte("format: dkf/0.1\n"), []byte("format: dkf/0.1  # flat, for now\n"), 1)
	c = append([]byte("# the team's knowledge\n"), append(c, []byte("x-team-note: keep me\n")...)...)
	if err := os.WriteFile(cfg, c, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir, old, legacy, unknown
}

// Migration moves every file byte-identical to its derived path, edits only
// the format line, carries the unknown index row across, and changes nothing
// validate reports; a second run writes nothing.
func TestMigrate(t *testing.T) {
	dir, old, legacy, unknown := flatFixture(t)
	before := objectBytes(t, dir)
	codesBefore := findingCodes(t)
	cfgBefore, _ := os.ReadFile(filepath.Join(dir, "dkf.yaml"))

	// A dry run writes nothing and says what would happen.
	hash := treeHash(t, dir)
	r := run(t, "", "migrate", "--dry-run", "--json")
	if r.code != 0 || r.js["moved"].(float64) != float64(len(before)) || treeHash(t, dir) != hash {
		t.Fatalf("dry run: %d %s", r.code, r.stdout)
	}
	oldMonth, _ := dkf.MonthOf(old)
	if !strings.Contains(r.stdout, oldMonth) || !strings.Contains(r.stdout, legacy) {
		t.Errorf("dry run should name the sealed month %s and the legacy id: %s", oldMonth, r.stdout)
	}

	r = run(t, "", "migrate", "--json")
	if r.code != 0 || r.js["moved"].(float64) != float64(len(before)) {
		t.Fatalf("migrate: %d %s %s", r.code, r.stdout, r.stderr)
	}
	after := objectBytes(t, dir)
	for id, b := range before {
		if !bytes.Equal(after[id], b) {
			t.Errorf("%s changed in migration", id)
		}
		p := objPath(t, dir, id)
		if got, err := os.ReadFile(p); err != nil || !bytes.Equal(got, b) {
			t.Errorf("%s is not at its derived path %s", id, p)
		}
	}
	if !strings.Contains(filepath.ToSlash(objPath(t, dir, legacy)), "claims/legacy/") {
		t.Errorf("legacy id not under legacy/: %s", objPath(t, dir, legacy))
	}
	cfgAfter, _ := os.ReadFile(filepath.Join(dir, "dkf.yaml"))
	if want := bytes.Replace(cfgBefore, []byte("format: dkf/0.1  # flat, for now"), []byte("format: dkf/0.2  # flat, for now"), 1); !bytes.Equal(cfgAfter, want) {
		t.Errorf("dkf.yaml should differ only on the format line:\n%s", cfgAfter)
	}
	if r := run(t, "", "index", "--check"); r.code != 0 {
		t.Errorf("index after migration: %s %s", r.stdout, r.stderr)
	}
	if got := findingCodes(t); strings.Join(got, ",") != strings.Join(codesBefore, ",") {
		t.Errorf("validate changed:\nbefore %v\nafter  %v", codesBefore, got)
	}
	var indexText strings.Builder
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && (filepath.Base(p) == "index.yaml" || filepath.Base(filepath.Dir(p)) == "index") {
			b, _ := os.ReadFile(p)
			indexText.Write(b)
		}
		return nil
	})
	if !strings.Contains(indexText.String(), unknown) {
		t.Error("the unknown index row was not carried across")
	}
	if _, err := os.Stat(filepath.Join(dir, "index", oldMonth+".yaml")); err != nil {
		t.Errorf("the old month's segment was not written: %v", err)
	}

	hash = treeHash(t, dir)
	r = run(t, "", "migrate", "--json")
	if r.code != 0 || r.js["already_migrated"] != true || treeHash(t, dir) != hash {
		t.Errorf("second run: %d %s", r.code, r.stdout)
	}
}

// An interrupted migration — format rewritten, some files moved — finishes
// when migrate runs again; a conflicting target refuses instead.
func TestMigrateResumes(t *testing.T) {
	dir, _, _, _ := flatFixture(t)
	before := objectBytes(t, dir)
	cfg := filepath.Join(dir, "dkf.yaml")
	c, _ := os.ReadFile(cfg)
	if err := os.WriteFile(cfg, bytes.Replace(c, []byte("format: dkf/0.1"), []byte("format: dkf/0.2"), 1), 0o644); err != nil {
		t.Fatal(err)
	}
	moved := 0
	entries, _ := os.ReadDir(filepath.Join(dir, "claims"))
	for _, e := range entries {
		if moved == 2 || e.IsDir() {
			continue
		}
		p := objPath(t, dir, strings.TrimSuffix(e.Name(), ".yaml"))
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.Rename(filepath.Join(dir, "claims", e.Name()), p); err != nil {
			t.Fatal(err)
		}
		moved++
	}
	if r := run(t, "", "recall", "Project X"); r.code == 0 {
		t.Fatal("a half-migrated workspace must be loud, not silently partial")
	}

	// A copy at the target with other bytes refuses, naming it.
	victim := ""
	entries, _ = os.ReadDir(filepath.Join(dir, "claims"))
	for _, e := range entries {
		if !e.IsDir() {
			victim = strings.TrimSuffix(e.Name(), ".yaml")
			break
		}
	}
	target := objPath(t, dir, victim)
	_ = os.MkdirAll(filepath.Dir(target), 0o755)
	if err := os.WriteFile(target, []byte("something else\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash := treeHash(t, dir)
	r := run(t, "", "migrate", "--json")
	if r.code != 1 || r.errCode(t) != "migration_refused" || !strings.Contains(r.stderr, victim) || treeHash(t, dir) != hash {
		t.Errorf("conflicting target: %d %s", r.code, r.stderr)
	}
	// An identical copy is an earlier run's half-finished rename.
	if err := os.WriteFile(target, before[victim], 0o644); err != nil {
		t.Fatal(err)
	}
	if r := run(t, "", "migrate", "--json"); r.code != 0 {
		t.Fatalf("resume: %d %s", r.code, r.stderr)
	}
	if after := objectBytes(t, dir); len(after) != len(before) {
		t.Errorf("resume left %d objects, want %d", len(after), len(before))
	}
	if r := run(t, "", "index", "--check"); r.code != 0 {
		t.Errorf("index after resume: %s", r.stdout)
	}
	if r := run(t, "", "validate", "--json"); r.js["errors"].(float64) != 0 {
		t.Errorf("validate after resume: %s", r.stdout)
	}
}

// Each precondition refuses and writes nothing.
func TestMigrateRefuses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		break_ func(t *testing.T, dir string)
		names  string
	}{
		{"an unreadable file", func(t *testing.T, dir string) {
			_ = os.WriteFile(filepath.Join(dir, "claims", "clm_broken.yaml"), []byte("id: [unclosed\n"), 0o644)
		}, "clm_broken.yaml"},
		{"a foreign subdirectory", func(t *testing.T, dir string) {
			_ = os.MkdirAll(filepath.Join(dir, "claims", "drafts"), 0o755)
			_ = os.WriteFile(filepath.Join(dir, "claims", "drafts", "note.yaml"), []byte("a: b\n"), 0o644)
		}, "claims/drafts"},
		{"an unusual format line", func(t *testing.T, dir string) {
			cfg := filepath.Join(dir, "dkf.yaml")
			c, _ := os.ReadFile(cfg)
			_ = os.WriteFile(cfg, bytes.Replace(c, []byte("format: dkf/0.1"), []byte(`format: "dkf/0.1"`), 1), 0o644)
		}, "dkf.yaml"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := initWS(t, "--format", "dkf/0.1")
			define(t, "Project X")
			assert(t, "Project X", "A")
			tc.break_(t, dir)
			hash := treeHash(t, dir)
			r := run(t, "", "migrate", "--json")
			if r.code != 1 || r.errCode(t) != "migration_refused" || !strings.Contains(r.stderr, tc.names) {
				t.Errorf("exit %d: %s", r.code, r.stderr)
			}
			if treeHash(t, dir) != hash {
				t.Error("a refused migration wrote something")
			}
		})
	}
}

// A server running across the migration read dkf.yaml at startup, so the
// text result tells the user to restart it.
func TestMigrateSaysRestartTheServer(t *testing.T) {
	initWS(t, "--format", "dkf/0.1")
	define(t, "Project X")
	r := run(t, "", "migrate")
	if r.code != 0 || !strings.Contains(r.stdout, "Restart any running particulars MCP server") {
		t.Errorf("migrate output: %d %s", r.code, r.stdout)
	}
}
