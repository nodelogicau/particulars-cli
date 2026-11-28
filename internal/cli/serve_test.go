package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeHTTPUsage(t *testing.T) {
	t.Setenv("DKF_HTTP_TOKEN", "")
	ws := t.TempDir() // no workspace: a usage check must fire before discovery
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"serve", "--http", "127.0.0.1:0"}, "requires --mcp"},
		{[]string{"serve", "--mcp", "--http-token-file", "tok"}, "needs --http"},
		{[]string{"serve", "--mcp", "--http", ":0"}, "token is required off loopback"},
		{[]string{"serve", "--mcp", "--http", "0.0.0.0:0"}, "token is required off loopback"},
		{[]string{"serve", "--mcp", "--http", "8080"}, "want host:port"},
		{[]string{"serve", "--mcp", "--http", "127.0.0.1:0", "--http-token-file", filepath.Join(ws, "missing")}, "--http-token-file"},
	}
	for _, c := range cases {
		r := run(t, "", append(c.args, "--workspace", ws)...)
		if r.code != 2 || !strings.Contains(r.stderr, c.want) {
			t.Errorf("%v: code %d, stderr %q; want 2 and %q", c.args, r.code, r.stderr, c.want)
		}
		if r.stdout != "" {
			t.Errorf("%v: stdout %q", c.args, r.stdout)
		}
	}
	// An empty token file counts as no token, even with the environment set.
	empty := filepath.Join(ws, "empty")
	if err := os.WriteFile(empty, []byte(" \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DKF_HTTP_TOKEN", "from-env")
	if r := run(t, "", "serve", "--mcp", "--http", ":0", "--http-token-file", empty, "--workspace", ws); r.code != 2 {
		t.Errorf("empty token file: code %d, stderr %q", r.code, r.stderr)
	}
}

func TestServeHTTPNoWorkspace(t *testing.T) {
	r := run(t, "", "serve", "--mcp", "--http", "127.0.0.1:0", "--workspace", t.TempDir())
	if r.code != 5 || r.stdout != "" {
		t.Errorf("code %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
}

func TestHTTPTokenPrecedence(t *testing.T) {
	file := filepath.Join(t.TempDir(), "tok")
	if err := os.WriteFile(file, []byte("  from-file\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DKF_HTTP_TOKEN", " from-env ")
	if got, _ := httpToken(file); got != "from-file" {
		t.Errorf("file and env: %q", got)
	}
	if got, _ := httpToken(""); got != "from-env" {
		t.Errorf("env only: %q", got)
	}
	t.Setenv("DKF_HTTP_TOKEN", "  ")
	if got, _ := httpToken(""); got != "" {
		t.Errorf("blank env: %q", got)
	}
}
