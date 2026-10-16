package query

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/store"
)

// genTestClaims is deliberately not a benchmark size: plain `go test` checks
// the generator without generating any workspace a benchmark would time.
const genTestClaims = 120

func readTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[rel] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestGenerateWorkspaceDeterministic(t *testing.T) {
	a, b := filepath.Join(t.TempDir(), "a"), filepath.Join(t.TempDir(), "b")
	sa, err := generateWorkspace(a, genTestClaims, genSeed)
	if err != nil {
		t.Fatal(err)
	}
	sb, err := generateWorkspace(b, genTestClaims, genSeed)
	if err != nil {
		t.Fatal(err)
	}
	if sa != sb {
		t.Fatalf("stats differ: %+v vs %+v", sa, sb)
	}
	ta, tb := readTree(t, a), readTree(t, b)
	if len(ta) != len(tb) || len(ta) != sa.Files {
		t.Fatalf("file counts: %d, %d, stats say %d", len(ta), len(tb), sa.Files)
	}
	for p, data := range ta {
		if !bytes.Equal(data, tb[p]) {
			t.Fatalf("%s differs between generations", p)
		}
	}
}

func TestGenerateWorkspaceShape(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kb")
	st, err := generateWorkspace(dir, genTestClaims, genSeed)
	if err != nil {
		t.Fatal(err)
	}
	if st.Claims != genTestClaims || st.Particulars != genTestClaims/genShape.claimsPerParticular {
		t.Fatalf("counts: %+v", st)
	}
	if st.Syntheses == 0 || st.Merges == 0 || st.Topics == 0 || st.HotParticular == "" || st.HotTopic == "" {
		t.Fatalf("missing structure: %+v", st)
	}
	assertValid(t, dir)
}

func assertValid(tb testing.TB, dir string) {
	tb.Helper()
	w, err := store.Open(dir)
	if err != nil {
		tb.Fatal(err)
	}
	fs, err := Validate(w)
	if err != nil {
		tb.Fatal(err)
	}
	for _, f := range fs {
		if f.Severity == SeverityError {
			tb.Errorf("%s %s: %s", f.Path, f.Code, f.Message)
		}
	}
	diff, err := w.CheckIndex()
	if err != nil {
		tb.Fatal(err)
	}
	if !diff.Clean() {
		tb.Fatalf("index drift: %+v", diff)
	}
}
