package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/nodelogicau/particulars-cli/internal/store"
)

// httpWorkspace is a fresh workspace with no provenance defaults anywhere.
func httpWorkspace(t *testing.T, author string) *store.Workspace {
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
	return ws
}

// bearerTransport adds a bearer token to every request.
type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// connectHTTP opens a session to base/mcp as a client named name.
func connectHTTP(t *testing.T, base, name, token string) (*sdk.ClientSession, error) {
	t.Helper()
	tr := &sdk.StreamableClientTransport{Endpoint: base + "/mcp"}
	if token != "" {
		tr.HTTPClient = &http.Client{Transport: bearerTransport{token}}
	}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: name, Version: "1"}, nil).Connect(context.Background(), tr, nil)
	if err == nil {
		t.Cleanup(func() { _ = cs.Close() })
	}
	return cs, err
}

func httpServer(t *testing.T, ws *store.Workspace, token string) (*Server, *httptest.Server) {
	t.Helper()
	srv := New(Options{Workspace: ws, Version: "test", Remote: true})
	ts := httptest.NewServer(srv.HTTPHandler(token))
	t.Cleanup(ts.Close)
	return srv, ts
}

func mustCall(t *testing.T, cs *sdk.ClientSession, name string, args map[string]any) map[string]any {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil || res.IsError {
		t.Fatalf("%s: %v %+v", name, err, res)
	}
	b, _ := json.Marshal(res.StructuredContent)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return m
}

func TestHTTPMatchesStdio(t *testing.T) {
	ws := httpWorkspace(t, "ben")
	_, ts := httpServer(t, ws, "")
	hc, err := connectHTTP(t, ts.URL, "remote", "")
	if err != nil {
		t.Fatal(err)
	}
	// The stdio side: the same workspace through a second server, started
	// before any write since instructions are a startup snapshot.
	local := New(Options{Workspace: ws, Version: "test"})
	st, ct := sdk.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() { _ = local.Run(ctx, st) }()
	sc, err := sdk.NewClient(&sdk.Implementation{Name: "local", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sc.Close() })
	mustCall(t, hc, "particular_define", map[string]any{"label": "Project X"})
	mustCall(t, hc, "claim_assert", map[string]any{"evidential": "observed", "particular_id": "Project X", "content": "Uses Postgres 16"})

	ht, err := hc.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	stt, err := sc.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ht.Tools, stt.Tools) {
		t.Error("tools differ between HTTP and stdio")
	}
	if hc.InitializeResult().Instructions != sc.InitializeResult().Instructions {
		t.Error("instructions differ between HTTP and stdio")
	}
	args := map[string]any{"particular_id": "Project X"}
	if h, s := mustCall(t, hc, "knowledge_recall", args), mustCall(t, sc, "knowledge_recall", args); !reflect.DeepEqual(h, s) {
		t.Errorf("recall differs:\nhttp  %v\nstdio %v", h, s)
	}
}

// post sends an initialize request to /mcp with the given Authorization.
func post(t *testing.T, url, auth string) int {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"raw","version":"1"}}}`
	req, _ := http.NewRequest(http.MethodPost, url+"/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode
}

func TestHTTPToken(t *testing.T) {
	ws := httpWorkspace(t, "ben")
	_, ts := httpServer(t, ws, "s3cret")
	for _, auth := range []string{"", "Bearer wrong", "Bearer s3cre", "Basic s3cret", "Bearer "} {
		if code := post(t, ts.URL, auth); code != http.StatusUnauthorized {
			t.Errorf("Authorization %q: got %d, want 401", auth, code)
		}
	}
	if code := post(t, ts.URL, "Bearer s3cret"); code != http.StatusOK {
		t.Errorf("correct token: got %d", code)
	}
	if _, err := connectHTTP(t, ts.URL, "c", "wrong"); err == nil {
		t.Error("connected with a wrong token")
	}
	cs, err := connectHTTP(t, ts.URL, "c", "s3cret")
	if err != nil {
		t.Fatal(err)
	}
	if tools, err := cs.ListTools(context.Background(), nil); err != nil || len(tools.Tools) != 13 {
		t.Errorf("tools: %v %v", err, tools)
	}
}

func TestHTTPTokenlessLoopbackServes(t *testing.T) {
	ws := httpWorkspace(t, "ben")
	_, ts := httpServer(t, ws, "") // httptest listens on 127.0.0.1
	if code := post(t, ts.URL, ""); code != http.StatusOK {
		t.Errorf("tokenless loopback: got %d", code)
	}
}

func TestIsLoopback(t *testing.T) {
	for addr, want := range map[string]bool{
		"127.0.0.1:8080": true, "[::1]:8080": true, "localhost:8080": true, "LOCALHOST:1": true,
		":8080": false, "0.0.0.0:8080": false, "[::]:8080": false, "10.0.0.1:80": false, "example.com:80": false,
	} {
		got, err := IsLoopback(addr)
		if err != nil || got != want {
			t.Errorf("IsLoopback(%q) = %v, %v; want %v", addr, got, err, want)
		}
	}
	if _, err := IsLoopback("8080"); err == nil {
		t.Error("a bare port without a colon was accepted")
	}
}

func TestHealthzAndUnknownPaths(t *testing.T) {
	ws := httpWorkspace(t, "ben")
	_, ts := httpServer(t, ws, "s3cret")
	res, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || !reflect.DeepEqual(body, map[string]any{"status": "ok", "version": "test"}) {
		t.Errorf("healthz: %d %v", res.StatusCode, body)
	}
	for _, p := range []string{"/", "/mcp/", "/healthz/x", "/sse"} {
		r, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		_ = r.Body.Close()
		if r.StatusCode != http.StatusNotFound {
			t.Errorf("%s: got %d, want 404", p, r.StatusCode)
		}
	}
}

func TestHTTPSessionsShareWriteLock(t *testing.T) {
	ws := httpWorkspace(t, "ben")
	_, ts := httpServer(t, ws, "")
	a, err := connectHTTP(t, ts.URL, "a", "")
	if err != nil {
		t.Fatal(err)
	}
	b, err := connectHTTP(t, ts.URL, "b", "")
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, a, "particular_define", map[string]any{"label": "P"})
	var wg sync.WaitGroup
	for _, cs := range []*sdk.ClientSession{a, b} {
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "claim_assert", Arguments: map[string]any{"particular_id": "P", "content": "claim", "evidential": "observed"}})
				if err != nil || res.IsError {
					t.Errorf("assert: %v %v", err, res)
				}
			}()
		}
	}
	wg.Wait()
	if n := objectFiles(filepath.Join(ws.Root, "claims")); n != 20 {
		t.Errorf("expected 20 claims, got %d", n)
	}
	if d, _ := ws.CheckIndex(); !d.Clean() {
		t.Errorf("index not clean: %+v", d)
	}
}

func harnessOf(t *testing.T, cs *sdk.ClientSession) any {
	t.Helper()
	r := mustCall(t, cs, "claim_assert", map[string]any{"evidential": "observed", "particular_id": "P", "content": "c"})
	return r["claim"].(map[string]any)["source"].(map[string]any)["harness"]
}

func TestHTTPSessionHarness(t *testing.T) {
	ws := httpWorkspace(t, "") // no default author or harness
	_, ts := httpServer(t, ws, "")
	exec, err := connectHTTP(t, ts.URL, "executive", "")
	if err != nil {
		t.Fatal(err)
	}
	back, err := connectHTTP(t, ts.URL, "backend", "")
	if err != nil {
		t.Fatal(err)
	}
	anon, err := connectHTTP(t, ts.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, exec, "particular_define", map[string]any{"label": "P"})
	for cs, want := range map[*sdk.ClientSession]string{exec: "executive", back: "backend", anon: RemoteHarness} {
		if got := harnessOf(t, cs); got != want {
			t.Errorf("harness %v, want %s", got, want)
		}
	}
	// An explicit default still wins over the remote default.
	t.Setenv("DKF_HARNESS", "configured")
	if got := harnessOf(t, anon); got != "configured" {
		t.Errorf("DKF_HARNESS: harness %v", got)
	}
	// Over stdio a nameless client still leaves the harness unset.
	t.Setenv("DKF_HARNESS", "")
	h := newHarness(t, "", "ben")
	h.ok("particular_define", map[string]any{"label": "P"})
	r := h.ok("claim_assert", map[string]any{"evidential": "observed", "particular_id": "P", "content": "c"})
	if got := r["claim"].(map[string]any)["source"].(map[string]any)["harness"]; got != nil {
		t.Errorf("stdio nameless harness: %v", got)
	}
}

// TestRunHTTPShutdownFinishesWrite holds the write lock so a claim_assert is
// in flight when shutdown begins, then releases it: the call completes and
// RunHTTP returns nil.
func TestRunHTTPShutdownFinishesWrite(t *testing.T) {
	ws := httpWorkspace(t, "ben")
	srv := New(Options{Workspace: ws, Version: "test", Remote: true})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- srv.RunHTTP(ctx, ln, "") }()
	cs, err := connectHTTP(t, "http://"+ln.Addr().String(), "c", "")
	if err != nil {
		t.Fatal(err)
	}
	mustCall(t, cs, "particular_define", map[string]any{"label": "P"})

	srv.mu.Lock()
	called := make(chan error, 1)
	go func() {
		res, err := cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "claim_assert", Arguments: map[string]any{"particular_id": "P", "content": "in flight", "evidential": "observed"}})
		if err == nil && res.IsError {
			err = io.ErrUnexpectedEOF
		}
		called <- err
	}()
	time.Sleep(200 * time.Millisecond) // the call is now waiting on the lock
	cancel()
	select {
	case err := <-done:
		t.Fatalf("RunHTTP returned with a call in flight: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	srv.mu.Unlock()
	if err := <-called; err != nil {
		t.Errorf("in-flight call: %v", err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("RunHTTP: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("RunHTTP did not return after the call finished")
	}
	if n := objectFiles(filepath.Join(ws.Root, "claims")); n != 1 {
		t.Errorf("expected 1 claim, got %d", n)
	}
	if d, _ := ws.CheckIndex(); !d.Clean() {
		t.Errorf("index not clean: %+v", d)
	}
}

// TestHTTPBinary runs the real binary over HTTP: the startup line names the
// address, stdout stays empty, and SIGTERM with an open session exits 0
// promptly.
func TestHTTPBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("short")
	}
	if runtime.GOOS == "windows" {
		t.Skip("no SIGTERM on windows")
	}
	bin := filepath.Join(t.TempDir(), "particulars")
	if out, err := exec.Command("go", "build", "-o", bin, "../../cmd/particulars").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	ws := httpWorkspace(t, "ben")
	cmd := exec.Command(bin, "serve", "--mcp", "--workspace", ws.Root, "--http", "127.0.0.1:0")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stderr).ReadString('\n')
	if err != nil {
		t.Fatalf("startup line: %v", err)
	}
	for _, want := range []string{ws.Root, "over http at 127.0.0.1:", "no token"} {
		if !strings.Contains(line, want) {
			t.Errorf("startup line %q lacks %q", line, want)
		}
	}
	addr := strings.TrimSuffix(strings.Fields(strings.SplitN(line, " at ", 2)[1])[0], "/mcp,")
	cs, err := connectHTTP(t, "http://"+addr, "c", "")
	if err != nil {
		t.Fatalf("connect %s: %v", addr, err)
	}
	if _, err := cs.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_ = cmd.Process.Signal(syscall.SIGTERM)
	if err := cmd.Wait(); err != nil {
		t.Errorf("exit after SIGTERM: %v", err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("shutdown took %v with an idle session", d)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout not empty: %q", stdout.String())
	}
}
