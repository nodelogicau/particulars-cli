## Context

`claim_assert`'s input is `assertIn` (`internal/mcp/tools.go`). It already accepts two fields as top-level shorthand:
- `scope`, for `context.scope`, where the context value wins;
- `topics`, for `context.topics`, where the two lists are joined.

Source fields go through `s.source(g, req, in.Source, false)`. That function runs `documentFrom`, then `withShorthand`, which folds `source.quote`, `source.hash` and `source.document_author` into the document. It refuses a field that is set both ways with different values. The input schema comes from `withSource[assertIn]()`. That function maps the `documentIn` type to the string-or-object schema, so any field of that type gets the stated shape, not only `source.document`.

On v0.17.0 the sessions that failed sent the CLI flags as a flat object, with no `source` at all. Each sample had `document` and `quote`, most had `document_author`, and many also had `timestamp`, `scope` and `topics`. Those three are already valid at the top level.

## Goals / Non-Goals

**Goals:**
- `claim_assert{particular_id, content, evidential, document, quote, document_author}` writes the same file as the nested form.
- Every check that applies to `source` applies here too, with no second code path.
- The preamble teaches one form and doesn't invite flattening.

**Non-Goals:**
- Top-level `author`, `harness` or `model` on `claim_assert`. No sample has them. `author` would also be ambiguous at the root: is it the asserting person or the document's author?
- Top-level source fields on the other tools that take `source` (`synthesis_create`, `claim_retract`, `knowledge_publish`, `particular_merge`). The only errors there were 3 in 12 `synthesis_create` calls, and those involved `harness`, `topics` and `scope`.
- `source.timestamp` (3 samples), for the reasons in mcp-cli-shaped-arguments.

## Decisions

### D1. Fold the top-level fields into `source` before `s.source` runs

`assertIn` gains:

```go
Document             documentIn `json:"document,omitempty" jsonschema:"shorthand for source.document"`
Quote                string     `json:"quote,omitempty" jsonschema:"shorthand for source.quote …"`
Hash                 string     `json:"hash,omitempty" …`
DocumentAuthor       string     `json:"document_author,omitempty" …`
DocumentAuthorHyphen string     `json:"document-author,omitempty" …`
```

A helper, `foldSource(in *assertIn) (*sourceIn, error)`, runs first in `claimAssert`.
- **Nothing set at the top level:** it returns `in.Source` unchanged.
- **Otherwise:** it copies `in.Source`, or starts from an empty `sourceIn`, and sets each top-level field the source leaves empty.
- **A field set in both places with a different value:** the call is refused with `pass document once: at the top level or in source, not both`.
- **The same value in both places:** it is accepted.
- **How `document` is compared:** with `reflect.DeepEqual` on the decoded values, so a string never equals a mapping.

The folded `sourceIn` then goes through `s.source` unchanged, and so through `documentFrom`, `withShorthand`, author resolution and the blank-quote check. A top-level `quote` with no document anywhere gets the existing `source.quote needs source.document` message.

*Alternative:* merge the fields in a second place, after `documentFrom`. That would duplicate the shorthand rules, and they would drift apart.

### D2. Only `claim_assert`

`scope` and `topics` are already shorthand on `claim_assert` alone, and the observed shape occurs only there. Adding the same fields to the other four tools would widen their schemas with nothing to measure the change against.

### D3. The preamble states only the nested form

The flag lines go back to:
- `--document-author`, `--quote`, `--document-hash` → `source.document.author`, `.quote`, `.hash`
- `--topic` → `context.topics`

The preamble stops mentioning shorthand. The v0.17.0 run suggests that naming alternative spellings encourages a small model to flatten further. The run can't prove that. Taking the lines out costs nothing, though, because every shorthand is still accepted. The field descriptions in the schema still say "shorthand for …", so a model reading the schema finds them, and `docs/mcp.md` still documents them for people.

## Risks / Trade-offs

- [`claim_assert` now has four ways to pass a quote, and the schema gets longer] → Every way writes the same file, the descriptions point at the canonical field, and the docs name one form.
- [Reverting the preamble brings back the v0.16.0 shapes] → Those shapes are accepted now, so they would only cost tokens, not errors.
- [The cause of the flattening isn't proven] → The change is worthwhile whatever the cause: the flat CLI shape is now accepted however a model arrives at it. The bench re-run reports both the error count and how many sessions sent the flat shape.

## Migration Plan

Nothing to migrate. Release in the next minor version, then re-run the bench against 85 errors in 962 calls.

## Open Questions

None.
