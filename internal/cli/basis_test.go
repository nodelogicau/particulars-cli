package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func warningsOf(r result) string {
	ws, _ := r.js["warnings"].([]any)
	var out []string
	for _, w := range ws {
		out = append(out, w.(string))
	}
	return strings.Join(out, "\n")
}

func TestClaimAssertBasis(t *testing.T) {
	dir := initWS(t)
	x := define(t, "Project X")
	a := assert(t, "Project X", "renewal A went wrong")
	b := assert(t, "Project X", "renewal B went wrong")
	c := assert(t, "Project X", "renewal C went wrong")

	inferred := func(extra ...string) result {
		return run(t, "", append([]string{"claim", "assert", "--subject", x, "--content", "Renewals go wrong when the TTL is long", "--json"}, extra...)...)
	}

	r := inferred("--evidential", "inferred", "--basis", a, "--basis", b, "--basis", c, "--confidence", "0.7")
	if r.code != 0 {
		t.Fatalf("assert with basis: %d %s", r.code, r.stderr)
	}
	if w := warningsOf(r); w != "" {
		t.Errorf("a well-founded rule carries no warnings: %s", w)
	}
	data, _ := os.ReadFile(filepath.Join(dir, r.js["path"].(string)))
	want := "evidential: inferred\nbasis:\n  - " + a + "\n  - " + b + "\n  - " + c + "\nconfidence: 0.7\n"
	if !strings.Contains(string(data), want) {
		t.Errorf("basis must be written between evidential and confidence, in order:\n%s", data)
	}
	if r := run(t, "", "index", "--check", "--json"); r.code != 0 {
		t.Errorf("index must be current after assert: %d %s", r.code, r.stderr)
	}

	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{"observation", []string{"--evidential", "observed", "--basis", a}, 2},
		{"position", []string{"--evidential", "held", "--basis", a}, 2},
		{"unknown id", []string{"--evidential", "inferred", "--basis", "clm_019196a5-8b4c-7def-8abc-0123456789ff"}, 3},
		{"particular", []string{"--evidential", "inferred", "--basis", x}, 2},
		{"label", []string{"--evidential", "inferred", "--basis", "Project X"}, 2},
		{"duplicate", []string{"--evidential", "inferred", "--basis", a, "--basis", a}, 2},
	} {
		before, _ := os.ReadDir(filepath.Join(dir, "claims"))
		if r := inferred(tc.args...); r.code != tc.code {
			t.Errorf("%s: exit %d, want %d (%s)", tc.name, r.code, tc.code, r.stderr)
		}
		after, _ := os.ReadDir(filepath.Join(dir, "claims"))
		if len(after) != len(before) {
			t.Errorf("%s: a refused basis must write nothing", tc.name)
		}
	}

	// Inferred without a basis: written, with a warning.
	r = inferred("--evidential", "inferred")
	if r.code != 0 || !strings.Contains(warningsOf(r), "inferred_without_basis") {
		t.Errorf("inferred without basis: %d %v", r.code, r.js["warnings"])
	}

	// A retracted member: written, stale from the start.
	if r := run(t, "", "retract", c, "--reason", "misremembered", "--json"); r.code != 0 {
		t.Fatalf("retract: %s", r.stderr)
	}
	r = inferred("--evidential", "inferred", "--basis", c)
	if r.code != 0 || !strings.Contains(warningsOf(r), "stale_claim") || !strings.Contains(warningsOf(r), c) {
		t.Errorf("retracted basis member: %d %v", r.code, r.js["warnings"])
	}

	// Wider than its basis: written, warned.
	r = inferred("--evidential", "inferred", "--basis", a, "--scope", "organisation")
	if r.code != 0 || !strings.Contains(warningsOf(r), "basis member(s) "+a) {
		t.Errorf("scope wider than basis: %d %v", r.code, r.js["warnings"])
	}
}

func TestLineageTextLabelsBasis(t *testing.T) {
	initWS(t)
	x := define(t, "Project X")
	a := assert(t, "Project X", "instance")
	r := run(t, "", "claim", "assert", "--subject", x, "--content", "rule", "--evidential", "inferred", "--basis", a, "--json")
	if r.code != 0 {
		t.Fatal(r.stderr)
	}
	rule := r.js["claim"].(map[string]any)["id"].(string)
	out := run(t, "", "lineage", rule)
	if !strings.Contains(out.stdout, a+" (basis)") {
		t.Errorf("text tree should label basis children:\n%s", out.stdout)
	}
	js := run(t, "", "lineage", rule, "--json")
	if b, _ := js.js["basis"].([]any); len(b) != 1 {
		t.Errorf("JSON tree should list the basis: %v", js.js)
	}
	if rec := run(t, "", "recall", "Project X"); !strings.Contains(rec.stdout, "basis: "+a) {
		t.Errorf("recall text should show the basis:\n%s", rec.stdout)
	}
}
