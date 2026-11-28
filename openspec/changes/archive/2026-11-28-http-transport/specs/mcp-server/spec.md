## MODIFIED Requirements

### Requirement: Stdio MCP server bound to one workspace
`particulars serve --mcp [--workspace <dir>] [--author] [--harness] [--model] [--http <addr>] [--http-token-file <path>]` SHALL run a Model Context Protocol server bound at startup to exactly one workspace, resolved as every verb resolves it: the flag, then `DKF_WORKSPACE`, then `dkf.yaml`/`.dkf` discovery from the working directory. Without `--http` the server SHALL speak stdio, and SHALL write nothing to stdout except JSON-RPC. With `--http` it SHALL speak Streamable HTTP as *HTTP transport* requires, and SHALL write nothing to stdout. Without a workspace it SHALL exit with code 5 and a message on stderr, before speaking either transport. `serve` without `--mcp` SHALL be a usage error, and so SHALL `--http-token-file` without `--http`.

#### Scenario: Started in a project
- **WHEN** `particulars serve --mcp` is started in a directory whose ancestor has a `.dkf` pointing at a valid workspace
- **THEN** the server initialises over stdio and every tool operates on that workspace

#### Scenario: No workspace
- **WHEN** `particulars serve --mcp` is started with no resolvable workspace
- **THEN** the process exits with code 5 and stdout is empty

#### Scenario: No workspace over HTTP
- **WHEN** `particulars serve --mcp --http 127.0.0.1:8080` is started with no resolvable workspace
- **THEN** the process exits with code 5 without listening

## ADDED Requirements

### Requirement: HTTP transport
With `--http <addr>`, where `addr` is `host:port` and a missing host means every interface, the server SHALL serve MCP Streamable HTTP at the path `/mcp`. Every session SHALL be served by the one server bound to the workspace. Tools, results, errors, instructions, prompts and resources SHALL be identical to those over stdio. Other paths SHALL return 404, except `/healthz`. On startup the server SHALL write to stderr the version, the workspace root, the address it listens on, and whether a token is required.

#### Scenario: The same result over both transports
- **WHEN** one client calls `knowledge_recall{particular_id: "Project X"}` over stdio and another calls it over HTTP, on the same workspace
- **THEN** both results equal `particulars recall "Project X" --json`

#### Scenario: Unknown path
- **WHEN** a request is sent to `/` on a server started with `--http`
- **THEN** the response is 404

### Requirement: HTTP requires a token off loopback
The token SHALL be read from `DKF_HTTP_TOKEN`, or from the file named by `--http-token-file`, whose contents with surrounding whitespace trimmed are the token. The file SHALL win when both are set. An empty token SHALL count as none. When a token is configured, every request to `/mcp` SHALL carry `Authorization: Bearer <token>`. The comparison SHALL be in constant time. A request without the token, or with a different one, SHALL get 401 and SHALL not reach the server. When no token is configured, the server SHALL listen only if `addr`'s host is `127.0.0.1`, `::1` or `localhost`. Otherwise it SHALL exit with a usage error before listening.

#### Scenario: Missing token
- **WHEN** a server started with `DKF_HTTP_TOKEN=s3cret --http :8080` receives an `initialize` request with no `Authorization` header
- **THEN** the response is 401 and no session is created

#### Scenario: Correct token
- **WHEN** the same server receives `initialize` with `Authorization: Bearer s3cret`
- **THEN** a session is created and the tools are listed as over stdio

#### Scenario: Tokenless on all interfaces
- **WHEN** `particulars serve --mcp --http :8080` is started with no token configured
- **THEN** it exits with a usage error saying a token is required off loopback, and does not listen

#### Scenario: Tokenless on loopback
- **WHEN** `particulars serve --mcp --http 127.0.0.1:8080` is started with no token configured
- **THEN** it listens, and serves requests without an `Authorization` header

### Requirement: Health endpoint
`GET /healthz` SHALL return 200 with a JSON body `{"status": "ok", "version": <version>}` once the workspace is loaded and the listener is up. It SHALL not require the token, and SHALL reveal no workspace content, path or configuration.

#### Scenario: Readiness probe
- **WHEN** a container platform sends `GET /healthz` without a token to a server started with a token
- **THEN** the response is 200 with `status: ok` and the binary's version, and nothing else

### Requirement: HTTP sessions share one workspace and one write lock
Concurrent HTTP sessions SHALL share the server-wide lock that serialises mutating tools, so that writes from different sessions never interleave load, write and index update. Each session SHALL take its provenance defaults from its own `initialize`, as *Provenance defaults from the handshake* requires.

#### Scenario: Parallel asserts from two sessions
- **WHEN** two HTTP sessions each issue ten `claim_assert` calls concurrently
- **THEN** twenty claim files exist and `particulars index --check` passes

#### Scenario: Each session's own harness
- **WHEN** a session whose client is named `executive` and a session whose client is named `backend` each assert a claim with no `source`
- **THEN** the claims carry `source.harness: executive` and `source.harness: backend` respectively

### Requirement: A remote session is never the person
Over HTTP, when no harness is supplied by the call, server flags, environment or `dkf.yaml`, and the session's `initialize` carries no `clientInfo.name`, `source.harness` SHALL default to `http`. Over stdio the existing default SHALL be unchanged.

#### Scenario: Nameless remote client
- **WHEN** an HTTP session whose `initialize` has an empty `clientInfo.name` calls `claim_assert` with no `source`, in a workspace with no default harness
- **THEN** the claim is written with `source.harness: http`

### Requirement: Clean shutdown over HTTP
On SIGTERM or SIGINT, a server speaking HTTP SHALL stop accepting new connections. It SHALL let requests in flight finish for up to 30 seconds, then exit 0. A mutating call that has started SHALL either complete, or leave the workspace as it was before the call.

#### Scenario: Terminated during a write
- **WHEN** SIGTERM arrives while a `claim_assert` call is in flight
- **THEN** the call completes, its claim file and index entry are written, and the process exits 0
