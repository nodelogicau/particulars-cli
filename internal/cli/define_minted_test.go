package cli

import (
	"os"
	"strings"
	"testing"
)

func countParticulars(t *testing.T) int {
	t.Helper()
	r := run(t, "", "register", "--json")
	if r.code != 0 {
		t.Fatalf("register: %+v", r)
	}
	ps, _ := r.js["particulars"].([]any)
	return len(ps)
}

func TestDefineMintedCollision(t *testing.T) {
	dir := initWS(t)
	r := run(t, "", "particular", "define", "--label", "C", "--json")
	if r.code != 0 || r.js["created"] != true {
		t.Fatalf("define C: %+v", r)
	}
	c := r.js["particular"].(map[string]any)
	cPath := objPath(t, dir, c["id"].(string))
	before, _ := os.ReadFile(cPath)

	for _, args := range [][]string{{"--label", "C#"}, {"--label", "C++"}, {"--label", "C#", "--alias", "C"}, {"--label", "C 東京"}} {
		r := run(t, "", append(append([]string{"particular", "define"}, args...), "--json")...)
		if r.code != 1 || r.errCode(t) != "conflict" {
			t.Fatalf("define %v: want exit 1 conflict, got %+v", args, r)
		}
		msg := r.errJS["error"].(map[string]any)["message"].(string)
		for _, want := range []string{c["id"].(string), c["uri"].(string), `"C"`, "--uri"} {
			if !strings.Contains(msg, want) {
				t.Errorf("define %v: message %q lacks %q", args, msg, want)
			}
		}
	}
	if after, _ := os.ReadFile(cPath); string(after) != string(before) || countParticulars(t) != 1 {
		t.Errorf("a refusal wrote something:\n%s", after)
	}

	// The same thing, differently cased, still resolves; label kept.
	r = run(t, "", "particular", "define", "--label", "c", "--json")
	if r.code != 0 || r.js["created"] != false || r.js["particular"].(map[string]any)["label"] != "C" {
		t.Errorf("define c: %+v", r)
	}
	// An explicit URI beside the minted one creates a second particular.
	r = run(t, "", "particular", "define", "--label", "C#", "--uri", "https://www.wikidata.org/entity/Q2370", "--json")
	if r.code != 0 || r.js["created"] != true {
		t.Fatalf("explicit C#: %+v", r)
	}
	if after, _ := os.ReadFile(cPath); string(after) != string(before) {
		t.Errorf("C changed by an explicit-URI define:\n%s", after)
	}
	// Renaming goes through the explicit URI.
	r = run(t, "", "particular", "define", "--label", "C (language)", "--uri", c["uri"].(string), "--json")
	p := r.js["particular"].(map[string]any)
	if r.code != 0 || r.js["created"] != false || p["label"] != "C (language)" || p["aliases"].([]any)[0] != "C" {
		t.Errorf("rename: %+v", r)
	}
}

func TestDefineEmptySlug(t *testing.T) {
	initWS(t)
	for _, label := range []string{"#", "東京", "Ωμέγα"} {
		r := run(t, "", "particular", "define", "--label", label, "--json")
		if r.code != 2 || r.errCode(t) != "usage" {
			t.Fatalf("define %q: want exit 2 usage, got %+v", label, r)
		}
		if msg := r.errJS["error"].(map[string]any)["message"].(string); !strings.Contains(msg, "explicit --uri is required") {
			t.Errorf("define %q: message %q", label, msg)
		}
	}
	if n := countParticulars(t); n != 0 {
		t.Errorf("empty-slug refusals wrote %d particulars", n)
	}
	r := run(t, "", "particular", "define", "--label", "東京", "--uri", "https://www.wikidata.org/entity/Q1490", "--json")
	if r.code != 0 || r.js["created"] != true || r.js["particular"].(map[string]any)["uri"] != "https://www.wikidata.org/entity/Q1490" {
		t.Errorf("explicit uri for 東京: %+v", r)
	}
	r = run(t, "", "particular", "define", "--label", "C 東京", "--json")
	if uri, _ := r.js["particular"].(map[string]any)["uri"].(string); r.code != 0 || !strings.HasSuffix(uri, ":c") {
		t.Errorf("C 東京 should mint …:c, got %+v", r)
	}
}
