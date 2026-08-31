## Context

Three DKF changes to implement at once, already reconciled upstream (#7 and nodelogicau/particulars#17–#22). The CLI's shape constrains where each piece lands: `internal/prov` is deliberately store-blind, so writer resolution cannot live there; the document codecs are hand-built `yaml.Node` writers with a scalar shortcut; the index drift check compares bytes; corpus-fact aggregation is a presentation-layer whitelist keyed by severity+code. The dogfood workspace (172 objects, one person particular with alias `ben`) is the acceptance environment.

## Goals / Non-Goals

**Goals:**
- Author references resolve everywhere a source is read, written, filtered, or validated, with one resolver.
- The writer refusal table exactly as settled upstream; retroactive attribution with no file changes.
- The drift check honours the immutability tolerance without losing retraction staleness or byte-level formatting checks.

**Non-Goals:**
- `harness`/`model` as references; a `kind` on particulars; scope on particulars; index-based recall candidates (the graph is fully loaded anyway — the index fields are written for other consumers, per the DKF spec).
- Minting the author's particular anywhere (now prohibited upstream).
- Rewriting any existing file.

## Decisions

**D1. One reference classifier, one resolver, in `internal/query`.** `ClassifyRef(value)` → id | uri | name: `par_` prefix → id; scheme prefix (`^[A-Za-z][A-Za-z0-9+.-]*:`) → URI; else name. `ResolveAuthor(g, value)` → the particular the value resolves to (nil when none or ambiguous), reusing `Resolve` with URI lookup extended through merge classes: a URI not carried by any particular but joined by non-retracted merges to one resolves to that particular's class. The extension lands in `Resolve` itself so subjects gain it too — harmless, since recall already operates over the class. Names stay case-insensitive (`strings.EqualFold`), matching `particular resolve`.

**D2. Writer resolution at the two provenance call sites, not in `prov`.** `prov.Resolve` stays string-pure. A new `resolveAuthorForWrite(g, value, explicit bool)` applies the settled table (refuse unknown id; refuse explicit-ambiguous name with candidates; else URI-or-unchanged) and is called from `cli` `resolveSource` and `mcp` `(*Server).source` after `prov.Resolve`, for both `source.author` and `source.document.author`. **Explicit** = the verb's `--author`/`--document-author` flag or the MCP call's `source.author`/`document.author`; **default** = `dkf.yaml`, `$DKF_AUTHOR`, and `serve --author` (set once, applies to every call — behaviourally a default, so ambiguity falls through to written-unchanged rather than failing all writes). Both call sites must therefore run after the graph is loaded; `claim assert` reorders so source resolution follows `loadGraph`.

**D3. `Document.Author` after `Ref`.** Struct field, four codecs (`UnmarshalYAML`/`MarshalYAML`/`UnmarshalJSON`/`MarshalJSON`), and `documentNode` order `ref, author, hash, quote`. The scalar shortcut widens its condition: scalar only when author, hash, and quote are all empty. `documentFrom` (MCP) accepts `ref` as canonical, `uri` as legacy alias (fixing the defect where only `uri` was read), plus `author` — and continues to ignore nothing silently: an unknown key is a usage error now that the accepted set is spec-complete.

**D4. Relations computed at recall time.** `RecallOptions.Author`; the filter resolves the query to a class (ambiguous name → usage error listing candidates, as subjects do), then for each candidate object resolves `source.author` and `source.document.author` and intersects with the class. `Entry.Relations []string` (`asserted`, `reported`), omitted when no author filter is given; never empty when it is. `--author` becomes a sufficient selector in `cmd_query` (subject, topic, or author).

**D5. Index: two mirrored fields, unknown-field preservation, masked comparison.** `Entry` gains `Author`/`DocumentAuthor` (yaml `author`, `document-author`) after `subject`, mirroring the file as written, plus an `Unknown []yaml.Node`-backed key/value tail captured by a custom `UnmarshalYAML` and re-emitted after known fields in committed order. Rebuild and incremental update merge each regenerated entry with the committed entry's unrecognised fields (the field-level twin of `spliceUnknown`). `CheckIndex` masks before encoding: for each rebuilt entry, drop any of the five immutable MAY fields (`scope`, `topics`, `timestamp`, `author`, `document-author`) absent from the committed entry; unknown committed fields ride along via preservation; `retracted` is never masked, so absent-vs-`true` stays drift. Byte comparison then works unchanged — an old index passes, formatting drift and retraction staleness still fail. The residual asymmetric case (a known field present in the committed entry, absent from the rebuild) can only be corruption of an immutable mirror and surfaces as a changed entry, which errs toward visibility.

**D6. Aggregation per author value.** `author_unresolved` (info) and `author_ambiguous` (warning, message carrying the candidates) join the corpus-fact whitelist. The renderer's grouping key becomes severity+code+message *for these two codes only* — their messages are constructed uniform per value ("author %q …"), so one line per name falls out; other codes keep severity+code grouping since their messages vary per object. `--json` still carries every finding. `orphan_particular` skips any particular whose class is the target of an asserted-by or reported-from resolution.

**D7. Validation reads authors once.** A single pass resolves every distinct author value (source and document) to none/one/many, feeding the aggregates, the orphan exemption, and nothing else — resolution stays best-effort and lenient; no author condition is an error.

## Risks / Trade-offs

- [Writer resolution changes what `author` looks like in every new dogfood claim (name → GitHub URI)] → Intended by the spec (the freeze argument); the diff visibility is the point. Tests pin the table.
- [A label containing a colon classifies as a URI and skips name resolution] → Written unchanged; degradation is harmless and documented. Names that resolve stay the common case.
- [Masking weakens the byte check] → Only for the five immutable fields, only on presence, only where the committed entry lacks them; every value difference and `retracted` still compares.
- [Message-keyed grouping could fragment aggregation if messages drift] → Scoped to the two author codes whose messages are constructed, not free-form.

## Migration Plan

Additive; ships in one release. Existing indexes pass the drift check unrebuilt; the next write or `particulars index` adds the fields. No rollback machinery: an older binary reading a newer workspace loses the join and, on rebuild, no longer strips the new fields it doesn't know only if it postdates D5 — which is why preservation ships now, in the same release as the first new field.

## Open Questions

- None; every upstream question is settled (#17–#22). The optional write-time warning for an ambiguous default author is taken as a `warnings` entry in the write's result (stderr in text mode), per the upstream design's suggestion — not a bare stderr line, which would break the `--json` contract that stderr carries only the error envelope.
