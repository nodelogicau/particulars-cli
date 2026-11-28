package mcp

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// RemoteHarness is source.harness for an HTTP session whose handshake names
// no client: the caller is remote, so its writes must not read as the
// person's own unattributed act.
const RemoteHarness = "http"

// ShutdownGrace bounds how long RunHTTP waits for requests in flight once
// its context is cancelled.
const ShutdownGrace = 30 * time.Second

// IsLoopback reports whether addr's host is one the server may listen on
// without a token: 127.0.0.1, ::1 or localhost. A missing host means every
// interface, so it is not loopback.
func IsLoopback(addr string) (bool, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false, err
	}
	switch strings.ToLower(host) {
	case "127.0.0.1", "::1", "localhost":
		return true, nil
	}
	return false, nil
}

// HTTPHandler serves MCP Streamable HTTP at /mcp — every session served by
// this one server, so all share the workspace and the write lock — and an
// unauthenticated readiness probe at /healthz. Every other path is 404. With
// a non-empty token, /mcp requires `Authorization: Bearer <token>`.
//
// The handler is for a server built with Options.Remote; one built without
// it would record a nameless remote session with no harness.
func (s *Server) HTTPHandler(token string) http.Handler {
	return s.httpHandler(token, nil)
}

func (s *Server) httpHandler(token string, streams context.Context) http.Handler {
	mcpH := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return s.srv }, &sdk.StreamableHTTPOptions{
		// The SDK rejects a loopback request whose Host is not loopback, to
		// stop DNS rebinding reaching a tokenless local server. A token
		// already stops that, and a proxy in front of the server — common in
		// a container — forwards from loopback with the public Host.
		DisableLocalhostProtection: token != "",
	})
	var h http.Handler = mcpH
	if streams != nil {
		h = endStreams(streams, h)
	}
	if token != "" {
		h = bearer(token, h)
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", h)
	mux.HandleFunc("/healthz", s.healthz)
	return mux
}

// healthz reports readiness and the version, and nothing about the workspace.
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "version": s.opts.Version})
}

// bearer refuses a request without the token before it reaches the server.
// Both sides are hashed first, so the comparison takes the same time
// whatever the lengths.
func bearer(token string, next http.Handler) http.Handler {
	want := sha256.Sum256([]byte(token))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := ""
		if scheme, cred, ok := strings.Cut(r.Header.Get("Authorization"), " "); ok && strings.EqualFold(scheme, "Bearer") {
			got = strings.TrimSpace(cred)
		}
		sum := sha256.Sum256([]byte(got))
		if got == "" || subtle.ConstantTimeCompare(sum[:], want[:]) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="particulars"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// endStreams ends GET requests — the standalone streams that carry only
// server-to-client notifications — when shutdown begins. They never finish on
// their own, so without this every shutdown would wait out the full grace.
// Tool calls arrive by POST and are left to finish.
func endStreams(shutdown context.Context, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			ctx, cancel := context.WithCancel(r.Context())
			defer cancel()
			stop := context.AfterFunc(shutdown, cancel)
			defer stop()
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// RunHTTP serves HTTPHandler on ln until ctx is cancelled. It then stops
// accepting connections, lets requests in flight finish for up to
// ShutdownGrace, and returns nil. A mutating call holds the write lock to
// completion, so one that has started finishes before its request does.
func (s *Server) RunHTTP(ctx context.Context, ln net.Listener, token string) error {
	shutdown, beginShutdown := context.WithCancel(context.Background())
	defer beginShutdown()
	hs := &http.Server{Handler: s.httpHandler(token, shutdown), ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- hs.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	beginShutdown()
	sctx, cancel := context.WithTimeout(context.Background(), ShutdownGrace)
	defer cancel()
	if err := hs.Shutdown(sctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	_ = hs.Close()
	if err := <-errc; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
