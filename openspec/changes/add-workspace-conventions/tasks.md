## 1. Store

- [x] 1.1 `workspace.conventions` in `WorkspaceConfig` with relative-inside-the-workspace validation
- [x] 1.2 `Workspace.Conventions()` resolving the configured file or the `CONVENTIONS.md` default; missing default is silence, missing configured file is an error carrying the path

## 2. MCP

- [x] 2.1 `instructions()` appends the document under `## Workspace conventions (<file>)`, capped at 16 KiB with a truncation note
- [x] 2.2 `serve` warns on stderr when a configured file cannot be read, and omits it

## 3. CLI

- [x] 3.1 `workspace` reports `conventions` / `conventions_missing` in JSON and text

## 4. Docs and skill

- [x] 4.1 docs/mcp.md ("What the model is told") and README Configuration
- [x] 4.2 Skill topics bullet points at the conventions file; regenerate the installed copy from a HEAD build

## 5. Verification

- [x] 5.1 Tests: config validation, Conventions() resolution, instructions with/without/configured/truncated, workspace verb JSON
- [x] 5.2 `go test ./...`, `golangci-lint`, `openspec validate add-workspace-conventions`
