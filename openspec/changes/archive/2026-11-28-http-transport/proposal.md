## Why

`serve --mcp` speaks only stdio, so a workspace can be reached only by a client that starts the binary itself on the same machine. The executive needs to reach memory from a hosted runtime: a Cloudflare Agent can act as an MCP client only over Streamable HTTP or SSE, and other hosts have the same limitation. A spike on 2026-11-22 (`artifacts-spike`) confirmed that a container can keep a workspace durable in a git remote, which leaves the transport as the missing piece.

## What Changes

- `serve --mcp --http <addr>` serves the same server over MCP Streamable HTTP at `/mcp`, bound to one workspace as today. Without `--http`, `serve --mcp` speaks stdio, unchanged.
- HTTP requests need a bearer token, read from `DKF_HTTP_TOKEN` or `--http-token-file`, never from a flag value. Without a token, the server refuses to listen on anything but a loopback address.
- `GET /healthz` reports that the workspace is loaded and the server is ready, without needing the token, for container and orchestrator readiness checks.
- Every HTTP session shares the one workspace and the server-wide write lock. Each session takes its provenance defaults from its own handshake.
- A remote session is never the person: a session whose handshake gives no client name defaults `source.harness` to `http`.
- On SIGTERM or SIGINT the server stops accepting requests, lets calls in flight finish, and exits 0.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`: *Stdio MCP server bound to one workspace* becomes a transport-neutral requirement with stdio as the default. New requirements cover the HTTP transport, the token, the health endpoint, sessions sharing the write lock, the harness for nameless remote sessions, and shutdown.

## Impact

- `internal/cli/cmd_serve.go`: the `--http` and `--http-token-file` flags, the HTTP server, the health handler, and signal-driven shutdown.
- `internal/mcp`: the server already serialises writes on one mutex and reads `ClientInfo` per request. HTTP sessions use it as it is, apart from the nameless-session harness default.
- `github.com/modelcontextprotocol/go-sdk` v1.7.0 already provides `StreamableHTTPHandler`. No new dependency.
- README and `serve --help`. The MCP bundle and plugin are unaffected: they start the binary over stdio.
- The same change is proposed for `intentions-cli` and `episodes-cli`, so the three servers share flags, endpoints and token handling.
