package query

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/nodelogicau/particulars-cli/internal/store"
)

// Scale benchmarks: the read paths that degrade as a workspace grows, over
// generated workspaces that bracket the dkf/0.2 sharding trigger (~800
// claims). Run with `make bench-scale`; plain `go test` never generates them.

var benchSizes = []int{100, 800, 5000, 20000}

type benchWorkspace struct {
	dir   string
	stats genStats
}

var (
	benchMu    sync.Mutex
	benchRoot  string
	benchCache = map[int]*benchWorkspace{}
	binOnce    sync.Once
	binDir     string
	binPath    string
	binErr     error
)

func TestMain(m *testing.M) {
	code := m.Run()
	for _, dir := range []string{benchRoot, binDir} {
		if dir != "" {
			_ = os.RemoveAll(dir)
		}
	}
	os.Exit(code)
}

// benchWS generates (once per process) and validates the workspace of the
// given size. Callers reset the timer after it returns.
func benchWS(b *testing.B, claims int) *benchWorkspace {
	b.Helper()
	benchMu.Lock()
	defer benchMu.Unlock()
	if ws := benchCache[claims]; ws != nil {
		return ws
	}
	if benchRoot == "" {
		dir, err := os.MkdirTemp("", "particulars-bench-")
		if err != nil {
			b.Fatal(err)
		}
		benchRoot = dir
	}
	dir := filepath.Join(benchRoot, fmt.Sprintf("%s-%d", genVersion, claims))
	st, err := generateWorkspace(dir, claims, genSeed)
	if err != nil {
		b.Fatal(err)
	}
	assertValid(b, dir)
	ws := &benchWorkspace{dir: dir, stats: st}
	benchCache[claims] = ws
	return ws
}

func openBench(b *testing.B, ws *benchWorkspace) *store.Workspace {
	b.Helper()
	w, err := store.Open(ws.dir)
	if err != nil {
		b.Fatal(err)
	}
	return w
}

func loadBench(b *testing.B, ws *benchWorkspace) *store.Graph {
	b.Helper()
	g, err := openBench(b, ws).Load()
	if err != nil {
		b.Fatal(err)
	}
	return g
}

// eachSize runs fn as a sub-benchmark per size, timing only fn's loop.
func eachSize(b *testing.B, fn func(b *testing.B, ws *benchWorkspace)) {
	for _, n := range benchSizes {
		b.Run(fmt.Sprintf("claims=%d", n), func(b *testing.B) {
			ws := benchWS(b, n)
			b.ReportAllocs()
			b.ResetTimer()
			fn(b, ws)
			// After the loop: resetting the timer discards reported metrics.
			b.ReportMetric(float64(ws.stats.Files), "files")
		})
	}
}

func BenchmarkLoad(b *testing.B) {
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		w := openBench(b, ws)
		for b.Loop() {
			if _, err := w.Load(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRebuildIndex(b *testing.B) {
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		w := openBench(b, ws)
		for b.Loop() {
			if _, err := w.RebuildIndex(); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRecall(b *testing.B) {
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		g := loadBench(b, ws)
		b.ResetTimer()
		for b.Loop() {
			if len(Recall(g, RecallOptions{Subject: ws.stats.HotParticular})) == 0 {
				b.Fatal("empty recall")
			}
		}
	})
}

func BenchmarkRecallTopic(b *testing.B) {
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		g := loadBench(b, ws)
		b.ResetTimer()
		for b.Loop() {
			if len(Recall(g, RecallOptions{Topics: []string{ws.stats.HotTopic}})) == 0 {
				b.Fatal("empty recall")
			}
		}
	})
}

func BenchmarkConflicts(b *testing.B) {
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		g := loadBench(b, ws)
		b.ResetTimer()
		for b.Loop() {
			Conflicts(g, "")
		}
	})
}

func BenchmarkValidate(b *testing.B) {
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		w := openBench(b, ws)
		for b.Loop() {
			if _, err := Validate(w); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkExecRecall runs the built binary, so process start and the index
// read are counted as a user waits for them.
func BenchmarkExecRecall(b *testing.B) {
	binOnce.Do(func() {
		dir, err := os.MkdirTemp("", "particulars-bench-bin-")
		if err != nil {
			binErr = err
			return
		}
		binDir, binPath = dir, filepath.Join(dir, "particulars")
		out, err := exec.Command("go", "build", "-o", binPath, "github.com/nodelogicau/particulars-cli/cmd/particulars").CombinedOutput()
		if err != nil {
			binErr = fmt.Errorf("go build: %v\n%s", err, out)
		}
	})
	if binErr != nil {
		b.Fatal(binErr)
	}
	eachSize(b, func(b *testing.B, ws *benchWorkspace) {
		for b.Loop() {
			cmd := exec.Command(binPath, "--workspace", ws.dir, "recall", ws.stats.HotParticular, "--json")
			if out, err := cmd.CombinedOutput(); err != nil {
				b.Fatalf("%v\n%s", err, out)
			}
		}
	})
}
