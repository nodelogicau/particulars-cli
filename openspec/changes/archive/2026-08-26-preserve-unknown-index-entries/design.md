## Context

The decode currently unmarshals `entries` straight into `[]Entry`, so an unknown type is squeezed into the known shape — extra fields dropped — and a rebuild then omits it entirely. "Preserve unchanged" therefore cannot be retrofitted onto `Entry`; the unknown rows have to be held in a form that never passed through the struct.

## Goals / Non-Goals

**Goals:** byte-faithful preservation through read → rebuild → write; drift checks blind to unknown types; zero behaviour change for an index containing only known types.

**Non-Goals:** interpreting unknown entries in any way; preserving unknown *fields on known types* beyond what the spec's MAY already covers (our encoder writes what it knows, which is the writer's prerogative — preservation is about entries, where dropping is data loss rather than normalisation).

## Decisions

### Unknown entries are raw YAML nodes, held beside `Entries`, not in it

`Index` gains `Unknown []yaml.Node`, populated at decode for any entry whose `type` is not one of the five known values. Every existing consumer iterates `Entries` and so ignores them without knowing they exist — the spec's MUST-ignore enforced by structure rather than by discipline at each call site. Encode merges both sets sorted by `id`, which is the canonical order the file already uses; the nodes re-serialise with the document's own 2-space style, so an untouched index stays byte-identical.

### The check compares against what a rebuild would actually write

`CheckIndex` builds its expectation from the files *plus* the committed index's unknown entries — the same merge the rebuild performs — so an unknown entry can never appear as `extra`, and a genuinely stale index still fails exactly as before. The alternative (filtering unknowns out of both sides before comparing) opens a gap where a rebuild would write something the check never examined.

### Type-based, not prefix-based

An entry is unknown when its `type` string is unrecognised. The id prefix is not consulted: a future record type may use an unforeseen prefix, and the lenient id grammar is about object files, not index rows.

## Risks / Trade-offs

- **A malformed entry with no `type` at all** is preserved rather than flagged — indistinguishable from a future type by construction. `validate`'s file-level checks still catch malformed *objects*; a hand-mangled index row rides along, which is the price of the MUST.
- **Byte-stability of preserved nodes** depends on yaml.v3 re-emitting a node as it was parsed; the round-trip test pins this, including comment-free multi-line entries.
