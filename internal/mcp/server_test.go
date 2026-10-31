package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/particulars-cli/internal/query"
	"github.com/nodelogicau/particulars-cli/internal/store"
)

type harness struct {
	t  *testing.T
	ws *store.Workspace
	cs *sdk.ClientSession
}

func newHarness(t *testing.T, clientName string, author string) *harness {
	t.Helper()
	cfg := store.NewConfig()
	cfg.Defaults.Source.Author = author
	ws, err := store.Init(filepath.Join(t.TempDir(), "kb"), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("DKF_AUTHOR", "")
	t.Setenv("DKF_HARNESS", "")
	t.Setenv("DKF_MODEL", "")
	srv := New(Options{Workspace: ws, Version: "test"})
	st, ct := sdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Run(ctx, st) }()
	client := sdk.NewClient(&sdk.Implementation{Name: clientName, Version: "1"}, nil)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return &harness{t: t, ws: ws, cs: cs}
}

// call invokes a tool and returns (structured, isError, text).
func (h *harness) call(name string, args map[string]any) (map[string]any, bool, string) {
	h.t.Helper()
	res, err := h.cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		h.t.Fatalf("%s: protocol error: %v", name, err)
	}
	text := ""
	if len(res.Content) > 0 {
		if tc, ok := res.Content[0].(*sdk.TextContent); ok {
			text = tc.Text
		}
	}
	var out map[string]any
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		_ = json.Unmarshal(b, &out)
	}
	return out, res.IsError, text
}

func (h *harness) ok(name string, args map[string]any) map[string]any {
	h.t.Helper()
	out, isErr, text := h.call(name, args)
	if isErr {
		h.t.Fatalf("%s returned error: %s", name, text)
	}
	return out
}

func (h *harness) fail(name string, args map[string]any, code string) string {
	h.t.Helper()
	out, isErr, text := h.call(name, args)
	if !isErr {
		h.t.Fatalf("%s should have failed, got %v", name, out)
	}
	if got := out["error"].(map[string]any)["code"]; got != code {
		h.t.Errorf("%s: error code %v, want %s (%s)", name, got, code, text)
	}
	return text
}

func id(m map[string]any, keys ...string) string {
	for _, k := range keys {
		m = m[k].(map[string]any)
	}
	return m["id"].(string)
}

func TestInstructionsAndPrompt(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	ins := h.cs.InitializeResult().Instructions
	if !strings.Contains(ins, h.ws.Root) || !strings.Contains(ins, "Recall **before** you assert") {
		t.Errorf("instructions:\n%s", ins[:200])
	}
	ps, err := h.cs.ListPrompts(context.Background(), nil)
	if err != nil || len(ps.Prompts) != 1 || ps.Prompts[0].Name != PromptName {
		t.Fatalf("prompts: %v %v", ps, err)
	}
	pr, err := h.cs.GetPrompt(context.Background(), &sdk.GetPromptParams{Name: PromptName})
	if err != nil || pr.Messages[0].Content.(*sdk.TextContent).Text != ins {
		t.Error("prompt text should equal instructions")
	}
	tools, _ := h.cs.ListTools(context.Background(), nil)
	names := map[string]*sdk.Tool{}
	for _, tl := range tools.Tools {
		names[tl.Name] = tl
	}
	for _, n := range []string{"particular_define", "particular_resolve", "particular_merge", "claim_assert", "claim_retract", "synthesis_create", "knowledge_recall", "conflict_detect", "lineage_trace", "topics_list", "unresolved_list", "workspace_status"} {
		if names[n] == nil {
			t.Errorf("missing tool %s", n)
		}
	}
	if !names["knowledge_recall"].Annotations.ReadOnlyHint || !names["particular_define"].Annotations.IdempotentHint || !strings.HasPrefix(names["topics_list"].Description, "(particulars extension") {
		t.Error("annotations/labels")
	}
}

func TestSpecToolsRoundTrip(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	// resolve miss → null, not an error
	out, isErr, _ := h.call("particular_resolve", map[string]any{"query": "nothing"})
	if isErr || out["particular"] != nil {
		t.Errorf("resolve miss: %v %v", out, isErr)
	}
	p := h.ok("particular_define", map[string]any{"label": "Project X", "aliases": []string{"px"}})
	if p["created"] != true {
		t.Fatalf("define: %v", p)
	}
	pid := id(p, "particular")
	if r := h.ok("particular_resolve", map[string]any{"query": "PX"}); id(r, "particular") != pid {
		t.Error("resolve by alias")
	}
	a := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Project X", "content": "Uses Postgres", "source": map[string]any{"document": "db.md"}, "context": map[string]any{"topics": []string{"db"}}, "confidence": 0.9})
	aid := id(a, "claim")
	if a["claim"].(map[string]any)["source"].(map[string]any)["author"] != "ben" {
		t.Errorf("author default: %v", a)
	}
	b := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": pid, "content": "Uses MySQL"})
	bid := id(b, "claim")
	rc := h.ok("knowledge_recall", map[string]any{"particular_id": "Project X"})
	if rc["count"].(float64) != 2 || rc["entries"].([]any)[0].(map[string]any)["unsynthesised"] != true {
		t.Errorf("recall: %v", rc)
	}
	s := h.ok("synthesis_create", map[string]any{"particular_id": "Project X", "content": "Postgres since 2025", "inputs": []map[string]any{{"id": aid, "role": "thesis"}, {"id": bid, "role": "antithesis", "weight": "qualifying"}}, "unresolved": "None identified", "source": map[string]any{"harness": "claude"}})
	sid := id(s, "synthesis")
	if s["synthesis"].(map[string]any)["source"].(map[string]any)["harness"] != "claude" || len(s["warnings"].([]any)) != 0 {
		t.Errorf("synthesis: %v", s)
	}
	rc = h.ok("knowledge_recall", map[string]any{"particular_id": "Project X"})
	last := rc["entries"].([]any)[2].(map[string]any)
	if last["id"] != sid || last["current"] != true {
		t.Errorf("current after synthesis: %v", last)
	}
	cf := h.ok("conflict_detect", map[string]any{"particular_id": pid})
	if cf["count"].(float64) != 0 {
		t.Errorf("no conflicts expected: %v", cf)
	}
	c := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": pid, "content": "C"})
	cid := id(c, "claim")
	cf = h.ok("conflict_detect", map[string]any{"particular_id": pid})
	rep := cf["reports"].([]any)[0].(map[string]any)
	if rep["current"] != sid || rep["unsynthesised"].([]any)[0] != cid {
		t.Errorf("conflicts: %v", rep)
	}
	// claim-set form
	set := h.ok("conflict_detect", map[string]any{"claim_ids": []string{aid, bid, sid, cid}})
	if set["current"] != sid || set["unsynthesised"].([]any)[0] != cid || len(set["set"].([]any)) != 4 {
		t.Errorf("set: %v", set)
	}
	h.fail("conflict_detect", map[string]any{}, "usage")
	h.fail("conflict_detect", map[string]any{"particular_id": pid, "claim_ids": []string{aid}}, "usage")
	// lineage with superseded_by
	h.ok("claim_retract", map[string]any{"claim_id": aid, "reason": "old", "superseded_by": cid})
	ln := h.ok("lineage_trace", map[string]any{"claim_id": sid})
	first := ln["inputs"].([]any)[0].(map[string]any)
	if first["retracted"] != true || first["superseded_by"] != cid {
		t.Errorf("lineage: %v", first)
	}
	cf = h.ok("conflict_detect", map[string]any{"particular_id": pid})
	if stale := cf["reports"].([]any)[0].(map[string]any)["stale"].([]any); len(stale) != 1 || stale[0] != sid {
		t.Errorf("stale: %v", cf)
	}
	// merge + class-aware recall
	h.ok("particular_define", map[string]any{"label": "Project X legacy"})
	h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Project X legacy", "content": "legacy"})
	m := h.ok("particular_merge", map[string]any{"uri_a": "Project X", "uri_b": "Project X legacy", "reason": "renamed"})
	mid := id(m, "merge")
	rc = h.ok("knowledge_recall", map[string]any{"particular_id": pid})
	if rc["class"] == nil || rc["count"].(float64) != 4 {
		t.Errorf("recall across merge: %v", rc["count"])
	}
	h.fail("particular_merge", map[string]any{"uri_a": "Project X", "uri_b": "Project X legacy"}, "usage")
	h.ok("claim_retract", map[string]any{"claim_id": mid, "reason": "undo"})
	if rc := h.ok("knowledge_recall", map[string]any{"particular_id": pid}); rc["class"] != nil {
		t.Error("class should be gone after merge retraction")
	}
	// errors
	h.fail("synthesis_create", map[string]any{"particular_id": pid, "content": "x", "inputs": []map[string]any{{"id": "clm_missing", "role": "thesis"}}, "unresolved": "n"}, "not_found")
	h.ok("particular_define", map[string]any{"label": "Auth Service", "aliases": []string{"auth"}})
	h.ok("particular_define", map[string]any{"label": "Auth Team", "aliases": []string{"auth"}})
	if msg := h.fail("knowledge_recall", map[string]any{"particular_id": "auth"}, "usage"); !strings.Contains(msg, "par_") {
		t.Errorf("ambiguity message should list ids: %s", msg)
	}
	tp := h.ok("topics_list", map[string]any{})
	if tp["count"].(float64) != 0 { // the only tagged claim (aid) was retracted above
		t.Errorf("topics: %v", tp)
	}
	if ul := h.ok("unresolved_list", map[string]any{}); ul["count"].(float64) != 0 {
		t.Errorf("unresolved_list should hide None identified: %+v", ul)
	}
	if ul := h.ok("unresolved_list", map[string]any{"include_none": true}); ul["count"].(float64) != 1 || ul["entries"].([]any)[0].(map[string]any)["synthesis"] != sid {
		t.Errorf("unresolved_list include_none: %+v", ul)
	}
	// The data rides in the text too, for clients that show the model only text.
	res, err := h.cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "unresolved_list", Arguments: map[string]any{"include_none": true}})
	if err != nil || len(res.Content) != 2 {
		t.Fatalf("unresolved_list content: %v %v", err, res)
	}
	var fromText map[string]any
	if err := json.Unmarshal([]byte(res.Content[1].(*sdk.TextContent).Text), &fromText); err != nil || res.Content[0].(*sdk.TextContent).Text != "1 unresolved" || fromText["entries"].([]any)[0].(map[string]any)["synthesis"] != sid {
		t.Errorf("unresolved_list text: %v %v", err, res.Content)
	}
	h.fail("unresolved_list", map[string]any{"particular_id": "Nobody"}, "not_found")
	if tp := h.ok("topics_list", map[string]any{"include_retracted": true}); tp["count"].(float64) != 1 {
		t.Errorf("topics incl. retracted: %v", tp)
	}
	st := h.ok("workspace_status", map[string]any{})
	if st["root"] != h.ws.Root || st["counts"].(map[string]any)["merges"].(float64) != 1 || st["validate"].(map[string]any)["errors"].(float64) != 0 {
		t.Errorf("status: %v", st)
	}
	if d, _ := h.ws.CheckIndex(); !d.Clean() {
		t.Errorf("index drift after tool calls: %+v", d)
	}
}

func TestHandshakeProvenance(t *testing.T) {
	h := newHarness(t, "claude-ai", "") // no default author anywhere
	h.ok("particular_define", map[string]any{"label": "P"})
	a := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "P", "content": "agent-only"})
	src := a["claim"].(map[string]any)["source"].(map[string]any)
	if src["harness"] != "claude-ai" || src["author"] != nil {
		t.Errorf("handshake default: %v", src)
	}
	b := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "P", "content": "override", "source": map[string]any{"harness": "other", "author": "ben"}})
	src = b["claim"].(map[string]any)["source"].(map[string]any)
	if src["harness"] != "other" || src["author"] != "ben" {
		t.Errorf("override: %v", src)
	}
	// synthesis harness also defaults from the client
	s := h.ok("synthesis_create", map[string]any{"particular_id": "P", "content": "s", "inputs": []map[string]any{{"id": id(a, "claim"), "role": "thesis"}}, "unresolved": "None identified"})
	if s["synthesis"].(map[string]any)["source"].(map[string]any)["harness"] != "claude-ai" {
		t.Errorf("synthesis harness default: %v", s)
	}
}

func TestParallelAssertsKeepIndexClean(t *testing.T) {
	h := newHarness(t, "t", "ben")
	h.ok("particular_define", map[string]any{"label": "P"})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := h.cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "claim_assert", Arguments: map[string]any{"particular_id": "P", "content": "claim", "evidential": "observed"}})
			if err != nil || res.IsError {
				t.Errorf("assert %d: %v %v", i, err, res)
			}
		}(i)
	}
	wg.Wait()
	entries, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims"))
	if len(entries) != 20 {
		t.Errorf("expected 20 claims, got %d", len(entries))
	}
	if d, _ := h.ws.CheckIndex(); !d.Clean() {
		t.Errorf("index not clean: %+v", d)
	}
}

// TestStdioBinary builds the real binary and speaks to it over stdio, which
// proves stdout carries only the protocol.
func TestStdioBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	bin := filepath.Join(t.TempDir(), "particulars")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "../../cmd/particulars")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	ws, err := store.Init(filepath.Join(t.TempDir(), "kb"), store.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bin, "serve", "--mcp", "--workspace", ws.Root, "--author", "ben")
	client := sdk.NewClient(&sdk.Implementation{Name: "stdio-test", Version: "1"}, nil)
	cs, err := client.Connect(context.Background(), &sdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer func() { _ = cs.Close() }()
	tools, err := cs.ListTools(context.Background(), nil)
	if err != nil || len(tools.Tools) != 13 {
		t.Fatalf("tools: %v %v", err, tools)
	}
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "workspace_status", Arguments: map[string]any{}})
	if err != nil || res.IsError {
		t.Fatalf("status: %v %v", err, res)
	}
	// no workspace → exit 5 and empty stdout
	bad := exec.Command(bin, "serve", "--mcp", "--workspace", t.TempDir())
	out, err := bad.Output()
	if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 5 || len(out) != 0 {
		t.Errorf("no workspace: err=%v stdout=%q", err, out)
	}
}

func TestInstructionsCarryWorkspaceConventions(t *testing.T) {
	mk := func(cfg store.Config) *store.Workspace {
		w, err := store.Init(filepath.Join(t.TempDir(), "kb"), cfg)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	// Default file at the root.
	w := mk(store.NewConfig())
	if err := os.WriteFile(filepath.Join(w.Root, store.ConventionsFile), []byte("Compose tags, never compound."), 0o644); err != nil {
		t.Fatal(err)
	}
	ins := New(Options{Workspace: w, Version: "test"}).Instructions()
	if !strings.Contains(ins, "## Workspace conventions (dkf.md)") || !strings.Contains(ins, "Compose tags, never compound.") {
		t.Errorf("default conventions should follow the skill body:\n%s", ins[len(ins)-300:])
	}
	if strings.Index(ins, "Workspace conventions") < strings.Index(ins, "Recall **before** you assert") {
		t.Error("conventions must come after the skill body")
	}
	// Configured file wins over the default name.
	cfg := store.NewConfig()
	cfg.Workspace.Conventions = "docs/TOPICS.md"
	w2 := mk(cfg)
	if err := os.MkdirAll(filepath.Join(w2.Root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(w2.Root, "docs", "TOPICS.md"), []byte("tag vocabulary"), 0o644); err != nil {
		t.Fatal(err)
	}
	ins2 := New(Options{Workspace: w2, Version: "test"}).Instructions()
	if !strings.Contains(ins2, "(docs/TOPICS.md)") || !strings.Contains(ins2, "tag vocabulary") {
		t.Error("configured conventions should be delivered under their own name")
	}
	// An oversized document is truncated with a note naming the file: at
	// least 16 KiB delivered, cut only on a character boundary — a character
	// straddling the mark is included, never dropped.
	w3 := mk(store.NewConfig())
	big := append(bytes.Repeat([]byte("x"), 16*1024-1), []byte("é")...) // é = 0xC3 0xA9 straddles 16 KiB
	big = append(big, bytes.Repeat([]byte("y"), 4*1024)...)
	if err := os.WriteFile(filepath.Join(w3.Root, store.ConventionsFile), big, 0o644); err != nil {
		t.Fatal(err)
	}
	ins3 := New(Options{Workspace: w3, Version: "test"}).Instructions()
	marker := "[truncated — read dkf.md"
	if !strings.Contains(ins3, marker) {
		t.Fatal("oversized conventions should truncate with a note")
	}
	section := ins3[strings.Index(ins3, "## Workspace conventions (dkf.md)"):strings.Index(ins3, marker)]
	body := section[strings.Index(section, "xxx"):]
	body = strings.TrimRight(body, "\n")
	if !utf8.ValidString(body) || !strings.HasSuffix(body, "é") || len(body) < 16*1024 {
		t.Errorf("cut must land after the straddling character: valid=%v len=%d tail=%q", utf8.ValidString(body), len(body), body[len(body)-4:])
	}
	if len(ins3) > len(ins)+17*1024 {
		t.Errorf("truncation should bound the instructions, got %d bytes", len(ins3))
	}
	// No document, no section; a configured-but-missing file is omitted.
	w4 := mk(store.NewConfig())
	if strings.Contains(New(Options{Workspace: w4, Version: "test"}).Instructions(), "Workspace conventions") {
		t.Error("no conventions document should mean no section")
	}
	cfg5 := store.NewConfig()
	cfg5.Workspace.Conventions = "absent.md"
	w5 := mk(cfg5)
	if strings.Contains(New(Options{Workspace: w5, Version: "test"}).Instructions(), "Workspace conventions") {
		t.Error("a missing configured file is omitted, not partially rendered")
	}
}

func TestConventionsResource(t *testing.T) {
	h := newHarness(t, "res-test", "ben")
	if rs, err := h.cs.ListResources(context.Background(), nil); err != nil || len(rs.Resources) != 0 {
		t.Fatalf("no document, no resource: %v %v", err, rs)
	}
	ws, err := store.Init(filepath.Join(t.TempDir(), "kb"), store.NewConfig())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(ws.Root, store.ConventionsFile)
	if err := os.WriteFile(path, []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := New(Options{Workspace: ws, Version: "test"})
	st, ct := sdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = srv.Run(ctx, st) }()
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "res-test", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	rs, err := cs.ListResources(ctx, nil)
	if err != nil || len(rs.Resources) != 1 {
		t.Fatalf("one resource expected: %v %v", err, rs)
	}
	r := rs.Resources[0]
	if r.Name != "dkf.md" || r.MIMEType != "text/markdown" || !strings.HasPrefix(r.URI, "file:///") || !strings.HasSuffix(r.URI, "/dkf.md") {
		t.Errorf("resource: %+v", r)
	}
	// Read on demand: an edit after startup is visible.
	if err := os.WriteFile(path, []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	rr, err := cs.ReadResource(ctx, &sdk.ReadResourceParams{URI: r.URI})
	if err != nil || len(rr.Contents) != 1 || rr.Contents[0].Text != "second" {
		t.Errorf("read: %v %+v", err, rr)
	}
	if !strings.Contains(srv.Instructions(), "first") {
		t.Error("instructions are a startup snapshot")
	}
}

// claim_assert carries the same assert-time quote warning as the CLI, so the
// structured result stays equal to `claim assert --json`.
func TestClaimAssertQuoteWarning(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	if err := os.WriteFile(filepath.Join(h.ws.Root, "note.md"), []byte("the billing service listens\non 443\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h.ok("particular_define", map[string]any{"label": "Billing"})
	doc := func(quote string) map[string]any {
		return map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443",
			"source": map[string]any{"document": map[string]any{"ref": "note.md", "quote": quote}}}
	}
	if r := h.ok("claim_assert", doc("the billing service listens on 443")); r["warnings"] != nil {
		t.Errorf("a quote across a wrap should not warn: %v", r["warnings"])
	}
	r := h.ok("claim_assert", doc("listens on 8443"))
	if ws, _ := r["warnings"].([]any); len(ws) != 1 || !strings.Contains(ws[0].(string), "does not appear in note.md") {
		t.Errorf("expected a quote warning, got %v", r["warnings"])
	}
	// A blank quote is refused on write, as the CLI refuses --quote "   ".
	before, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims"))
	if _, isErr, text := h.call("claim_assert", doc("\n\t")); !isErr || !strings.Contains(text, "source.document.quote is blank") {
		t.Errorf("a blank quote should be a usage error: %v %q", isErr, text)
	}
	if after, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims")); len(after) != len(before) {
		t.Error("a refused claim must not be written")
	}
}

// The register travels with the instructions, because an MCP client never
// reads the repository (workspace-register).
func TestInstructionsCarryRegister(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	h.ok("particular_define", map[string]any{"label": "Billing Service", "aliases": []string{"billing"}})
	h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing Service",
		"content": "Invoices run nightly", "context": map[string]any{"topics": []string{"invoicing"}}})

	ins := New(Options{Workspace: h.ws, Version: "test"}).Instructions()
	for _, want := range []string{"This workspace's particulars and topics", "Billing Service", "invoicing (1)"} {
		if !strings.Contains(ins, want) {
			t.Errorf("instructions missing %q", want)
		}
	}
	// Order: the skill states the discipline, the register the facts, the
	// human's conventions comment on them and are read last.
	if err := os.WriteFile(filepath.Join(h.ws.Root, "dkf.md"), []byte("# Conventions\n\nLocal rules.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ins = New(Options{Workspace: h.ws, Version: "test"}).Instructions()
	reg, conv := strings.Index(ins, "This workspace's particulars"), strings.Index(ins, "Local rules.")
	if reg < 0 || conv < 0 || reg > conv {
		t.Errorf("register must precede the conventions document: %d vs %d", reg, conv)
	}
}

// knowledge_recall with query is recall by query, not an alias of
// particular_id: grouped per particular, its result equals the one RecallQuery
// produces for `recall --query --json`.
func TestKnowledgeRecallByQuery(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	px := id(h.ok("particular_define", map[string]any{"label": "Project X"}), "particular")
	h.ok("particular_define", map[string]any{"label": "Session"})
	h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Project X", "content": "Uses Postgres"})
	h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Session", "content": "The session timeout is 30 minutes."})

	got, isErr, text := h.call("knowledge_recall", map[string]any{"query": "what is the session timeout?"})
	if isErr || !strings.HasPrefix(text, "1 particulars (0 omitted); top: Session [lexical]") {
		t.Fatalf("recall by query: %v %q", isErr, text)
	}
	g, err := h.ws.Load()
	if err != nil {
		t.Fatal(err)
	}
	want, err := query.RecallQuery(g, "what is the session timeout?", query.RecallOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var wantJS map[string]any
	b, _ := json.Marshal(want)
	_ = json.Unmarshal(b, &wantJS)
	if !reflect.DeepEqual(got, wantJS) {
		t.Errorf("MCP result differs from recall --query --json:\n%v\n%v", got, wantJS)
	}

	r := h.ok("knowledge_recall", map[string]any{"query": "Project X"})
	if p := r["particulars"].([]any)[0].(map[string]any); p["particular"] != px || p["resolution"] != "label" || r["entries"] != nil {
		t.Errorf("a name passed as query: %v", r)
	}
	h.fail("knowledge_recall", map[string]any{"particular_id": "Project X", "query": "timeout"}, "usage")
}

// source.quote and source.hash are shorthand for the document's fields: the
// flat shape a model sends first writes what the nested one does (#16).
func TestSourceShorthand(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	h.ok("particular_define", map[string]any{"label": "Billing"})
	h.ok("particular_define", map[string]any{"label": "Jane", "aliases": []string{"jane"}})
	assert := func(src map[string]any) map[string]any {
		return map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443", "source": src}
	}
	docOf := func(r map[string]any, key string) any {
		return r[key].(map[string]any)["source"].(map[string]any)["document"]
	}
	const hash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	flat := h.ok("claim_assert", assert(map[string]any{"document": "chat session 1", "quote": "listens on 443", "hash": hash}))
	nested := h.ok("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "chat session 1", "quote": "listens on 443", "hash": hash}}))
	if got, want := docOf(flat, "claim"), docOf(nested, "claim"); !reflect.DeepEqual(got, want) {
		t.Errorf("flat %v, nested %v", got, want)
	}

	// The shorthand fills a mapping document's empty field.
	r := h.ok("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "docs/a.md", "author": "jane"}, "quote": "the sentence"}))
	d := docOf(r, "claim").(map[string]any)
	if d["ref"] != "docs/a.md" || d["quote"] != "the sentence" || !strings.HasPrefix(d["author"].(string), "urn:") {
		t.Errorf("mapping + shorthand: %v", d)
	}

	before, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims"))
	for _, c := range []struct {
		src  map[string]any
		want string
	}{
		{map[string]any{"document": map[string]any{"ref": "docs/a.md", "quote": "one"}, "quote": "two"}, "pass quote once"},
		{map[string]any{"document": map[string]any{"ref": "docs/a.md", "hash": hash}, "hash": "sha256:1111111111111111111111111111111111111111111111111111111111111111"}, "pass hash once"},
		{map[string]any{"document": map[string]any{"ref": "docs/a.md", "author": "jane"}, "document_author": "ben"}, "pass document author once"},
		{map[string]any{"document": "docs/a.md", "document_author": "jane", "document-author": "ben"}, "pass document author once"},
		{map[string]any{"quote": "the sentence"}, "source.quote needs source.document"},
		{map[string]any{"document_author": "jane"}, "source.document_author needs source.document"},
		{map[string]any{"document": "docs/a.md", "quote": " \n"}, "source.quote is blank"},
	} {
		if text := h.fail("claim_assert", assert(c.src), "usage"); !strings.Contains(text, c.want) {
			t.Errorf("%v: %q, want %q", c.src, text, c.want)
		}
	}
	if after, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims")); len(after) != len(before) {
		t.Error("a refused claim must not be written")
	}

	// The same value given both ways loses nothing, so it is accepted.
	twice := h.ok("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "chat session 1", "quote": "listens on 443", "hash": hash}, "quote": "listens on 443", "hash": hash}))
	if got, want := docOf(twice, "claim"), docOf(nested, "claim"); !reflect.DeepEqual(got, want) {
		t.Errorf("equal values twice %v, nested %v", got, want)
	}

	// Both spellings of the document author write what the nested form does,
	// resolved to the particular's uri.
	byAuthor := h.ok("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "conversation with Jane", "author": "jane"}}))
	for _, key := range []string{"document_author", "document-author"} {
		flat := h.ok("claim_assert", assert(map[string]any{"document": "conversation with Jane", key: "jane"}))
		if got, want := docOf(flat, "claim"), docOf(byAuthor, "claim"); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %v, nested %v", key, got, want)
		}
	}
	h.ok("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "docs/a.md", "author": "jane"}, "document_author": "jane"}))
	unknown := "par_00000000-0000-7000-8000-000000000000"
	nestedErr := h.fail("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "docs/a.md", "author": unknown}}), "not_found")
	if text := h.fail("claim_assert", assert(map[string]any{"document": "docs/a.md", "document_author": unknown}), "not_found"); text != nestedErr {
		t.Errorf("unknown shorthand author: %q, want the nested form's %q", text, nestedErr)
	}

	// Top-level topics join context.topics, context first, once each.
	tc := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443",
		"topics": []string{"billing", "ports"}, "context": map[string]any{"topics": []string{"ports", "infra"}}})
	if got := tc["claim"].(map[string]any)["context"].(map[string]any)["topics"]; !reflect.DeepEqual(got, []any{"ports", "infra", "billing"}) {
		t.Errorf("topics union: %v", got)
	}
	tc = h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443", "topics": []string{"billing"}})
	if got := tc["claim"].(map[string]any)["context"].(map[string]any)["topics"]; !reflect.DeepEqual(got, []any{"billing"}) {
		t.Errorf("topics shorthand alone: %v", got)
	}
	nestedBlank := h.fail("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443", "context": map[string]any{"topics": []string{""}}}, "invalid")
	if text := h.fail("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443", "topics": []string{""}}, "invalid"); text != nestedBlank {
		t.Errorf("blank shorthand topic: %q, want the nested form's %q", text, nestedBlank)
	}

	// Every tool taking source shares it.
	cid := id(flat, "claim")
	s := h.ok("synthesis_create", map[string]any{"particular_id": "Billing", "content": "443", "unresolved": "None identified",
		"inputs": []map[string]any{{"id": cid, "role": "thesis"}}, "source": map[string]any{"document": "review notes", "quote": "the sentence"}})
	if d := docOf(s, "synthesis").(map[string]any); d["ref"] != "review notes" || d["quote"] != "the sentence" {
		t.Errorf("synthesis shorthand: %v", d)
	}
}

// claim_assert accepts the CLI's flags as top-level fields, folding them into
// source so the nested form's rules apply (#18).
func TestTopLevelSource(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	h.ok("particular_define", map[string]any{"label": "Billing"})
	h.ok("particular_define", map[string]any{"label": "Jane", "aliases": []string{"jane"}})
	assert := func(extra map[string]any) map[string]any {
		m := map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443"}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	docOf := func(r map[string]any) any {
		return r["claim"].(map[string]any)["source"].(map[string]any)["document"]
	}
	nested := h.ok("claim_assert", assert(map[string]any{"source": map[string]any{"document": map[string]any{"ref": "chat session 1", "quote": "listens on 443", "author": "jane"}}}))
	flat := h.ok("claim_assert", assert(map[string]any{"document": "chat session 1", "quote": "listens on 443", "document_author": "jane",
		"scope": "personal", "topics": []string{"ports"}, "timestamp": "2023-05-24T00:36:00Z"}))
	if got, want := docOf(flat), docOf(nested); !reflect.DeepEqual(got, want) {
		t.Errorf("flat CLI shape %v, nested %v", got, want)
	}
	mapping := h.ok("claim_assert", assert(map[string]any{"document": map[string]any{"ref": "chat session 1", "quote": "listens on 443"}, "document-author": "jane"}))
	if got, want := docOf(mapping), docOf(nested); !reflect.DeepEqual(got, want) {
		t.Errorf("top-level mapping %v, nested %v", got, want)
	}
	same := h.ok("claim_assert", assert(map[string]any{"document": "chat session 1", "quote": "listens on 443",
		"source": map[string]any{"document": "chat session 1", "quote": "listens on 443", "document_author": "jane"}}))
	if got, want := docOf(same), docOf(nested); !reflect.DeepEqual(got, want) {
		t.Errorf("same values both ways %v, nested %v", got, want)
	}

	// A string document is a mapping holding only its ref, so it merges with
	// the mapping it names; two partial mappings merge field by field.
	for _, args := range []map[string]any{
		{"document": "chat session 1", "source": map[string]any{"document": map[string]any{"ref": "chat session 1", "quote": "listens on 443", "author": "jane"}}},
		{"document": map[string]any{"ref": "chat session 1", "quote": "listens on 443"}, "source": map[string]any{"document": map[string]any{"uri": "chat session 1", "author": "jane"}}},
	} {
		if got, want := docOf(h.ok("claim_assert", assert(args))), docOf(nested); !reflect.DeepEqual(got, want) {
			t.Errorf("%v: merged %v, nested %v", args, got, want)
		}
	}

	// harness and model fold into source like the document fields.
	hm := h.ok("claim_assert", assert(map[string]any{"harness": "bench", "model": "m1", "source": map[string]any{"harness": "bench"}}))
	if s := hm["claim"].(map[string]any)["source"].(map[string]any); s["harness"] != "bench" || s["model"] != "m1" {
		t.Errorf("top-level harness and model: %v", s)
	}

	before, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims"))
	for _, c := range []struct {
		args map[string]any
		want string
	}{
		{map[string]any{"document": "docs/a.md", "source": map[string]any{"document": "docs/b.md"}}, "pass document.ref once"},
		{map[string]any{"document": "docs/a.md", "source": map[string]any{"document": map[string]any{"ref": "docs/b.md", "quote": "q"}}}, "pass document.ref once"},
		{map[string]any{"document": map[string]any{"ref": "docs/a.md", "quote": "one"}, "source": map[string]any{"document": map[string]any{"ref": "docs/a.md", "quote": "two"}}}, "pass document.quote once"},
		{map[string]any{"harness": "bench", "source": map[string]any{"harness": "other"}}, "pass harness once"},
		{map[string]any{"document": "docs/a.md", "source": map[string]any{"document": map[string]any{"ref": "docs/a.md", "page": 3}}}, `no field "page"`},
		{map[string]any{"document": "docs/a.md", "quote": "one", "source": map[string]any{"quote": "two"}}, "pass quote once"},
		{map[string]any{"quote": "listens on 443"}, "source.quote needs source.document"},
	} {
		if text := h.fail("claim_assert", assert(c.args), "usage"); !strings.Contains(text, c.want) {
			t.Errorf("%v: %q, want %q", c.args, text, c.want)
		}
	}
	if after, _ := os.ReadDir(filepath.Join(h.ws.Root, "claims")); len(after) != len(before) {
		t.Error("a refused claim must not be written")
	}

	tools, err := h.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tools.Tools {
		if tl.Name != "claim_assert" {
			continue
		}
		b, _ := json.Marshal(tl.InputSchema)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		doc, _ := m["properties"].(map[string]any)["document"].(map[string]any)
		if branches, _ := doc["anyOf"].([]any); len(branches) != 2 {
			t.Errorf("top-level document should be string | object, got %v", doc)
		}
	}
}

// synthesis_create shares claim_assert's top-level shorthand, so a flattened
// synthesis writes what the nested one does.
func TestFlatSynthesis(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	h.ok("particular_define", map[string]any{"label": "Billing"})
	cid := id(h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443"}), "claim")
	base := func(extra map[string]any) map[string]any {
		m := map[string]any{"particular_id": "Billing", "content": "443", "unresolved": "None identified", "inputs": []map[string]any{{"id": cid, "role": "thesis"}}}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	strip := func(r map[string]any) map[string]any {
		s := r["synthesis"].(map[string]any)
		return map[string]any{"source": s["source"], "context": s["context"]}
	}
	nested := strip(h.ok("synthesis_create", base(map[string]any{
		"source":  map[string]any{"harness": "bench", "model": "m1", "document": map[string]any{"ref": "review notes", "quote": "the sentence"}},
		"context": map[string]any{"scope": "organisation", "topics": []string{"goals", "ports"}}})))
	flat := strip(h.ok("synthesis_create", base(map[string]any{
		"harness": "bench", "model": "m1", "document": "review notes", "quote": "the sentence",
		"scope": "organisation", "topics": []string{"goals"}, "context": map[string]any{"topics": []string{"goals", "ports"}}})))
	if !reflect.DeepEqual(flat, nested) {
		t.Errorf("flat synthesis %v, nested %v", flat, nested)
	}
	h.fail("synthesis_create", base(map[string]any{"harness": "bench", "source": map[string]any{"harness": "other"}}), "usage")

	tools, err := h.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tools.Tools {
		if tl.Name != "synthesis_create" {
			continue
		}
		b, _ := json.Marshal(tl.InputSchema)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		props := m["properties"].(map[string]any)
		for _, k := range []string{"scope", "topics", "document", "quote", "hash", "document_author", "harness", "model"} {
			if props[k] == nil {
				t.Errorf("synthesis_create schema lacks top-level %s", k)
			}
		}
		if branches, _ := props["document"].(map[string]any)["anyOf"].([]any); len(branches) != 2 {
			t.Errorf("synthesis_create top-level document should be string | object: %v", props["document"])
		}
	}
}

// Every tool taking source states source.document's two forms in its input
// schema, and the quote shorthand beside it (#16).
func TestSourceDocumentSchema(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	tools, err := h.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	schemas := map[string]map[string]any{}
	for _, tl := range tools.Tools {
		b, _ := json.Marshal(tl.InputSchema)
		var m map[string]any
		_ = json.Unmarshal(b, &m)
		schemas[tl.Name] = m
	}
	for _, name := range []string{"claim_assert", "synthesis_create", "claim_retract", "knowledge_publish", "particular_merge"} {
		src, _ := schemas[name]["properties"].(map[string]any)["source"].(map[string]any)
		props, _ := src["properties"].(map[string]any)
		doc, _ := props["document"].(map[string]any)
		branches, _ := doc["anyOf"].([]any)
		if len(branches) != 2 || branches[0].(map[string]any)["type"] != "string" {
			t.Errorf("%s: source.document should be string | object, got %v", name, doc)
			continue
		}
		obj, _ := branches[1].(map[string]any)["properties"].(map[string]any)
		if obj["ref"] == nil || obj["quote"] == nil {
			t.Errorf("%s: object form should list ref and quote: %v", name, obj)
		}
		if d, _ := doc["description"].(string); !strings.Contains(d, "what was read") {
			t.Errorf("%s: source.document lost its description: %v", name, doc)
		}
		if props["quote"] == nil || props["hash"] == nil || props["document_author"] == nil {
			t.Errorf("%s: source should carry the quote, hash, and document_author shorthand", name)
		}
	}

	// An unknown key is refused by documentFrom, naming it, not by the schema.
	h.ok("particular_define", map[string]any{"label": "Billing"})
	text := h.fail("claim_assert", map[string]any{"evidential": "observed", "particular_id": "Billing", "content": "443",
		"source": map[string]any{"document": map[string]any{"ref": "a.md", "page": "3"}}}, "usage")
	if !strings.Contains(text, `no field "page"`) {
		t.Errorf("unknown document key: %q", text)
	}
}

// The instructions translate the skill's commands into tools before the skill
// is read, so a model does not call `particulars` as a tool (#16).
func TestInstructionsMapCommandsToTools(t *testing.T) {
	h := newHarness(t, "claude-ai", "ben")
	ins := h.cs.InitializeResult().Instructions
	pre, skillAt := strings.Index(ins, "## Using this skill over MCP"), strings.Index(ins, "Recall **before** you assert")
	if pre < 0 || skillAt < 0 || pre > skillAt {
		t.Fatalf("the preamble should precede the skill body: %d, %d", pre, skillAt)
	}
	preamble := ins[pre:skillAt]
	for _, want := range []string{"no `particulars` tool", "| `particulars particular resolve` | `particular_resolve` |", "| `particulars recall` | `knowledge_recall`", "the register is included below"} {
		if !strings.Contains(preamble, want) {
			t.Errorf("preamble lacks %q", want)
		}
	}
	// The preamble teaches the nested form only: naming the shorthand there
	// coincided with models flattening further (#18). The schema still has it.
	for _, unwanted := range []string{"may also be given", "top-level"} {
		if strings.Contains(preamble, unwanted) {
			t.Errorf("preamble should not teach shorthand, but says %q", unwanted)
		}
	}

	// The mapping names exactly the registered tools: adding a tool without a
	// row, or keeping a row for a removed tool, fails here.
	tools, err := h.cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	registered, mapped := map[string]bool{}, map[string]bool{}
	for _, tl := range tools.Tools {
		registered[tl.Name] = true
	}
	for _, m := range cliTools {
		if m.Tool != "" {
			mapped[m.Tool] = true
		}
	}
	if !reflect.DeepEqual(registered, mapped) {
		t.Errorf("registered tools %v, mapped %v", registered, mapped)
	}
}
