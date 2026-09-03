## 1. Store

- [x] 1.1 `ConventionsFile = "dkf.md"`; `Config.ConventionsPath() (rel, warning string)` with the lexical rule moved out of `Validate`; `Conventions()` treats an invalid key as unset
- [x] 1.2 `Workspace.LegacyConventions() bool` (key unset, no `dkf.md`, `CONVENTIONS.md` present)
- [x] 1.3 Store tests: invalid key opens the workspace and warns, absolute and `../` forms, legacy detection with and without a configured key, `dkf.md` default

## 2. CLI

- [x] 2.1 `workspace`: `conventions_invalid`, `conventions_legacy` in JSON; stderr warning and notice in both modes
- [x] 2.2 `serve --mcp`: stderr warning for an invalid key and the legacy notice at startup
- [x] 2.3 CLI tests for both verbs

## 3. MCP server

- [x] 3.1 Truncation: advance to the next rune start after 16 KiB; test with a multi-byte character straddling the mark (valid UTF-8, ≥ 16 KiB, note present)
- [x] 3.2 Register the conventions document as a resource at startup when readable (`file://` URI, relative name, `text/markdown`), read on demand; tests for listed-and-read and for no document
- [x] 3.3 Instructions test updated to `dkf.md`

## 4. Skill and docs

- [x] 4.1 `skills/particulars/SKILL.md`: `dkf.md` in the retire-tags rule; regenerate the installed copy from a HEAD build with `skill install --force`; `skill install --check` green
- [x] 4.2 `docs/mcp.md`: `dkf.md`, the lenient key, the resource, `AGENTS.md` as a good value for the key, and that instructions are a startup snapshot while the resource is live
- [x] 4.3 `README.md`: `dkf.md`; `.dkf` is blessed by the spec, not an implementation extension
- [x] 4.4 CHANGELOG entry under Unreleased, naming the rename, the migration, and the release in which the legacy notice is removed

## 5. Verification

- [x] 5.1 `go test ./...`, `golangci-lint run`, `openspec validate align-workspace-conventions`
- [x] 5.2 Manual: a workspace with `CONVENTIONS.md` only shows the notice and delivers nothing; renamed to `dkf.md` it is delivered and listed as a resource
- [x] 5.3 Close particulars-cli#8 from the PR
