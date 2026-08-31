## Why

DKF's first three post-v0.1 changes — [`source-as-particular`](https://github.com/nodelogicau/particulars/tree/main/openspec/changes/archive/2026-08-31-source-as-particular), [`attribution-review-round`](https://github.com/nodelogicau/particulars/tree/main/openspec/changes/archive/2026-08-31-attribution-review-round), and [`index-drift-retraction`](https://github.com/nodelogicau/particulars/tree/main/openspec/changes/archive/2026-09-01-index-drift-retraction) — make `source.author` a particular reference, add `document.author` for reported speech, define the asserted-by and reported-from relations over merge classes, and settle the writer refusals, aggregate reporting, and index drift tolerance. The CLI review that shaped them is [#7](https://github.com/nodelogicau/particulars-cli/issues/7); every question it raised is resolved upstream, so the implementation is unblocked in full. The payoff is retroactive: the dogfood workspace's 172 `author: ben` objects attribute to Ben's existing particular the moment this reader ships.

## What Changes

- **`claim assert --author` accepts a particular reference** — id, URI, or bare name — and the writer resolves it: a defined particular is written as its `uri`; an unknown `par_` id is refused (not found); an explicitly given name matching several particulars is refused with the candidates (usage); a default author (`dkf.yaml`, `$DKF_AUTHOR`, `serve --author`) that is ambiguous or unknown is written unchanged; an unknown URI is written unchanged. Applies to every verb that writes a `source` (assert, synthesis, retract, merge, publish) and to the MCP tools.
- **`--document-author <id|uri|name>`** on `claim assert` (and `source.document.author` over MCP): who produced what was read. Written after `ref` in the document mapping — order `ref`, `author`, `hash`, `quote` — with the same writer resolution.
- **`recall --author <id|uri|label|alias>`** returns non-retracted objects asserted by or reported from that particular's merge class; each entry carries `relations` (`asserted`, `reported`, or both), never empty. Combinable with a subject, `--topic`, `--scope`, `--include-retracted`; usable without a subject.
- **URI resolution goes through merge classes** wherever a particular reference is resolved: a query URI joined by non-retracted merges to a particular's `uri` resolves to that particular.
- **Index entries carry `author` and `document-author`**, mirroring the file as written. A rebuild preserves entry fields it does not recognise (the field-level twin of the unknown-entry-type rule), and the drift check tolerates a presence difference, in either direction, in a MAY field that mirrors an immutable property — `retracted` is compared as present, absent meaning false, so a retraction after the index was committed still fails the check.
- **`validate` reports `author_unresolved` (info) and `author_ambiguous` (warning) as corpus facts**, aggregated per author value with the candidates on the ambiguous line; and `orphan_particular` counts attribution — a particular that anything is asserted by or reported from is not an orphan.
- **MCP**: `knowledge_recall` gains `author`; the `claim_assert` document mapping accepts `ref` (canonical) with `uri` as the legacy alias — fixing an existing defect where only `uri` was read — plus `author`.
- **Skill**: who told you goes in `--document-author`, not the content; `--author` is you or the human you work for; pass it only when it differs from the default. `init` and writes never mint a person's particular (already true; now load-bearing).

Nothing is **BREAKING**: every existing file, index, and call keeps working; a reader that ignores the new fields loses the join and nothing else.

## Capabilities

### New Capabilities
- `source-attribution`: author values as particular references — reader resolution (id, URI through merges, name), writer resolution with its refusals, the asserted-by and reported-from relations over merge classes, and the aggregate author findings.

### Modified Capabilities
- `claims`: `claim assert` gains the `--author` reference forms, their refusals, and `--document-author`.
- `verifiable-provenance`: the document mapping gains optional `author` after `ref`; serialisation order `ref`, `author`, `hash`, `quote`.
- `knowledge-query`: `recall --author` with labelled relations; an author filter needs no subject.
- `particulars`: URI resolution reaches through non-retracted merge records.
- `index`: entries carry `author`/`document-author`; rebuilds preserve unknown entry fields; the drift check's immutability tolerance and the `retracted` rule.
- `validation`: the two aggregate author conditions; `orphan_particular` counts attribution.
- `mcp-server`: `knowledge_recall(author)`; document mapping keyed by `ref` with legacy `uri`; `document.author`.

## Impact

- `internal/dkf` (`Document.Author`, four codecs, `documentNode` order and scalar shortcut), `internal/query` (resolver through merges, author resolution and relations, recall filter, validate findings), `internal/store` (index entry fields, unknown-field preservation, drift masking), `internal/cli` (flags on `claim assert` and `recall`, writer resolution at `resolveSource`, aggregate rendering per author value), `internal/mcp` (`sourceIn`/`documentFrom`, `recallIn`, schema text), `skills/particulars/SKILL.md`, README, `docs/provenance.md`, `docs/mcp.md`.
- Closes the implementation half of [#7](https://github.com/nodelogicau/particulars-cli/issues/7). SPEC-FEEDBACK gains the record of the review round (items raised 2026-08-31, all resolved upstream same day and 2026-09-01).
- Existing workspaces: no file changes; the index gains two fields on next write or rebuild without tripping the drift check on committed indexes that predate them.
