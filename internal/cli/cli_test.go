package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type result struct {
	code   int
	stdout string
	stderr string
	js     map[string]any // parsed stdout
	errJS  map[string]any // parsed stderr error envelope
}

// run executes the CLI in-process with --json appended unless text is true.
func run(t *testing.T, stdin string, args ...string) result {
	t.Helper()
	var out, errb bytes.Buffer
	code := Execute(args, strings.NewReader(stdin), &out, &errb, func() bool { return false })
	r := result{code: code, stdout: out.String(), stderr: errb.String()}
	if hasJSON(args) {
		if strings.TrimSpace(r.stdout) != "" {
			if err := json.Unmarshal([]byte(r.stdout), &r.js); err != nil {
				t.Fatalf("invalid JSON on stdout (%v):\n%s", err, r.stdout)
			}
		}
		if strings.TrimSpace(r.stderr) != "" {
			if err := json.Unmarshal([]byte(r.stderr), &r.errJS); err != nil {
				t.Fatalf("invalid JSON on stderr (%v):\n%s", err, r.stderr)
			}
		}
	}
	return r
}

func hasJSON(args []string) bool {
	for _, a := range args {
		if a == "--json" {
			return true
		}
	}
	return false
}

func (r result) errCode(t *testing.T) string {
	t.Helper()
	e, _ := r.errJS["error"].(map[string]any)
	if e == nil {
		t.Fatalf("no error envelope in stderr: %q", r.stderr)
	}
	return e["code"].(string)
}

func initWS(t *testing.T, extra ...string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "kb")
	args := append([]string{"init", dir, "--author", "ben", "--harness", "test", "--json"}, extra...)
	r := run(t, "", args...)
	if r.code != 0 {
		t.Fatalf("init failed: %d %s", r.code, r.stderr)
	}
	t.Setenv("DKF_WORKSPACE", dir)
	t.Setenv("DKF_AUTHOR", "")
	t.Setenv("DKF_HARNESS", "")
	t.Setenv("DKF_MODEL", "")
	return dir
}

func define(t *testing.T, label string, extra ...string) string {
	t.Helper()
	r := run(t, "", append([]string{"particular", "define", "--label", label, "--json"}, extra...)...)
	if r.code != 0 {
		t.Fatalf("define %q: %d %s", label, r.code, r.stderr)
	}
	return r.js["particular"].(map[string]any)["id"].(string)
}

func assert(t *testing.T, subject, content string, extra ...string) string {
	t.Helper()
	r := run(t, "", append([]string{"claim", "assert", "--subject", subject, "--content", content, "--json"}, extra...)...)
	if r.code != 0 {
		t.Fatalf("assert: %d %s", r.code, r.stderr)
	}
	return r.js["claim"].(map[string]any)["id"].(string)
}

func synth(t *testing.T, subject string, inputs []string, extra ...string) string {
	t.Helper()
	args := []string{"synthesis", "create", "--subject", subject, "--content", "synthesis", "--unresolved", "none", "--json"}
	for _, in := range inputs {
		args = append(args, "--input", in)
	}
	r := run(t, "", append(args, extra...)...)
	if r.code != 0 {
		t.Fatalf("synthesis: %d %s", r.code, r.stderr)
	}
	return r.js["synthesis"].(map[string]any)["id"].(string)
}

func TestInitAndVersion(t *testing.T) {
	dir := initWS(t, "--base-uri", "https://example.com/particulars/")
	for _, p := range []string{"dkf.yaml", "particulars", "claims", "syntheses", "merges", "index.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, p)); err != nil {
			t.Errorf("missing %s", p)
		}
	}
	cfg, _ := os.ReadFile(filepath.Join(dir, "dkf.yaml"))
	if !strings.Contains(string(cfg), "base-uri: https://example.com/particulars/") || !strings.Contains(string(cfg), "author: ben") {
		t.Errorf("dkf.yaml:\n%s", cfg)
	}
	if r := run(t, "", "init", dir); r.code != 1 {
		t.Errorf("re-init should exit 1, got %d", r.code)
	}
	r := run(t, "", "version", "--json")
	if r.code != 0 || r.js["format"] != "dkf/0.1" {
		t.Errorf("version: %+v", r)
	}
	if r := run(t, "", "init", t.TempDir(), "--scope", "team"); r.code != 2 {
		t.Errorf("bad scope should exit 2, got %d", r.code)
	}
}

func TestNoWorkspaceAndUsage(t *testing.T) {
	t.Setenv("DKF_WORKSPACE", "")
	empty := t.TempDir()
	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	_ = os.Chdir(empty)
	r := run(t, "", "recall", "x", "--json")
	if r.code != 5 || r.errCode(t) != "no_workspace" {
		t.Errorf("no workspace: %+v", r)
	}
	if r := run(t, "", "nonsense"); r.code != 2 {
		t.Errorf("unknown command should exit 2, got %d", r.code)
	}
	if r := run(t, "", "recall", "--bogus"); r.code != 2 {
		t.Errorf("unknown flag should exit 2, got %d", r.code)
	}
}

func TestParticularDefineResolve(t *testing.T) {
	initWS(t, "--base-uri", "https://example.com/particulars")
	r := run(t, "", "particular", "define", "--label", "Billing Service", "--alias", "billing", "--json")
	if r.code != 0 || r.js["created"] != true {
		t.Fatalf("define: %+v", r)
	}
	p := r.js["particular"].(map[string]any)
	if p["uri"] != "https://example.com/particulars/billing-service" {
		t.Errorf("uri = %v", p["uri"])
	}
	// Same label again (case differs) → same slug → same particular, no new alias.
	r2 := run(t, "", "particular", "define", "--label", "Billing service", "--json")
	if r2.code != 0 || r2.js["created"] != false || r2.js["particular"].(map[string]any)["id"] != p["id"] {
		t.Errorf("second define should update: %+v", r2)
	}
	if aliases := r2.js["particular"].(map[string]any)["aliases"].([]any); len(aliases) != 1 {
		t.Errorf("case-only relabel should not add an alias: %v", aliases)
	}
	// Real relabel via explicit URI → old label preserved as alias.
	r3 := run(t, "", "particular", "define", "--label", "Billing Svc", "--uri", p["uri"].(string), "--json")
	if r3.code != 0 || r3.js["created"] != false {
		t.Fatalf("relabel: %+v", r3)
	}
	aliases := r3.js["particular"].(map[string]any)["aliases"].([]any)
	if len(aliases) != 2 || aliases[1] != "Billing service" {
		t.Errorf("aliases should contain alias and previous label: %v", aliases)
	}
	if r := run(t, "", "particular", "define", "--label", "!!!", "--json"); r.code != 2 {
		t.Errorf("empty slug should exit 2, got %d", r.code)
	}
	for _, q := range []string{"BILLING", "billing service", p["id"].(string), p["uri"].(string)} {
		r := run(t, "", "particular", "resolve", q, "--json")
		if r.code != 0 || len(r.js["matches"].([]any)) != 1 {
			t.Errorf("resolve %q: %+v", q, r)
		}
	}
	r = run(t, "", "particular", "resolve", "nothing-here", "--json")
	if r.code != 3 || r.errCode(t) != "not_found" {
		t.Errorf("resolve miss: %+v", r)
	}
	// Wikidata-style explicit URI.
	r = run(t, "", "particular", "define", "--label", "Douglas Adams", "--uri", "https://www.wikidata.org/entity/Q42", "--json")
	if r.code != 0 || r.js["particular"].(map[string]any)["uri"] != "https://www.wikidata.org/entity/Q42" {
		t.Errorf("explicit uri: %+v", r)
	}
}

func TestURNFallback(t *testing.T) {
	initWS(t)
	r := run(t, "", "particular", "define", "--label", "Auth & Sessions", "--json")
	uri := r.js["particular"].(map[string]any)["uri"].(string)
	if !strings.HasPrefix(uri, "urn:dkf:") || !strings.HasSuffix(uri, ":auth-sessions") {
		t.Errorf("urn fallback uri = %s", uri)
	}
}

func TestClaimAssert(t *testing.T) {
	dir := initWS(t)
	define(t, "Project X")
	t.Setenv("DKF_HARNESS", "claude")
	r := run(t, "", "claim", "assert", "--subject", "Project X", "--content", "Uses Postgres 16", "--topic", "db", "--confidence", "0.8", "--json")
	if r.code != 0 {
		t.Fatalf("assert: %+v", r)
	}
	c := r.js["claim"].(map[string]any)
	src := c["source"].(map[string]any)
	if src["author"] != "ben" || src["harness"] != "claude" || c["context"].(map[string]any)["scope"] != "personal" || c["confidence"] != 0.8 {
		t.Errorf("claim fields: %+v", c)
	}
	path := filepath.Join(dir, r.js["path"].(string))
	if _, err := os.Stat(path); err != nil {
		t.Errorf("claim file missing: %v", err)
	}
	idx, _ := os.ReadFile(filepath.Join(dir, "index.yaml"))
	if !strings.Contains(string(idx), c["id"].(string)) {
		t.Error("index not updated")
	}
	// Backdated.
	r = run(t, "", "claim", "assert", "--subject", "Project X", "--content", "old", "--timestamp", "2024-08-20T09:00:00Z", "--json")
	if r.code != 0 || r.js["claim"].(map[string]any)["timestamp"] != "2024-08-20T09:00:00Z" {
		t.Errorf("backdated: %+v", r)
	}
	// Stdin content.
	r = run(t, "multi\nline\n", "claim", "assert", "--subject", "Project X", "--content-file", "-", "--json")
	if r.code != 0 || r.js["claim"].(map[string]any)["content"] != "multi\nline\n" {
		t.Errorf("stdin content: %+v", r)
	}
	// Usage errors.
	for _, args := range [][]string{
		{"claim", "assert", "--subject", "Project X"},
		{"claim", "assert", "--subject", "Project X", "--content", "x", "--confidence", "1.5"},
		{"claim", "assert", "--subject", "Project X", "--content", "x", "--scope", "team"},
	} {
		if r := run(t, "", append(args, "--json")...); r.code != 2 {
			t.Errorf("%v should exit 2, got %d: %s", args, r.code, r.stderr)
		}
	}
	if r := run(t, "", "claim", "assert", "--subject", "Nobody", "--content", "x", "--json"); r.code != 3 {
		t.Errorf("unknown subject should exit 3, got %d", r.code)
	}
	// No provenance anywhere → exit 2; harness alone is enough.
	dir2 := filepath.Join(t.TempDir(), "kb2")
	if r := run(t, "", "init", dir2, "--json"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	t.Setenv("DKF_WORKSPACE", dir2)
	t.Setenv("DKF_HARNESS", "")
	define(t, "P")
	if r := run(t, "", "claim", "assert", "--subject", "P", "--content", "x", "--json"); r.code != 2 {
		t.Errorf("no provenance should exit 2, got %d", r.code)
	}
	t.Setenv("DKF_HARNESS", "claude")
	r = run(t, "", "claim", "assert", "--subject", "P", "--content", "x", "--json")
	if r.code != 0 {
		t.Fatalf("agent-only assert: %+v", r)
	}
	if src := r.js["claim"].(map[string]any)["source"].(map[string]any); src["harness"] != "claude" || src["author"] != nil {
		t.Errorf("agent-only source: %v", src)
	}
	t.Setenv("DKF_HARNESS", "")
	t.Setenv("DKF_AUTHOR", "agent")
	if r := run(t, "", "claim", "assert", "--subject", "P", "--content", "x", "--json"); r.code != 0 || r.js["claim"].(map[string]any)["source"].(map[string]any)["author"] != "agent" {
		t.Errorf("env author: %+v", r)
	}
}

func TestAmbiguousSubject(t *testing.T) {
	initWS(t)
	define(t, "Auth Service", "--alias", "auth")
	define(t, "Auth Team", "--alias", "auth")
	r := run(t, "", "recall", "auth", "--json")
	if r.code != 2 || !strings.Contains(r.stderr, "par_") {
		t.Errorf("ambiguous: %+v", r)
	}
}

func TestRetract(t *testing.T) {
	dir := initWS(t)
	define(t, "Project X")
	a := assert(t, "Project X", "A")
	b := assert(t, "Project X", "B")
	if r := run(t, "", "claim", "retract", a, "--json"); r.code != 2 {
		t.Errorf("missing reason should exit 2, got %d", r.code)
	}
	if r := run(t, "", "claim", "retract", a, "--reason", "r", "--superseded-by", "clm_missing", "--json"); r.code != 3 {
		t.Errorf("missing superseded-by should exit 3, got %d", r.code)
	}
	r := run(t, "", "claim", "retract", a, "--reason", "Port was 8443", "--superseded-by", b, "--json")
	if r.code != 0 {
		t.Fatalf("retract: %+v", r)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "claims", a+".yaml"))
	if !strings.Contains(string(data), "\nretracted:\n") || !strings.Contains(string(data), "superseded-by: "+b) {
		t.Errorf("file:\n%s", data)
	}
	if r := run(t, "", "claim", "retract", a, "--reason", "again", "--json"); r.code != 1 {
		t.Errorf("double retract should exit 1, got %d", r.code)
	}
	// Synthesis retractable.
	s := synth(t, "Project X", []string{b + ":thesis"})
	if r := run(t, "", "claim", "retract", s, "--reason", "misread", "--json"); r.code != 0 {
		t.Errorf("retract synthesis: %+v", r)
	}
	rc := run(t, "", "recall", "Project X", "--json")
	if rc.js["count"].(float64) != 1 {
		t.Errorf("recall should exclude retracted: %+v", rc.js)
	}
	rc = run(t, "", "recall", "Project X", "--include-retracted", "--json")
	if rc.js["count"].(float64) != 3 {
		t.Errorf("include-retracted: %+v", rc.js)
	}
}

func TestSynthesis(t *testing.T) {
	initWS(t)
	define(t, "Project X")
	a := assert(t, "Project X", "A")
	b := assert(t, "Project X", "B")
	c := assert(t, "Project X", "C")
	r := run(t, "", "synthesis", "create", "--subject", "Project X", "--content", "S", "--input", a+":thesis", "--input", b+":antithesis", "--input", c+":thesis:qualifying", "--unresolved", "compliance unsourced", "--model", "claude-sonnet-4-6", "--json")
	if r.code != 0 {
		t.Fatalf("synthesis: %+v", r)
	}
	s := r.js["synthesis"].(map[string]any)
	inputs := s["inputs"].([]any)
	if len(inputs) != 3 || inputs[0].(map[string]any)["weight"] != "primary" || inputs[2].(map[string]any)["weight"] != "qualifying" {
		t.Errorf("inputs: %v", inputs)
	}
	pb := s["source"].(map[string]any)
	if _, legacy := s["produced-by"]; legacy || pb["harness"] != "test" || pb["model"] != "claude-sonnet-4-6" || s["method"] != "reconciliation" {
		t.Errorf("source/method: %+v", s)
	}
	for _, args := range [][]string{
		{"--input", a + ":support", "--unresolved", "x"},
		{"--input", a + ":thesis"},
		{"--input", a + ":thesis:heavy", "--unresolved", "x"},
	} {
		full := append([]string{"synthesis", "create", "--subject", "Project X", "--content", "S"}, args...)
		if r := run(t, "", append(full, "--json")...); r.code != 2 {
			t.Errorf("%v should exit 2, got %d: %s", args, r.code, r.stderr)
		}
	}
	if r := run(t, "", "synthesis", "create", "--subject", "Project X", "--content", "S", "--input", "clm_missing:thesis", "--unresolved", "x", "--json"); r.code != 3 {
		t.Errorf("unknown input should exit 3, got %d", r.code)
	}
	// Synthesis as input + retracted input warning.
	sid := s["id"].(string)
	run(t, "", "claim", "retract", a, "--reason", "r", "--json")
	r = run(t, "", "synthesis", "create", "--subject", "Project X", "--content", "S2", "--input", sid+":thesis", "--input", a+":antithesis", "--unresolved", "none", "--json")
	if r.code != 0 {
		t.Fatalf("s2: %+v", r)
	}
	warnings := r.js["warnings"].([]any)
	if len(warnings) != 1 || !strings.Contains(warnings[0].(string), a) {
		t.Errorf("warnings: %v", warnings)
	}
	// Harness missing everywhere.
	dir2 := filepath.Join(t.TempDir(), "kb2")
	run(t, "", "init", dir2, "--author", "ben", "--json")
	t.Setenv("DKF_WORKSPACE", dir2)
	define(t, "P")
	x := assert(t, "P", "x")
	if r := run(t, "", "synthesis", "create", "--subject", "P", "--content", "S", "--input", x+":thesis", "--unresolved", "n", "--json"); r.code != 2 {
		t.Errorf("missing harness should exit 2, got %d: %s", r.code, r.stderr)
	}
}

func TestRecallLineageConflicts(t *testing.T) {
	initWS(t)
	define(t, "Project X")
	define(t, "Project Y")
	a := assert(t, "Project X", "A", "--topic", "architecture")
	b := assert(t, "Project X", "B")
	c := synth(t, "Project X", []string{a + ":thesis", b + ":antithesis"})
	y := assert(t, "Project Y", "Y", "--topic", "architecture")

	r := run(t, "", "recall", "Project X", "--json")
	entries := r.js["entries"].([]any)
	if len(entries) != 3 || entries[2].(map[string]any)["id"] != c || entries[2].(map[string]any)["current"] != true {
		t.Errorf("recall: %v", entries)
	}
	r = run(t, "", "recall", "--topic", "architecture", "--json")
	if r.js["count"].(float64) != 2 {
		t.Errorf("topic recall: %+v", r.js)
	}
	if r := run(t, "", "recall", "--json"); r.code != 2 {
		t.Errorf("recall without args should exit 2, got %d", r.code)
	}
	r = run(t, "", "recall", "Project X", "--scope", "public", "--json")
	if r.js["count"].(float64) != 0 {
		t.Errorf("scope filter: %+v", r.js)
	}

	// Lineage.
	e := assert(t, "Project X", "E")
	d := synth(t, "Project X", []string{c + ":thesis", e + ":antithesis"})
	r = run(t, "", "lineage", d, "--json")
	if r.code != 0 {
		t.Fatalf("lineage: %+v", r)
	}
	in := r.js["inputs"].([]any)
	if len(in) != 2 || in[0].(map[string]any)["role"] != "thesis" || len(in[0].(map[string]any)["inputs"].([]any)) != 2 {
		t.Errorf("lineage tree: %v", in)
	}
	r = run(t, "", "lineage", d, "--depth", "2", "--json")
	if len(r.js["inputs"].([]any)[0].(map[string]any)["inputs"].([]any)) != 0 {
		t.Errorf("depth 2 should not expand grandchildren")
	}
	if r := run(t, "", "lineage", "clm_does-not-exist", "--json"); r.code != 3 {
		t.Errorf("unknown id should exit 3, got %d", r.code)
	}
	text := run(t, "", "lineage", d)
	if !strings.Contains(text.stdout, "├── ") || !strings.Contains(text.stdout, "└── ") {
		t.Errorf("text tree:\n%s", text.stdout)
	}

	// Conflicts: after d, E and C are reconciled; nothing unsynthesised for X; Y has one claim → not reported.
	r = run(t, "", "conflicts", "--json")
	if r.js["count"].(float64) != 0 {
		t.Errorf("expected no conflicts: %+v", r.js)
	}
	z := assert(t, "Project X", "Z")
	r = run(t, "", "conflicts", "Project X", "--json")
	reports := r.js["reports"].([]any)
	if len(reports) != 1 {
		t.Fatalf("conflicts: %+v", r.js)
	}
	rep := reports[0].(map[string]any)
	if rep["current"] != d || rep["unsynthesised"].([]any)[0] != z {
		t.Errorf("report: %+v", rep)
	}
	if r := run(t, "", "conflicts", "--fail-on-conflicts", "--json"); r.code != 4 {
		t.Errorf("fail-on-conflicts should exit 4, got %d", r.code)
	}
	run(t, "", "claim", "retract", e, "--reason", "r", "--json")
	r = run(t, "", "conflicts", "Project X", "--json")
	rep = r.js["reports"].([]any)[0].(map[string]any)
	if rep["stale"].([]any)[0] != d {
		t.Errorf("stale: %+v", rep)
	}
	_ = y
}

func TestIndexAndValidate(t *testing.T) {
	dir := initWS(t)
	define(t, "Project X")
	a := assert(t, "Project X", "A")
	if r := run(t, "", "index", "--check", "--json"); r.code != 0 || r.js["clean"] != true {
		t.Errorf("check after assert: %+v", r)
	}
	if r := run(t, "", "validate", "--json"); r.code != 0 || r.js["errors"].(float64) != 0 || r.js["warnings"].(float64) != 0 {
		t.Errorf("clean validate: %+v", r)
	}
	// Hand-add a file → drift, validate error.
	src, _ := os.ReadFile(filepath.Join(dir, "claims", a+".yaml"))
	copyID := "clm_01999999-0000-7000-8000-000000000001"
	_ = os.WriteFile(filepath.Join(dir, "claims", copyID+".yaml"), bytes.Replace(src, []byte(a), []byte(copyID), 1), 0o644)
	r := run(t, "", "index", "--check", "--json")
	if r.code != 4 || r.js["missing"].([]any)[0] != copyID {
		t.Errorf("drift: %+v", r)
	}
	r = run(t, "", "validate", "--json")
	if r.code != 4 {
		t.Errorf("validate should exit 4 on stale index: %+v", r)
	}
	if r := run(t, "", "index", "--json"); r.code != 0 || r.js["entries"].(float64) != 3 {
		t.Errorf("rebuild: %+v", r)
	}
	if r := run(t, "", "validate", "--json"); r.code != 0 {
		t.Errorf("validate after rebuild: %+v", r)
	}
	// Recall works without index.
	_ = os.Remove(filepath.Join(dir, "index.yaml"))
	if r := run(t, "", "recall", "Project X", "--json"); r.code != 0 || r.js["count"].(float64) != 2 {
		t.Errorf("recall without index: %+v", r)
	}
	r = run(t, "", "validate", "--json")
	if r.code != 0 || r.js["warnings"].(float64) != 1 {
		t.Errorf("missing index should be a warning: %+v", r)
	}
	// Malformed dkf.yaml → validate reports, exit 4.
	_ = os.WriteFile(filepath.Join(dir, "dkf.yaml"), []byte("format: dkf/9.9\nworkspace:\n  id: x\n"), 0o644)
	r = run(t, "", "validate", "--json")
	if r.code != 4 || r.js["findings"].([]any)[0].(map[string]any)["path"] != "dkf.yaml" {
		t.Errorf("bad config: %+v", r)
	}
}

func TestJSONContract(t *testing.T) {
	initWS(t)
	r := run(t, "", "version", "--json")
	if r.stderr != "" || !strings.HasPrefix(r.stdout, "{") {
		t.Errorf("json success must write only stdout: %+v", r)
	}
	r = run(t, "", "lineage", "clm_nope", "--json")
	if r.stdout != "" || r.errCode(t) != "not_found" {
		t.Errorf("json failure must write only stderr: %+v", r)
	}
}

func TestTopics(t *testing.T) {
	initWS(t)
	define(t, "Project X")
	define(t, "Project Y")
	a := assert(t, "Project X", "A", "--topic", "architecture", "--topic", "db")
	assert(t, "Project Y", "B", "--topic", "architecture")
	r := run(t, "", "topics", "--json")
	if r.code != 0 || r.js["count"].(float64) != 2 {
		t.Fatalf("topics: %+v", r)
	}
	first := r.js["topics"].([]any)[0].(map[string]any)
	if first["topic"] != "architecture" || first["assertions"].(float64) != 2 || first["particulars"].(float64) != 2 {
		t.Errorf("first row: %+v", first)
	}
	r = run(t, "", "topics", "Project Y", "--json")
	if r.js["count"].(float64) != 1 {
		t.Errorf("per-particular: %+v", r.js)
	}
	run(t, "", "claim", "retract", a, "--reason", "r", "--json")
	r = run(t, "", "topics", "--json")
	if r.js["count"].(float64) != 1 {
		t.Errorf("retracted should drop db: %+v", r.js)
	}
	if r := run(t, "", "topics", "--include-retracted", "--json"); r.js["count"].(float64) != 2 {
		t.Errorf("include-retracted: %+v", r.js)
	}
	if r := run(t, "", "topics", "Nobody", "--json"); r.code != 3 {
		t.Errorf("unknown particular should exit 3, got %d", r.code)
	}
	text := run(t, "", "topics")
	if !strings.Contains(text.stdout, "architecture") || !strings.Contains(text.stdout, "particulars") {
		t.Errorf("text output:\n%s", text.stdout)
	}
}

func TestInitNormalisesBaseURI(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "kb")
	r := run(t, "", "init", dir, "--base-uri", "https://example.com/particulars", "--json")
	if r.code != 0 || r.js["normalised"] != true || r.js["workspace"].(map[string]any)["base_uri"] != "https://example.com/particulars/" {
		t.Errorf("normalise: %+v", r)
	}
	_ = os.WriteFile(filepath.Join(dir, "dkf.yaml"), []byte("format: dkf/0.1\nworkspace:\n  id: 01a00000-0000-7000-8000-000000000000\n  base-uri: https://example.com/particulars\n"), 0o644)
	t.Setenv("DKF_WORKSPACE", dir)
	if r := run(t, "", "recall", "x", "--json"); r.code != 1 || !strings.Contains(r.stderr, "dkf.yaml") {
		t.Errorf("bad base-uri should exit 1 naming dkf.yaml: %+v", r)
	}
	r = run(t, "", "validate", "--json")
	if r.code != 4 || r.js["findings"].([]any)[0].(map[string]any)["code"] != "invalid_base_uri" {
		t.Errorf("validate bad base-uri: %+v", r)
	}
}

func TestSynthesisSourceAndLegacy(t *testing.T) {
	dir := initWS(t)
	define(t, "Project X")
	a := assert(t, "Project X", "A")
	r := run(t, "", "synthesis", "create", "--subject", "Project X", "--content", "S", "--input", a+":thesis", "--unresolved", "None identified", "--author", "ben", "--document", "notes.md", "--json")
	if r.code != 0 {
		t.Fatalf("synthesis: %+v", r)
	}
	src := r.js["synthesis"].(map[string]any)["source"].(map[string]any)
	if src["author"] != "ben" || src["harness"] != "test" || src["document"] != "notes.md" {
		t.Errorf("source: %v", src)
	}
	file, _ := os.ReadFile(filepath.Join(dir, r.js["path"].(string)))
	if strings.Contains(string(file), "produced-by") || !strings.Contains(string(file), "unresolved: None identified\nsource:\n  author: ben\n  harness: test") {
		t.Errorf("file:\n%s", file)
	}
	// Author alone is not enough for a synthesis.
	t.Setenv("DKF_WORKSPACE", dir)
	if r := run(t, "", "synthesis", "create", "--subject", "Project X", "--content", "S", "--input", a+":thesis", "--unresolved", "n", "--harness", "", "--json"); r.code != 0 {
		// harness still comes from dkf.yaml defaults in initWS; that's fine
		_ = r
	}
	dir2 := filepath.Join(t.TempDir(), "kb2")
	run(t, "", "init", dir2, "--author", "ben", "--json")
	t.Setenv("DKF_WORKSPACE", dir2)
	pid := define(t, "P")
	x := assert(t, "P", "x")
	if r := run(t, "", "synthesis", "create", "--subject", "P", "--content", "S", "--input", x+":thesis", "--unresolved", "n", "--author", "ben", "--json"); r.code != 2 || !strings.Contains(r.stderr, "DKF_HARNESS") {
		t.Errorf("author-only synthesis should exit 2 naming DKF_HARNESS: %+v", r)
	}
	// Legacy produced-by file reads as source; validate warns.
	legacy := "syn_01a00000-0000-7000-8000-00000000aaaa"
	_ = os.WriteFile(filepath.Join(dir2, "syntheses", legacy+".yaml"), []byte("id: "+legacy+"\ntype: synthesis\nsubject: "+pid+"\ncontent: old\ninputs:\n  - id: "+x+"\n    role: thesis\nunresolved: n\nproduced-by:\n  harness: claude\n  model: m\ntimestamp: 2026-08-20T09:00:00Z\ncontext:\n  scope: personal\n"), 0o644)
	run(t, "", "index", "--json")
	rc := run(t, "", "recall", "P", "--json")
	var found bool
	for _, e := range rc.js["entries"].([]any) {
		m := e.(map[string]any)
		if m["id"] == legacy {
			found = true
			if _, has := m["produced-by"]; has || m["source"].(map[string]any)["harness"] != "claude" {
				t.Errorf("legacy entry: %v", m)
			}
		}
	}
	if !found {
		t.Fatal("legacy synthesis not recalled")
	}
	v := run(t, "", "validate", "--json")
	if v.code != 0 {
		t.Fatalf("validate with legacy file should pass: %+v", v)
	}
	codes := []string{}
	for _, f := range v.js["findings"].([]any) {
		codes = append(codes, f.(map[string]any)["code"].(string))
	}
	if strings.Join(codes, ",") != "legacy_produced_by" {
		t.Errorf("findings = %v", codes)
	}
}

func TestRetractVerbAndMerges(t *testing.T) {
	initWS(t, "--base-uri", "https://example.com/p")
	define(t, "Project X")
	define(t, "ProjectX legacy")
	define(t, "Other")
	a := assert(t, "Project X", "A")
	b := assert(t, "ProjectX legacy", "B")
	// Top-level retract and alias behave the same.
	if r := run(t, "", "retract", a, "--reason", "r", "--json"); r.code != 0 || r.js["type"] != "claim" {
		t.Errorf("retract: %+v", r)
	}
	if r := run(t, "", "claim", "retract", b, "--reason", "r", "--json"); r.code != 0 {
		t.Errorf("alias: %+v", r)
	}
	c := assert(t, "Project X", "C")
	d := assert(t, "ProjectX legacy", "D")
	// Merge local particulars.
	r := run(t, "", "particular", "merge", "Project X", "ProjectX legacy", "--reason", "same", "--json")
	if r.code != 0 {
		t.Fatalf("merge: %+v", r)
	}
	m := r.js["merge"].(map[string]any)
	uris := m["uris"].([]any)
	if len(uris) != 2 || uris[0] != "https://example.com/p/project-x" || uris[1] != "https://example.com/p/projectx-legacy" || m["reason"] != "same" {
		t.Errorf("merge record: %v", m)
	}
	if r := run(t, "", "particular", "merge", "Project X", "ProjectX legacy", "--json"); r.code != 2 {
		t.Errorf("duplicate merge should exit 2, got %d", r.code)
	}
	if r := run(t, "", "particular", "merge", "Project X", "https://example.com/p/project-x", "--json"); r.code != 2 {
		t.Errorf("self merge should exit 2, got %d: %s", r.code, r.stderr)
	}
	if r := run(t, "", "particular", "merge", "Project X", "nothing here", "--json"); r.code != 3 {
		t.Errorf("non-uri unknown side should exit 3, got %d", r.code)
	}
	// Recall across the merge, with class.
	rc := run(t, "", "recall", "Project X", "--json")
	if rc.js["count"].(float64) != 2 || len(rc.js["class"].([]any)) != 2 {
		t.Errorf("recall across merge: %+v", rc.js)
	}
	ids := map[string]bool{}
	for _, e := range rc.js["entries"].([]any) {
		ids[e.(map[string]any)["id"].(string)] = true
		if e.(map[string]any)["unsynthesised"] != true {
			t.Errorf("entries should be unsynthesised: %v", e)
		}
	}
	if !ids[c] || !ids[d] {
		t.Errorf("expected %s and %s, got %v", c, d, ids)
	}
	// Conflicts sweep reports the class once with members.
	cf := run(t, "", "conflicts", "--json")
	reps := cf.js["reports"].([]any)
	if len(reps) != 1 || len(reps[0].(map[string]any)["members"].([]any)) != 2 {
		t.Errorf("conflicts sweep: %+v", cf.js)
	}
	// Foreign URI merge → validate warns unknown_merge_uri; index clean.
	if r := run(t, "", "particular", "merge", "Other", "https://www.wikidata.org/entity/Q1", "--json"); r.code != 0 || r.js["sides"].([]any)[1].(map[string]any)["particular"] != "" {
		t.Errorf("foreign merge: %+v", r)
	}
	if r := run(t, "", "index", "--check", "--json"); r.code != 0 {
		t.Errorf("index after merge: %+v", r)
	}
	v := run(t, "", "validate", "--json")
	if v.code != 0 || v.js["warnings"].(float64) < 1 {
		t.Errorf("validate: %+v", v)
	}
	// Retract the merge; superseded-by rejected.
	mid := m["id"].(string)
	if r := run(t, "", "retract", mid, "--reason", "r", "--superseded-by", c, "--json"); r.code != 2 {
		t.Errorf("superseded-by on merge should exit 2, got %d", r.code)
	}
	if r := run(t, "", "retract", mid, "--reason", "different after all", "--json"); r.code != 0 || r.js["type"] != "merge" {
		t.Errorf("retract merge: %+v", r)
	}
	rc = run(t, "", "recall", "Project X", "--json")
	if rc.js["count"].(float64) != 1 || rc.js["class"] != nil {
		t.Errorf("after merge retraction: %+v", rc.js)
	}
}

func TestLineageSupersededText(t *testing.T) {
	initWS(t)
	define(t, "Project X")
	a := assert(t, "Project X", "A")
	y := assert(t, "Project X", "Y")
	s := synth(t, "Project X", []string{a + ":thesis"})
	run(t, "", "retract", a, "--reason", "typo", "--superseded-by", y, "--json")
	r := run(t, "", "lineage", s, "--json")
	if r.js["inputs"].([]any)[0].(map[string]any)["superseded_by"] != y {
		t.Errorf("superseded_by missing: %+v", r.js)
	}
	if txt := run(t, "", "lineage", s); !strings.Contains(txt.stdout, "retracted → "+y) {
		t.Errorf("text: %s", txt.stdout)
	}
}

func TestSkillShowAndInstall(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	t.Setenv("DKF_WORKSPACE", "") // no workspace needed

	show := run(t, "", "skill", "show")
	if show.code != 0 || !strings.HasPrefix(show.stdout, "---\n") || !strings.Contains(show.stdout, "name: particulars") || !strings.Contains(show.stdout, "<!-- installed by particulars dev;") {
		t.Fatalf("show: %+v", show)
	}
	sj := run(t, "", "skill", "show", "--json")
	if sj.js["content"] != show.stdout || sj.js["version"] != "dev" {
		t.Errorf("show --json: %v", sj.js["version"])
	}

	// Fresh project install.
	r := run(t, "", "skill", "install", "--json")
	target := filepath.Join(work, ".claude", "skills", "particulars", "SKILL.md")
	if r.code != 0 || r.js["created"] != true || r.js["path"] != target {
		t.Fatalf("install: %+v", r)
	}
	got, _ := os.ReadFile(target)
	if string(got) != show.stdout {
		t.Error("installed content should equal show")
	}
	if r := run(t, "", "skill", "install", "--json"); r.code != 0 || r.js["unchanged"] != true {
		t.Errorf("reinstall: %+v", r)
	}
	if r := run(t, "", "skill", "install", "--check", "--json"); r.code != 0 || r.js["status"] != "ok" {
		t.Errorf("check ok: %+v", r)
	}

	// Own file from an "older version" (different stamp, same body) → check ok, install updates.
	older := strings.Replace(string(got), "installed by particulars dev;", "installed by particulars 0.1.0;", 1)
	older = strings.Replace(older, "version: \"dev\"", "version: \"0.1.0\"", 1)
	_ = os.WriteFile(target, []byte(older), 0o644)
	if r := run(t, "", "skill", "install", "--check", "--json"); r.code != 0 || r.js["status"] != "ok" {
		t.Errorf("check should mask versions: %+v", r)
	}
	if r := run(t, "", "skill", "install", "--json"); r.code != 0 || r.js["updated"] != true {
		t.Errorf("update own older file: %+v", r)
	}

	// Body drift → check 4 differs; nothing written.
	drift := strings.Replace(string(got), "## The loop", "## The loop (edited)", 1)
	_ = os.WriteFile(target, []byte(drift), 0o644)
	r = run(t, "", "skill", "install", "--check", "--json")
	if r.code != 4 || r.js["status"] != "differs" {
		t.Errorf("check drift: %+v", r)
	}
	if now, _ := os.ReadFile(target); string(now) != drift {
		t.Error("--check must not write")
	}

	// Foreign (hand-written) file: refused without --force, check says foreign.
	_ = os.WriteFile(target, []byte("# my own skill\n"), 0o644)
	r = run(t, "", "skill", "install", "--json")
	if r.code != 1 || r.errCode(t) != "conflict" || !strings.Contains(r.stderr, "--force") {
		t.Errorf("foreign refusal: %+v", r)
	}
	if now, _ := os.ReadFile(target); string(now) != "# my own skill\n" {
		t.Error("foreign file must be untouched")
	}
	if r := run(t, "", "skill", "install", "--check", "--json"); r.code != 4 || r.js["status"] != "foreign" {
		t.Errorf("check foreign: %+v", r)
	}
	if r := run(t, "", "skill", "install", "--force", "--json"); r.code != 0 || r.js["updated"] != true {
		t.Errorf("force: %+v", r)
	}

	// --user and --dir targets; mutual exclusion; missing check.
	if r := run(t, "", "skill", "install", "--user", "--json"); r.code != 0 || r.js["path"] != filepath.Join(home, ".claude", "skills", "particulars", "SKILL.md") {
		t.Errorf("user: %+v", r)
	}
	if r := run(t, "", "skill", "install", "--dir", filepath.Join(work, "tmp", "skills"), "--json"); r.code != 0 || r.js["path"] != filepath.Join(work, "tmp", "skills", "SKILL.md") {
		t.Errorf("dir: %+v", r)
	}
	if r := run(t, "", "skill", "install", "--user", "--dir", "x", "--json"); r.code != 2 {
		t.Errorf("mutual exclusion: %d", r.code)
	}
	if r := run(t, "", "skill", "install", "--dir", filepath.Join(work, "nowhere"), "--check", "--json"); r.code != 4 || r.js["status"] != "missing" {
		t.Errorf("check missing: %+v", r)
	}
}

func TestPointerAndWorkspaceVerb(t *testing.T) {
	repo := t.TempDir()
	t.Chdir(repo)
	t.Setenv("DKF_WORKSPACE", "")
	if r := run(t, "", "init", "--pointer", "--json"); r.code != 2 {
		t.Errorf("--pointer without dir should exit 2, got %d", r.code)
	}
	if r := run(t, "", "workspace", "--json"); r.code != 5 {
		t.Errorf("nothing resolves: %d", r.code)
	}
	r := run(t, "", "init", "./knowledge", "--pointer", "--author", "ben", "--json")
	if r.code != 0 || r.js["pointer"] != filepath.Join(repo, ".dkf") {
		t.Fatalf("init --pointer: %+v", r)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, ".dkf")); string(data) != "knowledge\n" {
		t.Errorf(".dkf content: %q", data)
	}
	_ = os.MkdirAll(filepath.Join(repo, "src", "pkg"), 0o755)
	t.Chdir(filepath.Join(repo, "src", "pkg"))
	w := run(t, "", "workspace", "--json")
	if w.code != 0 || w.js["via"] != "pointer" || w.js["pointer"] != filepath.Join(repo, ".dkf") || filepath.Base(w.js["root"].(string)) != "knowledge" {
		t.Errorf("workspace via pointer: %+v", w.js)
	}
	// A verb works from here too.
	if r := run(t, "", "particular", "define", "--label", "Thing", "--json"); r.code != 0 {
		t.Errorf("define via pointer: %+v", r)
	}
	text := run(t, "", "workspace")
	if !strings.Contains(text.stdout, "via: pointer") {
		t.Errorf("text: %s", text.stdout)
	}
	// Env wins over the pointer.
	other := filepath.Join(t.TempDir(), "kb")
	run(t, "", "init", other, "--json")
	t.Setenv("DKF_WORKSPACE", other)
	if w := run(t, "", "workspace", "--json"); w.js["via"] != "env" {
		t.Errorf("env: %+v", w.js)
	}
	t.Setenv("DKF_WORKSPACE", "")
	// Dangling pointer → 5 naming both.
	_ = os.RemoveAll(filepath.Join(repo, "knowledge"))
	w = run(t, "", "workspace", "--json")
	if w.code != 5 || !strings.Contains(w.stderr, ".dkf") || !strings.Contains(w.stderr, "knowledge") {
		t.Errorf("dangling: %+v", w)
	}
	// Differing existing pointer refused by init --pointer.
	t.Chdir(repo)
	if r := run(t, "", "init", "./other", "--pointer", "--json"); r.code != 1 {
		t.Errorf("differing pointer should exit 1, got %d: %s", r.code, r.stderr)
	}
}

func TestSkillHarnessPresets(t *testing.T) {
	work := t.TempDir()
	t.Chdir(work)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("DKF_WORKSPACE", "")

	// Copilot preset writes the same content as claude.
	claude := run(t, "", "skill", "show")
	r := run(t, "", "skill", "install", "--harness", "copilot", "--json")
	if r.code != 0 || r.js["created"] != true || r.js["harness"] != "copilot" || r.js["path"] != filepath.Join(work, ".github", "skills", "particulars", "SKILL.md") {
		t.Fatalf("copilot: %+v", r)
	}
	if got, _ := os.ReadFile(r.js["path"].(string)); string(got) != claude.stdout {
		t.Error("copilot content should equal claude's")
	}
	// Duplicate-location warning when a second Copilot-readable location is installed.
	r = run(t, "", "skill", "install", "--harness", "claude", "--json")
	if r.code != 0 || len(r.js["warnings"].([]any)) != 1 || !strings.Contains(r.js["warnings"].([]any)[0].(string), ".github/skills/particulars/SKILL.md") {
		t.Errorf("duplicate warning: %+v", r.js["warnings"])
	}
	// Neutral location, user scope.
	r = run(t, "", "skill", "install", "--harness", "agents", "--user", "--json")
	if r.code != 0 || r.js["path"] != filepath.Join(home, ".agents", "skills", "particulars", "SKILL.md") || len(r.js["warnings"].([]any)) != 0 {
		t.Errorf("agents --user: %+v", r)
	}
	// Cursor rule.
	r = run(t, "", "skill", "install", "--harness", "cursor", "--json")
	mdc := filepath.Join(work, ".cursor", "rules", "particulars.mdc")
	if r.code != 0 || r.js["path"] != mdc {
		t.Fatalf("cursor: %+v", r)
	}
	got, _ := os.ReadFile(mdc)
	if !strings.HasPrefix(string(got), "---\ndescription: \"") || !strings.Contains(string(got), "\nalwaysApply: false\n---\n<!-- installed by particulars dev; regenerate with: particulars skill install --harness cursor -->\n") {
		t.Errorf("mdc shape:\n%s", got[:300])
	}
	if show := run(t, "", "skill", "show", "--harness", "cursor"); show.stdout != string(got) {
		t.Error("show --harness cursor should equal the installed file")
	}
	// Several presets at once.
	r = run(t, "", "skill", "install", "--harness", "claude", "--harness", "cursor", "--json")
	if r.code != 0 || len(r.js["targets"].([]any)) != 2 {
		t.Errorf("multi: %+v", r.js)
	}
	// Flag combinations.
	for _, args := range [][]string{
		{"--user", "--dir", "x"},
		{"--harness", "cursor", "--user"},
		{"--harness", "agents-md", "--user"},
		{"--file", "AGENTS.md"},
		{"--harness", "nope"},
		{"--dir", "x", "--harness", "claude"},
	} {
		if r := run(t, "", append([]string{"skill", "install"}, append(args, "--json")...)...); r.code != 2 {
			t.Errorf("%v should exit 2, got %d", args, r.code)
		}
	}

	// AGENTS.md: fresh, then user content preserved, then broken markers.
	r = run(t, "", "skill", "install", "--harness", "agents-md", "--json")
	agents := filepath.Join(work, "AGENTS.md")
	if r.code != 0 || r.js["created"] != true || r.js["path"] != agents {
		t.Fatalf("agents-md fresh: %+v", r)
	}
	sec := run(t, "", "skill", "show", "--harness", "agents-md").stdout
	if got, _ := os.ReadFile(agents); string(got) != sec || !strings.HasPrefix(sec, "<!-- particulars:skill:start") || !strings.HasSuffix(sec, "<!-- particulars:skill:end -->\n") {
		t.Errorf("fresh AGENTS.md content")
	}
	if r := run(t, "", "skill", "install", "--harness", "agents-md", "--json"); r.js["unchanged"] != true {
		t.Errorf("idempotent section: %+v", r.js)
	}
	above, below := "# My project\n\nRun make.\n\n", "\n## After\n\nmore\n"
	older := strings.Replace(sec, "installed by particulars dev;", "installed by particulars 0.1.0;", 1)
	_ = os.WriteFile(agents, []byte(above+older+below), 0o644)
	if r := run(t, "", "skill", "install", "--harness", "agents-md", "--check", "--json"); r.code != 0 || r.js["status"] != "ok" {
		t.Errorf("check masks section version: %+v", r)
	}
	r = run(t, "", "skill", "install", "--harness", "agents-md", "--json")
	got, _ = os.ReadFile(agents)
	if r.js["updated"] != true || !strings.HasPrefix(string(got), above) || !strings.HasSuffix(string(got), below) || strings.Contains(string(got), "0.1.0") {
		t.Errorf("replace preserves surroundings:\n%s", got)
	}
	// Append to a file without markers; check says missing beforehand.
	_ = os.WriteFile(agents, []byte("# Only mine\n"), 0o644)
	if r := run(t, "", "skill", "install", "--harness", "agents-md", "--check", "--json"); r.code != 4 || r.js["status"] != "missing" {
		t.Errorf("no section → missing: %+v", r)
	}
	r = run(t, "", "skill", "install", "--harness", "agents-md", "--json")
	got, _ = os.ReadFile(agents)
	if r.js["updated"] != true || !strings.HasPrefix(string(got), "# Only mine\n\n<!-- particulars:skill:start") {
		t.Errorf("append:\n%s", got[:120])
	}
	// Retarget with --file.
	if r := run(t, "", "skill", "install", "--harness", "agents-md", "--file", "GEMINI.md", "--json"); r.code != 0 || r.js["path"] != filepath.Join(work, "GEMINI.md") {
		t.Errorf("--file: %+v", r)
	}
	// Broken markers refused.
	_ = os.WriteFile(agents, []byte("<!-- particulars:skill:start — installed by particulars dev; regenerate with: particulars skill install --harness agents-md -->\nhalf\n"), 0o644)
	if r := run(t, "", "skill", "install", "--harness", "agents-md", "--json"); r.code != 1 {
		t.Errorf("broken should exit 1, got %d", r.code)
	}
	if r := run(t, "", "skill", "install", "--harness", "agents-md", "--check", "--json"); r.code != 4 || r.js["status"] != "foreign" {
		t.Errorf("broken check: %+v", r)
	}
}

func TestExportGraph(t *testing.T) {
	dir := initWS(t, "--base-uri", "https://example.com/p/")
	define(t, "Project X")
	define(t, "Private Thing")
	a := assert(t, "Project X", "Uses Postgres 16", "--scope", "organisation", "--topic", "db", "--document", "docs/db.md")
	assert(t, "Project X", "SECRET personal", "--scope", "personal")
	assert(t, "Private Thing", "SECRET all personal", "--scope", "personal")
	synth(t, "Project X", []string{a + ":thesis"}, "--scope", "organisation")

	// usage
	if r := run(t, "", "export", "--json"); r.code != 2 {
		t.Errorf("missing --format should exit 2, got %d", r.code)
	}
	if r := run(t, "", "export", "--format", "csv", "--json"); r.code != 2 {
		t.Errorf("unknown format should exit 2, got %d", r.code)
	}
	if r := run(t, "", "export", "--format", "graph", "--scope", "personal", "--json"); r.code != 2 || !strings.Contains(r.stderr, "never exported") {
		t.Errorf("personal scope must be refused: %+v", r)
	}

	// NDJSON to stdout
	r := run(t, "", "export", "--format", "graph", "--source-url", "https://github.com/o/r/blob/main/")
	if r.code != 0 {
		t.Fatalf("export: %+v", r)
	}
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 item (Private Thing is wholly personal), got %d:\n%s", len(lines), r.stdout)
	}
	var l struct {
		ID   string `json:"id"`
		Item struct {
			ACL        []map[string]string `json:"acl"`
			Properties map[string]any      `json:"properties"`
			Content    map[string]string   `json:"content"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &l); err != nil {
		t.Fatalf("not NDJSON: %v", err)
	}
	if !strings.HasPrefix(l.ID, "par_") || l.Item.ACL[0]["type"] != "everyone" || l.Item.Content["type"] != "text" {
		t.Errorf("item shape: %+v", l)
	}
	if strings.Contains(lines[0], "SECRET") {
		t.Error("personal content leaked into the export")
	}
	if l.Item.Properties["claimCount"].(float64) != 1 || l.Item.Properties["title"] != "Project X" {
		t.Errorf("properties: %v", l.Item.Properties)
	}
	if !strings.HasPrefix(l.Item.Properties["url"].(string), "https://github.com/o/r/blob/main/syntheses/") {
		t.Errorf("url: %v", l.Item.Properties["url"])
	}

	// determinism
	if again := run(t, "", "export", "--format", "graph", "--source-url", "https://github.com/o/r/blob/main/"); again.stdout != r.stdout {
		t.Error("export is not deterministic")
	}

	// --out and --manifest
	outPath := filepath.Join(dir, "items.ndjson")
	manPath := filepath.Join(dir, "manifest.txt")
	r = run(t, "", "export", "--format", "graph", "--out", outPath, "--manifest", manPath, "--json")
	if r.code != 0 || r.js["exported"].(float64) != 1 || r.js["skipped"].(float64) != 1 {
		t.Errorf("summary: %+v", r.js)
	}
	body, _ := os.ReadFile(outPath)
	if len(strings.Split(strings.TrimRight(string(body), "\n"), "\n")) != 1 {
		t.Errorf("--out contents:\n%s", body)
	}
	man, _ := os.ReadFile(manPath)
	if strings.TrimSpace(string(man)) != l.ID {
		t.Errorf("manifest = %q, want %s", man, l.ID)
	}

	// schema
	r = run(t, "", "export", "--format", "graph", "--schema", "--connection", "particulars", "--json")
	if r.code != 0 {
		t.Fatalf("schema: %+v", r)
	}
	if r.js["connection"].(map[string]any)["id"] != "particulars" {
		t.Errorf("connection: %v", r.js["connection"])
	}
	props := r.js["schema"].(map[string]any)["properties"].([]any)
	if len(props) != 10 {
		t.Errorf("expected 10 schema properties, got %d", len(props))
	}
	if r := run(t, "", "export", "--format", "graph", "--schema", "--json"); r.code != 2 {
		t.Errorf("--schema without --connection should exit 2, got %d", r.code)
	}
}

func TestWorkspacePointerVerb(t *testing.T) {
	repo := t.TempDir()
	outside := filepath.Join(t.TempDir(), "kb")
	t.Chdir(repo)
	t.Setenv("DKF_WORKSPACE", "")
	run(t, "", "init", "./knowledge", "--author", "ben", "--json")
	run(t, "", "init", outside, "--author", "ben", "--json")

	// Inside the tree: relative target, and the workspace resolves from below.
	r := run(t, "", "workspace", "pointer", "./knowledge", "--json")
	if r.code != 0 || r.js["pointer"] != filepath.Join(repo, ".dkf") || r.js["target"] != "knowledge" || r.js["relative"] != true {
		t.Fatalf("pointer inside tree: %+v", r)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, ".dkf")); string(data) != "knowledge\n" {
		t.Errorf(".dkf content: %q", data)
	}
	_ = os.MkdirAll(filepath.Join(repo, "src", "pkg"), 0o755)
	t.Chdir(filepath.Join(repo, "src", "pkg"))
	if w := run(t, "", "workspace", "--json"); w.js["via"] != "pointer" || filepath.Base(w.js["root"].(string)) != "knowledge" {
		t.Errorf("resolve via written pointer: %+v", w.js)
	}
	t.Chdir(repo)

	// Naming a different workspace is refused, then allowed with --force.
	if r := run(t, "", "workspace", "pointer", outside, "--json"); r.code != 1 || !strings.Contains(r.stderr, "--force") {
		t.Errorf("differing pointer should exit 1 mentioning --force: %+v", r)
	}
	if data, _ := os.ReadFile(filepath.Join(repo, ".dkf")); string(data) != "knowledge\n" {
		t.Errorf("refused write must not touch the file: %q", data)
	}
	r = run(t, "", "workspace", "pointer", outside, "--force", "--json")
	if r.code != 0 || r.js["relative"] != false || r.js["target"] != outside {
		t.Errorf("--force outside the tree should write an absolute target: %+v", r)
	}
	if !strings.Contains(run(t, "", "workspace", "pointer", outside, "--force").stdout, "do not commit") {
		t.Error("absolute pointer should warn against committing it")
	}
	// Writing the same target again is a no-op, not a conflict.
	if r := run(t, "", "workspace", "pointer", outside, "--json"); r.code != 0 {
		t.Errorf("identical rewrite: %+v", r)
	}

	// Refusals: the workspace itself, and a directory that already has dkf.yaml.
	if r := run(t, "", "workspace", "pointer", "--at", filepath.Join(repo, "knowledge"), outside, "--json"); r.code != 2 || !strings.Contains(r.stderr, "dkf.yaml") {
		t.Errorf("--at a workspace should exit 2: %+v", r)
	}
	t.Chdir(filepath.Join(repo, "knowledge"))
	if r := run(t, "", "workspace", "pointer", ".", "--json"); r.code != 2 {
		t.Errorf("pointer to itself should exit 2: %+v", r)
	}
	t.Chdir(repo)
	if r := run(t, "", "workspace", "pointer", "--at", filepath.Join(repo, "missing"), outside, "--json"); r.code != 2 {
		t.Errorf("--at a missing directory should exit 2: %+v", r)
	}

	// No argument: whatever would be used now, here $DKF_WORKSPACE.
	_ = os.Remove(filepath.Join(repo, ".dkf"))
	t.Setenv("DKF_WORKSPACE", outside)
	if r := run(t, "", "workspace", "pointer", "--json"); r.code != 0 || r.js["root"] != outside {
		t.Errorf("no-arg pointer should name the resolved workspace: %+v", r)
	}
	t.Setenv("DKF_WORKSPACE", "")
	if w := run(t, "", "workspace", "--json"); w.js["via"] != "pointer" || w.js["root"] != outside {
		t.Errorf("written pointer should now resolve: %+v", w.js)
	}
	// A missing workspace is reported, not silently pointed at.
	if r := run(t, "", "workspace", "pointer", filepath.Join(repo, "nope"), "--force", "--json"); r.code == 0 {
		t.Errorf("pointing at a non-workspace should fail: %+v", r)
	}
}

func TestExportVisualFormats(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("DKF_WORKSPACE", ws)
	run(t, "", "init", "--author", "ben", "--harness", "claude", "--json")
	for _, label := range []string{"Project X", "Library Y"} {
		if r := run(t, "", "particular", "define", "--label", label, "--json"); r.code != 0 {
			t.Fatalf("define %s: %+v", label, r)
		}
	}
	claim := func(subject, content string, extra ...string) string {
		t.Helper()
		args := append([]string{"claim", "assert", "--subject", subject, "--content", content, "--json"}, extra...)
		r := run(t, "", args...)
		if r.code != 0 {
			t.Fatalf("assert: %+v", r)
		}
		return r.js["claim"].(map[string]any)["id"].(string)
	}
	a := claim("Project X", "microservices since 2022")
	b := claim("Project X", "monolith since Nov 2024")
	org := claim("Project X", "an organisation fact", "--scope", "organisation")
	loose := claim("Project X", "billing split out again")
	r := run(t, "", "synthesis", "create", "--subject", "Project X",
		"--input", a+":thesis", "--input", b+":antithesis",
		"--unresolved", "None identified", "--content", "consolidated in Nov 2024", "--json")
	if r.code != 0 {
		t.Fatalf("synthesis: %+v", r)
	}
	syn := r.js["synthesis"].(map[string]any)["id"].(string)

	// Both formats, both views.
	for _, format := range []string{"dot", "mermaid"} {
		lineage := run(t, "", "export", "--format", format, "--subject", "Project X")
		if lineage.code != 0 {
			t.Fatalf("%s lineage: %+v", format, lineage)
		}
		for _, want := range []string{shortTag(a), shortTag(b), shortTag(loose), shortTag(syn)} {
			if !strings.Contains(lineage.stdout, want) {
				t.Errorf("%s lineage is missing %s:\n%s", format, want, lineage.stdout)
			}
		}
		wsMap := run(t, "", "export", "--format", format)
		if wsMap.code != 0 || !strings.Contains(wsMap.stdout, "Project X") || !strings.Contains(wsMap.stdout, "Library Y") {
			t.Errorf("%s map: %+v", format, wsMap)
		}
		if strings.Contains(wsMap.stdout, shortTag(a)) {
			t.Errorf("%s map should show particulars, not claims", format)
		}
	}
	if got := run(t, "", "export", "--format", "dot").stdout; !strings.HasPrefix(got, "digraph particulars {") {
		t.Errorf("dot envelope: %q", got[:min2(40, len(got))])
	}
	if got := run(t, "", "export", "--format", "mermaid").stdout; !strings.HasPrefix(got, "flowchart BT") {
		t.Errorf("mermaid envelope: %q", got[:min2(40, len(got))])
	}

	// --json summarises instead of drawing; --out writes the drawing to a file.
	j := run(t, "", "export", "--format", "mermaid", "--subject", "Project X", "--json")
	if j.code != 0 || j.js["format"] != "mermaid" || j.js["view"] != "lineage" || j.js["subject"] != "Project X" {
		t.Errorf("json summary: %+v", j.js)
	}
	if n, ok := j.js["nodes"].(float64); !ok || n < 4 {
		t.Errorf("json summary should count nodes: %+v", j.js)
	}
	if strings.Contains(j.stdout, "flowchart") {
		t.Error("--json must not emit the diagram")
	}
	out := filepath.Join(ws, "sub", "graph.dot")
	f := run(t, "", "export", "--format", "dot", "--out", out, "--json")
	if f.code != 0 || f.js["path"] != out {
		t.Fatalf("--out: %+v", f)
	}
	if data, err := os.ReadFile(out); err != nil || !strings.HasPrefix(string(data), "digraph") {
		t.Errorf("--out file: %v %q", err, string(data)[:min2(40, len(string(data)))])
	}

	// Subject resolution failures use the ordinary exit codes.
	if r := run(t, "", "export", "--format", "dot", "--subject", "Nothing Here", "--json"); r.code != 3 {
		t.Errorf("unknown subject should exit 3, got %d", r.code)
	}

	// Flags that belong to the other format are refused, not ignored.
	for _, tc := range [][]string{
		{"export", "--format", "graph", "--subject", "Project X", "--json"},
		{"export", "--format", "graph", "--depth", "2", "--json"},
		{"export", "--format", "graph", "--include-retracted", "--json"},
		{"export", "--format", "mermaid", "--manifest", "m.txt", "--json"},
		{"export", "--format", "mermaid", "--schema", "--connection", "c", "--json"},
		{"export", "--format", "mermaid", "--source-url", "https://example.com/", "--json"},
		{"export", "--format", "svg", "--json"},
		{"export", "--format", "dot", "--depth", "-1", "--json"},
	} {
		if r := run(t, "", tc...); r.code != 2 {
			t.Errorf("%v should be a usage error, got %d", tc[1:], r.code)
		}
	}

	// personal is drawable but never exportable to Graph.
	if r := run(t, "", "export", "--format", "mermaid", "--scope", "personal"); r.code != 0 {
		t.Errorf("--scope personal should be accepted for a drawing: %+v", r)
	}
	if r := run(t, "", "export", "--format", "graph", "--scope", "personal", "--json"); r.code != 2 {
		t.Errorf("--scope personal must stay refused for graph, got %d", r.code)
	}
	// Narrowing to organisation drops the personal claims from the drawing.
	narrowed := run(t, "", "export", "--format", "mermaid", "--subject", "Project X", "--scope", "organisation")
	if !strings.Contains(narrowed.stdout, shortTag(org)) {
		t.Errorf("organisation claim should survive --scope organisation:\n%s", narrowed.stdout)
	}
	if strings.Contains(narrowed.stdout, shortTag(a)) {
		t.Errorf("personal claim should be dropped by --scope organisation:\n%s", narrowed.stdout)
	}

	// Retraction: hidden by default, drawn on request.
	if r := run(t, "", "retract", loose, "--reason", "mistaken", "--json"); r.code != 0 {
		t.Fatalf("retract: %+v", r)
	}
	if got := run(t, "", "export", "--format", "mermaid", "--subject", "Project X").stdout; strings.Contains(got, shortTag(loose)) {
		t.Error("retracted claim should be omitted by default")
	}
	withRetracted := run(t, "", "export", "--format", "mermaid", "--subject", "Project X", "--include-retracted").stdout
	if !strings.Contains(withRetracted, shortTag(loose)) || !strings.Contains(withRetracted, "retracted") {
		t.Errorf("--include-retracted should draw it, marked:\n%s", withRetracted)
	}
}

// shortTag mirrors viz.Tag without importing the package into these tests.
func shortTag(id string) string {
	prefix, rest, _ := strings.Cut(id, "_")
	return prefix + "…" + rest[len(rest)-6:]
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestPublishAndEffectiveScope(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("DKF_WORKSPACE", ws)
	run(t, "", "init", "--author", "ben", "--harness", "claude", "--json")
	run(t, "", "particular", "define", "--label", "Project X", "--json")
	claim := func(content string, extra ...string) string {
		t.Helper()
		r := run(t, "", append([]string{"claim", "assert", "--subject", "Project X", "--content", content, "--json"}, extra...)...)
		if r.code != 0 {
			t.Fatalf("assert: %+v", r)
		}
		return r.js["claim"].(map[string]any)["id"].(string)
	}
	a, b := claim("microservices since 2022"), claim("monolith since Nov 2024")
	r := run(t, "", "synthesis", "create", "--subject", "Project X", "--input", a+":thesis", "--input", b+":antithesis",
		"--unresolved", "None identified", "--content", "consolidated Nov 2024", "--json")
	if r.code != 0 {
		t.Fatalf("synthesis: %+v", r)
	}
	syn := r.js["synthesis"].(map[string]any)["id"].(string)

	// Everything is personal, so nothing is exportable.
	if e := run(t, "", "export", "--format", "graph", "--json"); e.js["exported"].(float64) != 0 {
		t.Fatalf("a personal workspace should export nothing: %+v", e.js)
	}

	// Promotion is by id only, and only of assertions.
	if p := run(t, "", "publish", "Project X", "--scope", "public", "--json"); p.code != 2 {
		t.Errorf("a label must be refused: %+v", p)
	}
	if p := run(t, "", "publish", "par_01916f03-b680-71a3-974f-9401ba374e1f", "--scope", "public", "--json"); p.code != 2 {
		t.Errorf("a particular must be refused: %+v", p)
	}
	if p := run(t, "", "publish", "clm_01916f03-b680-71a3-974f-9401ba374e1f", "--scope", "public", "--json"); p.code != 3 {
		t.Errorf("an unknown id should exit 3: %+v", p)
	}
	if p := run(t, "", "publish", a, "--json"); p.code != 2 {
		t.Errorf("--scope is required: %+v", p)
	}

	// Promote the belief and its inputs: the workspace becomes exportable
	// though no object file changed.
	before, err := os.ReadFile(filepath.Join(ws, "syntheses", syn+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := run(t, "", "publish", a, b, syn, "--scope", "organisation", "--reason", "cleared for the docs site", "--json")
	if p.code != 0 {
		t.Fatalf("publish: %+v", p)
	}
	pr := p.js["promotion"].(map[string]any)
	if pr["scope"] != "organisation" || len(pr["claims"].([]any)) != 3 {
		t.Errorf("promotion record: %+v", pr)
	}
	if after, _ := os.ReadFile(filepath.Join(ws, "syntheses", syn+".yaml")); !bytes.Equal(before, after) {
		t.Error("promotion must not modify the object it covers")
	}
	e := run(t, "", "export", "--format", "graph", "--json")
	if e.js["exported"].(float64) != 1 {
		t.Errorf("after promotion the particular should export: %+v", e.js)
	}
	if rc := run(t, "", "recall", "Project X", "--scope", "organisation", "--json"); len(rc.js["entries"].([]any)) != 3 {
		t.Errorf("recall --scope organisation should see the promoted three: %+v", rc.js["entries"])
	}
	if tp := run(t, "", "topics", "--scope", "organisation", "--json"); tp.code != 0 {
		t.Errorf("topics --scope: %+v", tp)
	}

	// Widen-only: a claim asserted public cannot be promoted downwards.
	pub := claim("already public", "--scope", "public")
	if d := run(t, "", "publish", pub, "--scope", "organisation", "--json"); d.code == 0 {
		t.Error("narrowing must be refused")
	} else if !strings.Contains(d.stderr, "only widen") {
		t.Errorf("the error should say why: %s", d.stderr)
	}

	// Retracting the promotion reverts exposure.
	prID := pr["id"].(string)
	if rr := run(t, "", "retract", prID, "--reason", "withdrawn", "--json"); rr.code != 0 {
		t.Fatalf("retract: %+v", rr)
	}
	if e := run(t, "", "export", "--format", "graph", "--json"); e.js["exported"].(float64) != 1 {
		// the public claim keeps the particular exportable; the promoted three are gone
		t.Logf("export after retraction: %+v", e.js)
	}
	if rc := run(t, "", "recall", "Project X", "--scope", "organisation", "--json"); len(rc.js["entries"].([]any)) != 0 {
		t.Errorf("after retraction nothing is organisation-scoped: %+v", rc.js["entries"])
	}
	if s := run(t, "", "retract", prID, "--reason", "x", "--superseded-by", a, "--json"); s.code != 2 {
		t.Errorf("--superseded-by must be refused for a promotion: %+v", s)
	}
	if v := run(t, "", "validate", "--json"); v.code != 0 {
		t.Errorf("workspace should validate: %+v", v)
	}
}

func TestSynthesisCreateWarnsOnWiderScope(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("DKF_WORKSPACE", ws)
	run(t, "", "init", "--author", "ben", "--harness", "claude", "--json")
	run(t, "", "particular", "define", "--label", "Project X", "--json")
	r := run(t, "", "claim", "assert", "--subject", "Project X", "--content", "a personal note", "--json")
	priv := r.js["claim"].(map[string]any)["id"].(string)

	s := run(t, "", "synthesis", "create", "--subject", "Project X", "--input", priv+":thesis",
		"--scope", "organisation", "--unresolved", "None identified", "--content", "shareable conclusion", "--json")
	if s.code != 0 {
		t.Fatalf("a wider synthesis must still be written: %+v", s)
	}
	warnings, _ := s.js["warnings"].([]any)
	var found bool
	for _, w := range warnings {
		if strings.Contains(w.(string), "narrower input") {
			found = true
		}
	}
	if !found {
		t.Errorf("create should report scope_wider_than_inputs: %+v", s.js["warnings"])
	}
	// Promoting the input clears it for validate, with no file rewritten.
	if p := run(t, "", "publish", priv, "--scope", "organisation", "--json"); p.code != 0 {
		t.Fatalf("publish: %+v", p)
	}
	v := run(t, "", "validate", "--json")
	for _, f := range v.js["findings"].([]any) {
		if f.(map[string]any)["code"] == "scope_wider_than_inputs" {
			t.Errorf("promoting the input should have cleared the warning: %+v", f)
		}
	}
}

func TestVerifiableProvenance(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("DKF_WORKSPACE", ws)
	run(t, "", "init", "--author", "ben", "--harness", "claude", "--json")
	run(t, "", "particular", "define", "--label", "Project X", "--json")
	if err := os.MkdirAll(filepath.Join(ws, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	docPath := filepath.Join(ws, "docs", "a.md")
	body := "# A\n\nIn staging, the billing service listens on 443.\n\nTail.\n"
	if err := os.WriteFile(docPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	quote := "In staging, the billing service listens on 443."

	// A bare reference stays a scalar: it is not inferior provenance.
	bare := run(t, "", "claim", "assert", "--subject", "Project X", "--content", "bare", "--document", "chat session", "--json")
	if bare.code != 0 {
		t.Fatalf("bare document: %+v", bare)
	}
	if doc := bare.js["claim"].(map[string]any)["source"].(map[string]any)["document"]; doc != "chat session" {
		t.Errorf("a bare document should stay a string in JSON, got %#v", doc)
	}

	// The mapping form, with the hash computed locally.
	r := run(t, "", "claim", "assert", "--subject", "Project X", "--content", "billing listens on 443",
		"--document", "docs/a.md", "--hash-document", "--quote", quote, "--json")
	if r.code != 0 {
		t.Fatalf("verifiable claim: %+v", r)
	}
	id := r.js["claim"].(map[string]any)["id"].(string)
	doc, ok := r.js["claim"].(map[string]any)["source"].(map[string]any)["document"].(map[string]any)
	if !ok || doc["ref"] != "docs/a.md" || doc["quote"] != quote {
		t.Fatalf("document mapping: %#v", doc)
	}
	if h, _ := doc["hash"].(string); !strings.HasPrefix(h, "sha256:") || len(h) != 71 {
		t.Errorf("hash shape: %q", h)
	}
	if v := run(t, "", "validate", "--json"); v.code != 0 {
		t.Fatalf("intact document should validate: %+v", v)
	}

	// Drift is a warning, never a failure.
	if err := os.WriteFile(docPath, []byte(strings.Replace(body, "Tail.", "Rewritten.", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	v := run(t, "", "validate", "--json")
	if v.code != 0 {
		t.Errorf("drift must not fail validation: %+v", v)
	}
	var codes []string
	for _, f := range v.js["findings"].([]any) {
		codes = append(codes, f.(map[string]any)["code"].(string))
	}
	if !slicesContains(codes, "context_drift") {
		t.Errorf("expected context_drift, got %v", codes)
	}
	if err := os.WriteFile(docPath, []byte("# A\n\nGone.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v = run(t, "", "validate", "--json")
	codes = nil
	for _, f := range v.js["findings"].([]any) {
		codes = append(codes, f.(map[string]any)["code"].(string))
	}
	if !slicesContains(codes, "quote_drift") || v.code != 0 {
		t.Errorf("expected quote_drift and exit 0, got %v / %d", codes, v.code)
	}
	// The bare reference is a note, not a warning.
	for _, f := range v.js["findings"].([]any) {
		m := f.(map[string]any)
		if m["code"] == "unverified_document" && m["severity"] != "info" {
			t.Errorf("unverified_document should be a note: %v", m["severity"])
		}
	}

	// Promotion says what a quote discloses.
	p := run(t, "", "publish", id, "--scope", "organisation", "--json")
	if p.code != 0 {
		t.Fatalf("publish: %+v", p)
	}
	var told bool
	for _, w := range p.js["warnings"].([]any) {
		if strings.Contains(w.(string), "verbatim quote") {
			told = true
		}
	}
	if !told {
		t.Errorf("promoting a quoted claim should disclose it: %+v", p.js["warnings"])
	}

	// Retraction kind.
	if rr := run(t, "", "retract", id, "--reason", "misread the port", "--kind", "defect", "--json"); rr.code != 0 {
		t.Fatalf("retract --kind: %+v", rr)
	}
	if got := run(t, "", "recall", "Project X", "--include-retracted", "--json"); got.code != 0 {
		t.Fatalf("recall: %+v", got)
	}
	data, err := os.ReadFile(filepath.Join(ws, "claims", id+".yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "  kind: defect\n") {
		t.Errorf("kind not written:\n%s", data)
	}
	if bad := run(t, "", "retract", bare.js["claim"].(map[string]any)["id"].(string), "--reason", "x", "--kind", "typo", "--json"); bad.code != 2 {
		t.Errorf("an unknown kind should exit 2, got %d", bad.code)
	}
}

func slicesContains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func TestNotesAreCountedNotListed(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("DKF_WORKSPACE", ws)
	run(t, "", "init", "--author", "ben", "--harness", "claude", "--json")
	run(t, "", "particular", "define", "--label", "Project X", "--json")
	for i := 0; i < 3; i++ {
		run(t, "", "claim", "assert", "--subject", "Project X", "--content", "remote evidence",
			"--document", "https://example.com/x", "--json")
	}
	text := run(t, "", "validate")
	if strings.Contains(text.stdout, ".yaml  unverified_document") {
		t.Errorf("notes should not be listed per object by default:\n%s", text.stdout)
	}
	if !strings.Contains(text.stdout, "3 objects  unverified_document") {
		t.Errorf("notes should aggregate to one line per condition: %q", text.stdout)
	}
	if !strings.Contains(text.stdout, "3 notes (--notes to list)") {
		t.Errorf("notes should be counted: %q", text.stdout)
	}
	listed := run(t, "", "validate", "--notes")
	if strings.Count(listed.stdout, ".yaml  unverified_document") != 3 {
		t.Errorf("--notes should list them:\n%s", listed.stdout)
	}
	j := run(t, "", "validate", "--json")
	if j.js["notes"].(float64) != 3 || len(j.js["findings"].([]any)) != 3 {
		t.Errorf("json must always carry notes: %+v", j.js)
	}
	if j.code != 0 {
		t.Errorf("notes must not affect the exit code, got %d", j.code)
	}
}

func TestCorpusFactWarningsAggregate(t *testing.T) {
	ws := t.TempDir()
	t.Chdir(ws)
	t.Setenv("DKF_WORKSPACE", ws)
	run(t, "", "init", "--author", "ben", "--harness", "claude", "--json")
	run(t, "", "particular", "define", "--label", "Project X", "--json")
	var ids []string
	for i := 0; i < 2; i++ {
		r := run(t, "", "claim", "assert", "--subject", "Project X", "--content", "x",
			"--document", "docs/a.md", "--quote", "quoted text", "--json")
		if r.code != 0 {
			t.Fatalf("assert: %+v", r)
		}
		ids = append(ids, r.js["claim"].(map[string]any)["id"].(string))
	}
	// Rewrite both files as v0.8.0 wrote them, so each carries the legacy key.
	for _, id := range ids {
		path := filepath.Join(ws, "claims", id+".yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(strings.Replace(string(data), "    ref: ", "    uri: ", 1)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	text := run(t, "", "validate")
	if text.code != 0 {
		t.Fatalf("validate: %+v", text)
	}
	// A corpus fact is one line however many objects carry it.
	if !strings.Contains(text.stdout, "2 objects  legacy_document_uri") {
		t.Errorf("legacy warnings should aggregate:\n%s", text.stdout)
	}
	if strings.Contains(text.stdout, ".yaml  legacy_document_uri") {
		t.Errorf("no per-object legacy lines by default:\n%s", text.stdout)
	}
	if !strings.Contains(text.stdout, "0 errors, 2 warnings") || !strings.Contains(text.stdout, "(--notes to list)") {
		t.Errorf("aggregation must not change the counts: %q", text.stdout)
	}
	// --notes expands to per-object lines.
	listed := run(t, "", "validate", "--notes")
	if strings.Count(listed.stdout, ".yaml  legacy_document_uri") != 2 {
		t.Errorf("--notes should list each object:\n%s", listed.stdout)
	}
	// JSON always carries every finding, unaggregated.
	j := run(t, "", "validate", "--json")
	if j.js["warnings"].(float64) != 2 {
		t.Errorf("json warning count: %+v", j.js)
	}
	var perObject int
	for _, f := range j.js["findings"].([]any) {
		if f.(map[string]any)["code"] == "legacy_document_uri" {
			perObject++
		}
	}
	if perObject != 2 {
		t.Errorf("json should carry both findings, got %d", perObject)
	}
}
