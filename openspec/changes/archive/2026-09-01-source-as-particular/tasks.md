## 1. Reference plumbing (dkf, query)

- [x] 1.1 Add `Author` to `dkf.Document` after `Ref`; update `UnmarshalYAML`, `MarshalYAML`, `UnmarshalJSON`, `MarshalJSON`, and `Document.Validate` (mapping with `author` but no `ref` stays invalid)
- [x] 1.2 Update `codec.go` `documentNode`: order `ref, author, hash, quote`; widen the scalar shortcut so a ref+author document keeps the mapping form
- [x] 1.3 Add `ClassifyRef` (id | uri | name) and extend `query.Resolve` URI lookup through non-retracted merge classes
- [x] 1.4 Add `query.ResolveAuthor(g, value)` → resolved particular or nil (none/ambiguous), with a variant returning candidates for error messages

## 2. Writer resolution

- [x] 2.1 Add `resolveAuthorForWrite(g, value, explicit)` implementing the settled table (refuse unknown id → not found; refuse explicit ambiguous name → usage listing candidates; name→URI on exactly one; else unchanged; stderr note on ambiguous default)
- [x] 2.2 Wire into CLI `resolveSource` for `source.author` and `source.document.author` on every writing verb (assert, synthesis, retract, merge, publish); reorder `claim assert` so source resolution follows graph load
- [x] 2.3 Add `--document-author` to `provenanceFlags`/`claim assert` (usage error without `--document`)
- [x] 2.4 Wire into MCP `(*Server).source`; call-level values are explicit, server flag and dkf.yaml are defaults

## 3. Recall by author

- [x] 3.1 `RecallOptions.Author`; resolve the query (exit 3 none / exit 2 ambiguous with candidates); compute per-entry `relations` over the class; `Entry.Relations` omitted without the filter, never empty with it
- [x] 3.2 CLI: `recall --author` flag; author alone is a sufficient selector; text renderer shows the relations
- [x] 3.3 MCP: `recallIn.Author`; results equal the CLI's `--json`

## 4. Index

- [x] 4.1 `store.Entry` gains `Author`, `DocumentAuthor` after `Subject`, mirrored as written; `EntryFor` populates from `GetSource()`
- [x] 4.2 Unknown-field preservation: custom `Entry` unmarshal capturing unrecognised keys, re-emitted after known fields; rebuild and `UpsertIndex` merge committed unknowns per entry
- [x] 4.3 `CheckIndex` masking: drop `scope`/`topics`/`timestamp`/`author`/`document-author` from a rebuilt entry when absent from the committed one, before both byte and entry comparison; `retracted` never masked; tests for old-index-passes, first-retraction-fails, changed-value-fails

## 5. Validate

- [x] 5.1 One author-resolution pass feeding: `author_unresolved` (info) and `author_ambiguous` (warning, candidates in message), aggregated per value via message-keyed grouping for these two codes; both on the corpus-fact whitelist; `--json`/`--notes` unchanged in behaviour
- [x] 5.2 `orphan_particular` exempts particulars whose class is the target of asserted-by or reported-from

## 6. MCP document mapping

- [x] 6.1 `documentFrom` accepts `ref` (canonical) + legacy `uri` (both → usage error), plus `author`; unknown keys become usage errors; schema description strings updated

## 7. Skill, docs, dogfood

- [x] 7.1 SKILL.md: `--author`/`--document-author` in the verbs table and rules — who told you goes in `--document-author`; `--author` is you or the human you work for; pass it only when it differs from the default
- [x] 7.2 README (Configuration + verb table) and docs/provenance.md: reference forms, writer table, relations; docs/mcp.md tool notes
- [x] 7.3 SPEC-FEEDBACK.md: record items 16–22 round-trip (raised 2026-08-31, resolved upstream 2026-08-31/2026-09-01)
- [x] 7.4 Run against the dogfood workspace: `validate` clean (no orphan for Ben, no author findings), `recall --author ben` returns the corpus, new assert writes the GitHub URI

## 8. Verification

- [x] 8.1 `go test ./...` and `golangci-lint run` green; `openspec validate --change source-as-particular` passes
