## Why

`particular define --label <label>` without `--uri` mints `<base><slug>`, and define is idempotent on URI. Labels that differ only in punctuation or diacritics (`C` / `C#` / `C++`, `F` / `F#`, `Café` / `Cafe`, `.NET` / `NET`) mint the same URI. So the second define silently relabels the first particular and reports `Updated`. An agent defining a new thing by label reads that as success and asserts claims about the wrong particular (#32). Upstream has settled the rule (nodelogicau/particulars#64, closing #62 and #63, now in `particular-uris` on `main`). This change aligns the CLI and MCP server with it.

## What Changes

- **BREAKING**: `define` without `--uri` (and `particular_define` without `uri`), where the minted URI already names a particular directly or through a merge record, resolves to it only when the supplied label equals, under Unicode case folding, the `label` or an alias of a particular in that merge class. Supplied aliases don't count. Otherwise the call is refused with code `conflict` and exit 1, and nothing is written. The refusal names the existing particular's `id`, `uri` and `label`. It says a different thing needs `--uri`, and the same thing can be referred to by its URI or id.
- **BREAKING**: on that minted path, define never changes the label. It adds any supplied aliases the particular lacks and removes none. Re-defining `project x` after `Project X` keeps `Project X`.
- Minting through a merge: when the minted URI is joined by a merge record to a particular that doesn't carry it, a matching define resolves to that particular instead of creating a second particular at the minted URI.
- An explicit `--uri` / `uri` behaves as before: idempotent on that URI, and it may relabel.
- The empty-slug refusal (already implemented) gains the upstream framing: it precedes any lookup, writes nothing, and covers labels in a script the slug drops (`#`, `東京`). `C 東京` still mints `…:c` and is subject to the collision rule.
- The skill and the `particular_define` description stop promising that "the same label always hits the same particular". They explain the refusal and say that symbols and labels entirely in a non-Latin script need `--uri`.
- Existing workspaces where a collision already relabelled a particular are left as they are. No detection or migration.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `particulars`: *URI minting* says a slug doesn't decide identity, and specifies the empty-slug refusal. *Define is idempotent on URI* is limited to an explicit URI. A new requirement, *A minted URI resolves only to the particular it was minted for*, covers both CLI and MCP define.

The skill body text changes too, but no `skill-distribution` requirement does.

## Impact

- `internal/store/workspace.go`: new store method for minted define. `UpsertParticular` stays the explicit-URI path.
- `internal/cli/cmd_particular.go` and `internal/mcp/tools.go`: both define paths route minted URIs to the new method. The `particular_define` description text changes.
- `skills/particulars/SKILL.md` and the regenerated `.claude/skills/particulars/SKILL.md`.
- Tests in `internal/store`, `internal/cli` and `internal/mcp`.
- Agents and scripts that relied on relabelling through a minted define now get a conflict. They should pass `--uri` (or the id-derived URI) to rename.
