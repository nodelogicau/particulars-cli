## 1. Store

- [x] 1.1 Add `store.ErrMintedCollision` and `*store.MintedCollisionError{Label, URI, Existing}`. `Is` matches the sentinel (design D4).
- [x] 1.2 Implement `Workspace.DefineMinted(uri, label, aliases)` in `internal/store/workspace.go` (D1, D2, D5):
  - Look up the class with `MergedURIOwners`, and create when it's empty.
  - Match the label against members' label and aliases using `EqualFold`. Supplied aliases don't count.
  - On several matches, prefer the URI's carrier, otherwise the lowest id.
  - Union-only alias add that never relabels. Write nothing when no alias is new.
  - Refuse with the typed error, naming the carrier, otherwise the lowest id.
- [x] 1.3 Leave `UpsertParticular` unchanged, and update its doc comment to say it is the explicit-URI path.
- [x] 1.4 Store tests for each new-requirement scenario:
  - `C` then `C#` refused, file count unchanged
  - `Project X` then `project x` keeps the label
  - alias match at `…:c-lang`
  - supplied aliases don't steer
  - aliases added, never removed
  - repeat define is byte-identical
  - resolve through a merge to the ORCID particular, no URN file
  - merged class mismatch refused
  - `Café` / `Cafe` refused

## 2. Error mapping

- [x] 2.1 Map `ErrMintedCollision` to code `conflict`, exit 1 in `apperr.Classify`, and add it to `IsDomain`.

## 3. Front ends

- [x] 3.1 `internal/cli/cmd_particular.go`: route minted URIs to `DefineMinted` and explicit `--uri` to `UpsertParticular`. Build the refusal message from the typed error with `--uri` wording. Make the empty-slug message say an explicit `--uri` is required. Update the command's `Long` text, which currently says defining the same label twice hits the same particular.
- [x] 3.2 `internal/mcp/tools.go`: the same routing, with `uri` wording in the refusal and the empty-slug message. Update the `particular_define` description (D6).
- [x] 3.3 CLI tests in `internal/cli/cli_test.go`:
  - `C#` after `C`: exit 1, JSON error code `conflict`, message names id, URI and label
  - `#` and `東京` without `--uri`: exit 2, nothing written
  - `東京` with the Wikidata URI: created
  - `C 東京` mints `…:c` and is refused after `C#`, or resolves after `c`
  - explicit URI beside minted `c`: created, `C` unchanged
  - rename through explicit URI
- [x] 3.4 MCP tests in `internal/mcp/server_test.go`: `particular_define{label: "C#"}` after `C` returns an error result with code `conflict`, and nothing is written. `東京` without `uri` is refused.

## 4. Existing tests

- [x] 4.1 Run `go test ./...` and fix any test that relied on relabelling through a minted define. Switch it to an explicit URI where the rename is the point.

## 5. Skill and docs

- [x] 5.1 Rewrite *Rules for particulars* in `skills/particulars/SKILL.md` (D6). Replace "the same label always hits the same particular", and cover the refusal, symbols and non-Latin labels needing `--uri`, and renaming only through `--uri`.
- [x] 5.2 Build from HEAD and run `particulars skill install --force` to regenerate `.claude/skills/particulars/SKILL.md`. Never copy by hand.
- [x] 5.3 Add a CHANGELOG entry under "Changed", marked breaking: the minted-collision refusal, never relabelling on the minted path, resolving through merges, and closing #32.

## 6. Verify

- [x] 6.1 `go test ./...`, `golangci-lint run` and `openspec validate fix-slug-collisions --strict` all pass.
- [x] 6.2 Manually re-run the #32 repro in a scratch workspace. `define --label C#` after `C` is refused, and `C` is untouched.
- [x] 6.3 Open an upstream issue so that *Matching an existing alias* in `particular-uris` states the particular's URI. Filed as nodelogicau/particulars#65.
