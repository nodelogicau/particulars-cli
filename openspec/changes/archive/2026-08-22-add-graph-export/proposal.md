## Why

Microsoft 365 Copilot cannot reach a local stdio MCP server: it runs in Microsoft's cloud with no access to a workspace on disk or in a git checkout. Its own MCP route — [custom federated connectors](https://learn.microsoft.com/en-us/microsoft-365/copilot/connectors/set-up-custom-federated-connectors) — requires a *remote* HTTPS MCP server with OAuth, and is **read-only by contract**, so it costs a hosted service to land in the same capability class as an export. A [synced Copilot connector](https://learn.microsoft.com/en-us/microsoft-365/copilot/extensibility/overview-copilot-connector) reaches the same surfaces (Copilot Chat, Search, Excel, Researcher) with semantic indexing and citations, needs no server at all, and composes with the review model this project already has: an agent proposes, a human merges, and *merged* knowledge becomes available to Copilot.

## What Changes

- New verb `particulars export --format graph` writing Microsoft Graph `externalItem` payloads (NDJSON, one per line) to stdout or `--out`. The CLI **emits**; it never talks to Microsoft. No Graph SDK, no MSAL, no Entra configuration enters this codebase, and the export is offline and unit-testable.
- `export --format graph --schema` emits the connection and schema registration payload, so first-time setup is a command rather than a wiki page.
- **One item per particular**, whose `content` is a rendered brief: the current belief (the current synthesis's content), its `unresolved`, the supporting claims with evidence and confidence, and the claims not yet reconciled. Retracted objects are never exported, so Copilot cannot cite withdrawn knowledge.
- **Scope is the export filter**: `personal` is never exported; `organisation` and `public` are, with an `acl` granting everyone. This is a requirement, not a default.
- `url` on each item points at the current synthesis's file in the source repository (`--source-url`), so a Copilot citation opens the reviewed YAML with its provenance.
- A sample GitHub Actions workflow (`docs/examples/graph-sync.yml`) that, on merge to the knowledge repository's default branch, runs the export and PUTs each item, deleting items that no longer exist — authenticating with **GitHub OIDC federated to Entra**, so no client secret is stored.
- `docs/graph.md`: what is indexed, what is not, the one-time connection setup, the scope rule, and the data-movement trade-off against the federated route.
- Release as **v0.5.0**.

## Capabilities

### New Capabilities
- `graph-export`: the `export --format graph` verb — item shape and identity, the rendered brief, the schema payload, scope filtering and ACLs, deletion semantics, and determinism.

### Modified Capabilities
<!-- none: export is an additional verb; no existing requirement changes -->

## Impact

- New package `internal/graph` (item and schema types, brief rendering); `internal/cli/cmd_export.go`. No new third-party dependencies.
- README verb table and a pointer from `docs/mcp.md` (which currently frames the MCP server as the only way to reach Copilot-family clients).
- Knowledge workspace gains claims about the M365 integration paths once shipped.
- **Verification gap, stated plainly:** we have no M365 tenant wired up here, so this ships unit-tested against the documented API shapes and verified by a dry-run push against a real connection *by the user*. Unlike every other change in this project, "seen working in the product" is not something I can close.
- Not in scope: pushing from the CLI, a hosted/remote MCP server (`serve --http`), federated connectors, declarative agents or API plugins (the write path), per-assertion items, and incremental crawling via Graph's connector crawl APIs.
