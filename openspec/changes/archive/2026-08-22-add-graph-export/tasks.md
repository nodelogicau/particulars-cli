## 1. Export Core (`internal/graph`)

- [x] 1.1 Types: `Item{ACL, Properties, Content}`, `ACL`, `Content`, and the schema/connection payload types with the `@odata.type` collection specifiers
- [x] 1.2 `Build(g *store.Graph, opts)` — one item per particular from `query.Recall` and `CurrentSynthesis`; excludes retracted and `personal`; skips particulars with no exportable assertions; ordered by particular id
- [x] 1.3 `Brief(...)` — the rendered `content.value` (label/uri header, belief with id/timestamp/confidence, not-reconciled, supporting claims with evidence and unsynthesised markers, no-synthesis variant, sections omitted when empty)
- [x] 1.4 Properties: title, particularUri, scope (highest of organisation/public), topics, authors, lastModifiedDateTime (ISO 8601), claimCount, openQuestions, currentSynthesis, and url from `--source-url`
- [x] 1.5 `Schema()` — the registration payload, with a unit test asserting Microsoft's constraints (no searchable+refinable, labels imply retrievable, exactMatch implies not searchable)
- [x] 1.6 Unit tests for every scenario in `specs/graph-export`, including byte-identical repeat runs and the personal-withheld cases

## 2. CLI

- [x] 2.1 `internal/cli/cmd_export.go`: `export --format graph` with `--source-url`, `--out`, `--manifest`, `--scope`; `--scope personal` is a usage error; `--json` summary; no network
- [x] 2.2 `--schema` mode with `--connection`, `--name`, `--description`
- [x] 2.3 Register under root; `export` without a recognised `--format` is a usage error naming the formats
- [x] 2.4 In-process CLI tests: NDJSON shape and ordering, manifest contents, scope refusal, determinism, `--out` file, summary JSON

## 3. Sync Workflow and Docs

- [x] 3.1 `docs/examples/graph-sync.yml`: on push to the knowledge repo's default branch — install particulars, export, obtain a token via **GitHub OIDC federated to Entra** (no stored secret), PUT each item, DELETE ids absent from the current manifest using the previous run's artifact, and log when deletions are skipped because the artifact is missing
- [x] 3.2 `docs/graph.md`: what is indexed and what is not (scope rule, retracted never exported), one-time setup (`--schema`, create connection, register schema, Entra app + federated credential with `ExternalItem.ReadWrite.OwnedBy`), the workflow, and the data-movement trade-off against a federated connector
- [x] 3.3 README: `export` in the verb table; a line in the M365/Copilot area pointing at `docs/graph.md`; `docs/mcp.md` cross-reference so the MCP page no longer implies the server is the only route to Copilot-family clients
- [x] 3.4 `CHANGELOG.md` v0.5.0

## 4. Verification and Release

- [x] 4.1 Dry run against the knowledge workspace: export, inspect a brief by eye, confirm no `personal` content and no retracted content appears, confirm determinism
- [x] 4.2 Validate the emitted schema and one item against Microsoft's documented shapes (payload ≤ 30 MB, ISO 8601 datetimes, collection type specifiers present)
- [x] 4.3 Tag `v0.5.0`; verify the released binary produces the same export as the working tree
- [ ] 4.4 **User-side (cannot be closed here):** create the connection and register the schema in a tenant, run the workflow once, and confirm the knowledge appears and cites correctly in Copilot Chat. Record the outcome — including any schema or brief changes it forces — in the knowledge workspace
