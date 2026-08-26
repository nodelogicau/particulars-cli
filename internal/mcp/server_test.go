package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

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
	for _, n := range []string{"particular_define", "particular_resolve", "particular_merge", "claim_assert", "claim_retract", "synthesis_create", "knowledge_recall", "conflict_detect", "lineage_trace", "topics_list", "workspace_status"} {
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
	rc = h.ok("knowledge_recall", map[string]any{"query": "Project X"})
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
	if err != nil || len(tools.Tools) != 12 {
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
