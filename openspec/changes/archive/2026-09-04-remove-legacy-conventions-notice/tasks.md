## 1. Removal

- [x] 1.1 Store: remove `LegacyConventionsFile`, `LegacyConventionsNotice`, `LegacyConventions()`; drop the legacy section of `TestConventionsResolution`
- [x] 1.2 CLI: remove `conventions_legacy` and the notice from `workspace`; remove the notice from `serve`; drop the legacy section of `TestWorkspaceReportsConventions`
- [x] 1.3 `docs/mcp.md`: remove the sentence about the notice; CHANGELOG entry under Unreleased

## 2. Verification

- [x] 2.1 `go test ./...`, `golangci-lint run`, `openspec validate remove-legacy-conventions-notice`; no remaining reference to `LegacyConventions` or `conventions_legacy` outside archives and the changelog
