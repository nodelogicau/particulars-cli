package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nodelogicau/particulars-cli/internal/dkf"
	"github.com/nodelogicau/particulars-cli/internal/query"
)

// populate fills the current workspace with one of each object type,
// including a retraction, and returns a claim id and the subject label.
func populate(t *testing.T) (claim string) {
	t.Helper()
	define(t, "Project X", "--alias", "px")
	define(t, "ProjectX legacy")
	define(t, "Other")
	c1 := assert(t, "Project X", "Uses Postgres", "--topic", "db", "--timestamp", "2023-05-24T00:36:00Z")
	c2 := assert(t, "Project X", "Uses MySQL", "--topic", "db")
	synth(t, "Project X", []string{c1 + ":thesis", c2 + ":antithesis"})
	if r := run(t, "", "retract", c2, "--reason", "wrong", "--json"); r.code != 0 {
		t.Fatalf("retract: %+v", r)
	}
	if r := run(t, "", "particular", "merge", "Project X", "ProjectX legacy", "--reason", "same", "--json"); r.code != 0 {
		t.Fatalf("merge: %+v", r)
	}
	if r := run(t, "", "publish", c1, "--scope", "organisation", "--json"); r.code != 0 {
		t.Fatalf("publish: %+v", r)
	}
	return c1
}

// shard rewrites a flat workspace as dkf/0.2: each object file moves to the
// month its id derives (or legacy/), byte-identical, and dkf.yaml's format
// changes. index.yaml is left as it was; this release does not read it.
func shard(t *testing.T, dir string) {
	t.Helper()
	for _, td := range []string{"particulars", "claims", "syntheses", "merges", "publishes"} {
		entries, _ := os.ReadDir(filepath.Join(dir, td))
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			id := strings.TrimSuffix(e.Name(), ".yaml")
			month, ok := dkf.MonthOf(id)
			if !ok {
				month = "legacy"
			}
			to := filepath.Join(dir, td, month, e.Name())
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(filepath.Join(dir, td, e.Name()), to); err != nil {
				t.Fatal(err)
			}
		}
	}
	cfg := filepath.Join(dir, "dkf.yaml")
	b, _ := os.ReadFile(cfg)
	if err := os.WriteFile(cfg, []byte(strings.Replace(string(b), "format: dkf/0.1", "format: dkf/0.2", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// treeHash fingerprints every file under dir, so a refusal can be shown to
// have left the workspace byte-identical.
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
		h.Write([]byte(rel + "\x00"))
		h.Write(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// withoutPaths drops file paths from a JSON result, the one thing that
// legitimately differs between the two layouts.
func withoutPaths(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, x := range t {
			if k != "path" && k != "root" {
				out[k] = withoutPaths(x)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = withoutPaths(x)
		}
		return out
	}
	return v
}

// A dkf/0.2 workspace reads exactly as its dkf/0.1 equivalent.
func TestShardedReadsLikeFlat(t *testing.T) {
	dir := initWS(t)
	claim := populate(t)
	verbs := [][]string{
		{"recall", "Project X", "--include-retracted"},
		{"recall", "--query", "postgres"},
		{"lineage", claim},
		{"register"},
		{"topics"},
		{"conflicts"},
		{"unresolved", "--include-none"},
	}
	flat := map[string]any{}
	for _, v := range verbs {
		r := run(t, "", append(v, "--json")...)
		if r.code != 0 {
			t.Fatalf("flat %v: %d %s", v, r.code, r.stderr)
		}
		flat[strings.Join(v, " ")] = withoutPaths(r.js)
	}

	shard(t, dir)
	if _, err := os.Stat(filepath.Join(dir, "claims", time.Now().UTC().Format("2006-01"))); err != nil {
		t.Fatalf("fixture not sharded: %v", err)
	}
	for _, v := range verbs {
		r := run(t, "", append(v, "--json")...)
		if r.code != 0 {
			t.Fatalf("sharded %v: %d %s", v, r.code, r.stderr)
		}
		if got := withoutPaths(r.js); !reflect.DeepEqual(got, flat[strings.Join(v, " ")]) {
			t.Errorf("%v differs:\nsharded %v\nflat    %v", v, got, flat[strings.Join(v, " ")])
		}
	}

	// validate passes; the one finding is that the index went unchecked.
	r := run(t, "", "validate", "--json")
	if r.code != 0 || r.js["errors"].(float64) != 0 {
		t.Fatalf("validate on sharded: %d %s", r.code, r.stdout)
	}
	if !strings.Contains(r.stdout, query.CodeIndexUnchecked) {
		t.Errorf("validate should warn %s: %s", query.CodeIndexUnchecked, r.stdout)
	}
}

// Each way a file can be out of place is reported once and never read.
func TestShardedMisplacedFiles(t *testing.T) {
	dir := initWS(t)
	define(t, "Project X")
	claim := assert(t, "Project X", "Uses Postgres")
	shard(t, dir)
	month, _ := dkf.MonthOf(claim)
	src, _ := os.ReadFile(filepath.Join(dir, "claims", month, claim+".yaml"))
	other := dkf.NewID(dkf.TypeClaim)
	write := func(rel string, body []byte) {
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	retag := func(id string) []byte { return []byte(strings.ReplaceAll(string(src), claim, id)) }
	cases := map[string]string{ // misplaced file → derived location named in the finding
		"claims/" + other + ".yaml":                    "claims/" + month + "/" + other + ".yaml",
		"claims/2026-08/clm_07m3zp9s2q1r4t8v.yaml":     "claims/legacy/clm_07m3zp9s2q1r4t8v.yaml",
		"claims/legacy/" + other + ".yaml":             "claims/" + month + "/" + other + ".yaml",
		"claims/drafts/" + other + ".yaml":             "claims/" + month + "/" + other + ".yaml",
		"particulars/" + month + "/" + other + ".yaml": "claims/" + month + "/" + other + ".yaml",
	}
	for rel := range cases {
		id := strings.TrimSuffix(filepath.Base(rel), ".yaml")
		write(rel, retag(id))
	}
	write("claims/"+month+"/.hidden.yaml", []byte("not: read"))
	write("claims/"+month+"/README.md", []byte("not read"))

	r := run(t, "", "validate", "--json")
	if r.code != 4 {
		t.Fatalf("validate should fail: %d %s", r.code, r.stdout)
	}
	seen := map[string]int{}
	for _, f := range r.js["findings"].([]any) {
		f := f.(map[string]any)
		if f["code"] == query.CodeMisplacedFile {
			seen[f["path"].(string)]++
			if want := cases[f["path"].(string)]; want != "" && !strings.Contains(f["message"].(string), want) {
				t.Errorf("%s: %q should name %s", f["path"], f["message"], want)
			}
		}
	}
	for rel := range cases {
		if seen[rel] != 1 {
			t.Errorf("%s reported %d times as misplaced, want 1", rel, seen[rel])
		}
	}
	if len(seen) != len(cases) {
		t.Errorf("misplaced findings %v, want exactly %d", seen, len(cases))
	}

	// None of them is read as an object: like any unreadable file, a
	// misplaced one stops the read verbs, naming it, until it is moved.
	r = run(t, "", "recall", "Project X")
	if r.code == 0 || !strings.Contains(r.stderr, "claims/drafts/"+other+".yaml") || !strings.Contains(r.stderr, "particulars validate") {
		t.Errorf("recall over misplaced files: %d %s", r.code, r.stderr)
	}
}

// A dkf/0.2 workspace refuses every write and is left byte-identical.
func TestShardedIsReadOnly(t *testing.T) {
	dir := initWS(t)
	claim := populate(t)
	shard(t, dir)
	before := treeHash(t, dir)
	for _, args := range [][]string{
		{"particular", "define", "--label", "New"},
		{"particular", "define", "--label", "Project X", "--alias", "another"},
		{"claim", "assert", "--evidential", "observed", "--subject", "Project X", "--content", "x"},
		{"synthesis", "create", "--subject", "Project X", "--content", "s", "--unresolved", "none", "--input", claim + ":thesis"},
		{"retract", claim, "--reason", "r"},
		{"particular", "merge", "Project X", "Other", "--reason", "same"},
		{"publish", claim, "--scope", "public"},
		{"index"},
		{"index", "--check"},
	} {
		r := run(t, "", append(args, "--json")...)
		if r.code != 1 || r.errCode(t) != "read_only_format" {
			t.Errorf("%v: exit %d, code %q; want 1 read_only_format (%s)", args, r.code, r.errCode(t), r.stderr)
		}
		if after := treeHash(t, dir); after != before {
			t.Fatalf("%v changed the workspace", args)
		}
	}
	// export is a read verb: it writes derived output outside the workspace.
	out := filepath.Join(t.TempDir(), "graph.mmd")
	if r := run(t, "", "export", "--format", "mermaid", "--out", out); r.code != 0 {
		t.Errorf("export on sharded: %d %s", r.code, r.stderr)
	}
}

// An unknown format is refused by name, and validate reports it as such.
func TestUnsupportedFormat(t *testing.T) {
	dir := initWS(t)
	cfg := filepath.Join(dir, "dkf.yaml")
	b, _ := os.ReadFile(cfg)
	if err := os.WriteFile(cfg, []byte(strings.Replace(string(b), "format: dkf/0.1", "format: dkf/0.3", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	r := run(t, "", "recall", "--query", "x")
	if r.code != 1 || !strings.Contains(r.stderr, `"dkf/0.3"`) || !strings.Contains(r.stderr, "dkf/0.1, dkf/0.2") || !strings.Contains(r.stderr, "dkf.yaml") {
		t.Errorf("recall on dkf/0.3: %d %s", r.code, r.stderr)
	}
	r = run(t, "", "validate", "--json")
	if r.code != 4 || !strings.Contains(r.stdout, query.CodeUnsupportedFormat) {
		t.Errorf("validate on dkf/0.3: %d %s", r.code, r.stdout)
	}
}

// Time the validator cannot have seen warns; an id two months ahead fails.
// Moving the validator's clock back stands in for an id minted ahead of it.
func TestTimeChecks(t *testing.T) {
	for _, sharded := range []bool{false, true} {
		dir := initWS(t)
		define(t, "Project X")
		c := assert(t, "Project X", "Uses Postgres", "--timestamp", "2023-05-24T00:36:00Z")
		if r := run(t, "", "retract", c, "--reason", "wrong"); r.code != 0 {
			t.Fatalf("retract: %+v", r)
		}
		if sharded {
			shard(t, dir)
		}
		now := time.Now().UTC()
		y, m, _ := now.Date()
		for _, tc := range []struct {
			name     string
			clock    time.Time
			code     int
			warn     bool
			farError bool
		}{
			{"real clock; backdated timestamp is silent", now, 0, false, false},
			{"two days slow", now.AddDate(0, 0, -2), 0, true, false},
			{"a month slow", time.Date(y, m-1, 1, 0, 0, 0, 0, time.UTC), 0, true, false},
			{"two months slow", time.Date(y, m-2, 1, 0, 0, 0, 0, time.UTC), 4, true, true}, // the retraction's timestamp still warns
		} {
			clock := tc.clock
			query.Now = func() time.Time { return clock }
			r := run(t, "", "validate", "--json")
			query.Now = time.Now
			if r.code != tc.code {
				t.Errorf("sharded=%v %s: exit %d, want %d: %s", sharded, tc.name, r.code, tc.code, r.stdout)
			}
			if got := strings.Contains(r.stdout, query.CodeFutureTime); got != tc.warn {
				t.Errorf("sharded=%v %s: future_time %v, want %v: %s", sharded, tc.name, got, tc.warn, r.stdout)
			}
			if got := strings.Contains(r.stdout, query.CodeFarFutureID); got != tc.farError {
				t.Errorf("sharded=%v %s: far_future_id %v, want %v: %s", sharded, tc.name, got, tc.farError, r.stdout)
			}
		}
	}
}
