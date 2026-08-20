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
	for _, p := range []string{"dkf.yaml", "particulars", "claims", "syntheses", "index.yaml"} {
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
	// Missing author anywhere.
	dir2 := filepath.Join(t.TempDir(), "kb2")
	if r := run(t, "", "init", dir2, "--json"); r.code != 0 {
		t.Fatal(r.stderr)
	}
	t.Setenv("DKF_WORKSPACE", dir2)
	define(t, "P")
	if r := run(t, "", "claim", "assert", "--subject", "P", "--content", "x", "--json"); r.code != 2 {
		t.Errorf("missing author should exit 2, got %d", r.code)
	}
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
	pb := s["produced-by"].(map[string]any)
	if pb["harness"] != "test" || pb["model"] != "claude-sonnet-4-6" || s["method"] != "reconciliation" {
		t.Errorf("produced-by/method: %+v", s)
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
