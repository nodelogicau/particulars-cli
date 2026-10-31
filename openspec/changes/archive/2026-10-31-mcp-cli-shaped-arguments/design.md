## Context

mcp-model-friction (v0.16.0) added `source.quote` and `source.hash` as shorthand, a schema for `source.document`, and an MCP preamble that maps CLI flags to parameters. The bench re-run (#18) shows the remaining refusals come from three flag-shaped inputs:

- **Top-level `topics`.**
  - `assertIn` already takes `scope` as shorthand for `context.scope`, and `context.scope` wins when both are set (`tools.go:318-324`).
  - `topics` has no shorthand, so the SDK refuses it with `unexpected additional properties ["topics"]`.
- **`source.document_author` and `source.document-author`.**
  - `withShorthand` (`server.go:250`) is table-driven over `quote` and `hash`.
  - mcp-model-friction put document-author shorthand out of scope because `source.author` already names the asserting person. That reasoning rules out `author` as shorthand. It doesn't rule out `document_author`, which is unambiguous and is the CLI flag's own name.
- **Equal values set twice.** `withShorthand` refuses any field that the document sets and the shorthand also sets, even when the two values are equal.

## Goals / Non-Goals

**Goals:**
- Each of the three shapes writes the same file as the canonical nested form.
- No input that differs from what gets written is silently dropped.

**Non-Goals:**
- `source.timestamp` (4 samples). It could mean when the claim was asserted or when the document was dated, so there is no single place to put it.
- CLI flags at the top level, such as `document` and `quote` beside `particular_id` (2 samples).
- A top-level `harness` on `synthesis_create` (3 of 13 calls). `source.harness` exists, and the error names the extra key. This should be revisited if a later run shows it is common.
- `topics` on `synthesis_create`. No sample has it, and `synthesis_create` has no `scope` shorthand for it to sit beside.
- Rewriting SDK validation errors into hints (#16's option (c)), which is still out of scope.

## Decisions

### D1. A field set twice is refused only when the values differ

In `withShorthand`, `*f.into != "" && *f.into != f.short` refuses; equal values pass, and the value is written once. Equality is exact string equality on the raw input, before any author resolution. Two spellings of one author (a name and a URI) are refused with the existing "pass … once" message. The model can then drop one of them. Resolving both and comparing would make the outcome depend on the register's state.

*Alternative:* trim whitespace before comparing quotes. A quote is verbatim evidence, so the server shouldn't decide that two different strings are the same.

### D2. Document-author shorthand reuses the shorthand table

`sourceIn` gains:

```go
DocumentAuthor       string `json:"document_author,omitempty" jsonschema:"shorthand for source.document's author …; needs source.document"`
DocumentAuthorHyphen string `json:"document-author,omitempty" jsonschema:"the same as document_author"`
```

- If both spellings are set and their values differ, the call is refused. If they're equal, they collapse into one value before the table runs.
- The table gains an `author` row, so the rules for quote and hash apply: the shorthand needs a per-call `document`, equal values are accepted, and differing values are refused.
- The filled `doc.Author` then flows through the existing `source.document.author` resolution (`server.go:364`). An unknown id or an ambiguous name gets the same error as in the nested form.
- The messages name the canonical place: `source.document_author needs source.document` and `pass document author once: in source.document or as source.document_author, not both`.

The hyphenated spelling accounts for 6 of the 19 samples. JSON allows the key, and the SDK infers it from the tag. Accepting it costs one field. Refusing it costs a turn every time it appears.

### D3. Top-level `topics` is unioned with `context.topics`

`assertIn` gains `Topics []string \`json:"topics,omitempty" jsonschema:"shorthand for context.topics"\``. `claimAssert` writes `context.topics` first, then each shorthand topic not already present.
- Topics are a set, so a union drops nothing. `scope` has to pick one value; `topics` doesn't.
- Normalisation and validation stay where they are today, so a shorthand topic is checked exactly like a context topic.

### D4. Preamble lines

- `--topic` → `context.topics` (or top-level `topics`)
- `--document-author`, `--quote`, `--document-hash` → `source.document.author`, `.quote`, `.hash`; each may also be given on `source` as `document_author`, `quote`, `hash`

`docs/mcp.md` keeps naming the nested form as canonical.

## Risks / Trade-offs

- [More accepted spellings mean more variety in the wild] → Every spelling is input-only and writes an identical file, and the docs name one canonical form.
- [`document-author` is an unusual JSON key] → It is accepted, but the preamble and docs never teach it, so it serves only as a fallback.
- [Accepting equal duplicates hides a sloppy call] → Nothing is lost or misattributed, and the file is identical to the one written from the nested form.

## Migration Plan

Nothing to migrate. Release in the next minor version, then re-run the bench against 157 errors in 1,075 calls.

## Open Questions

None.
