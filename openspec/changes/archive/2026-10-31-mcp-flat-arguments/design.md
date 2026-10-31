## Context

`claim_assert` accepts the CLI's flags as top-level fields:
- `scope` and `topics` fold into `context`. For `scope`, the context value wins. For `topics`, the two lists are joined.
- `document`, `quote`, `hash`, `document_author` and `document-author` fold into `source`, through `foldSource(in *assertIn)` (`internal/mcp/tools.go`).

`foldSource` runs before `s.source`, so the folded fields go through the same rules as the nested form: `documentFrom`, `withShorthand`, author resolution and the blank-quote check. Two fields given in both places must be equal, and `document` is compared with `reflect.DeepEqual`.

`synthesis_create` (`synthesisIn`) has none of these shortcuts. It reads `context` directly, and its source needs a harness (`needHarness = true`), which falls back to the connected client's name.

The v0.18.0 bench left three error shapes, listed in the proposal. All three come from the model shaping its arguments after the CLI's flags.

## Goals / Non-Goals

**Goals:**
- A top-level document and `source.document` that describe the same document are accepted, and nothing either one carries is dropped.
- `claim_assert` and `synthesis_create` accept the same top-level shorthand, implemented in one place.
- Every call that was valid before writes the same file as before.

**Non-Goals:**
- Top-level `author`. At the top level it could mean the asserting person or the document's author, and no sample has it.
- Top-level shorthand on `claim_retract`, `knowledge_publish` or `particular_merge`. They had no errors in four runs.
- `source.timestamp`: none in v0.18.0, and its meaning is still unclear.
- Invented tool names (one per run). The preamble already lists every tool.

## Decisions

### D1. A document given both ways merges field by field

`foldSource` no longer compares the raw documents. When both are present:
1. It decodes each with `documentFrom`, which turns a string into `{ref}` and a mapping's `uri` into `ref`.
2. It merges the two `dkf.Document` values one field at a time (`ref`, `author`, `hash`, `quote`):
   - a field set on one side only is taken;
   - a field set to equal values on both sides is accepted;
   - a field set to different values is refused with `pass document.<field> once: at the top level and in source they differ`.
3. It writes the merged document back to `src.Document` as a mapping (`map[string]any`), so `s.source` decodes it as before.

A decode error, for example an unknown key, is returned as is, so it keeps `documentFrom`'s message.

This replaces mcp-top-level-source's decision D1 ("a string never equals a mapping"). That rule refused a pair whose meaning was clear, which is the kind of refusal these changes exist to remove. The field-by-field merge follows the same rule `withShorthand` already uses for quote, hash and author.

*Alternative:* treat a string as equal to a mapping when the `ref`s match, and keep the whole mapping. That works for the observed case. The field-by-field merge covers it and also two partial mappings, with one rule.

### D2. One embedded struct, one fold

```go
type flatIn struct {
	Scope, Topics                                              // → context
	Document, Quote, Hash, DocumentAuthor, DocumentAuthorHyphen // → source
	Harness, Model                                             // → source
}
```

- `assertIn` and `synthesisIn` embed `flatIn`. `encoding/json` and jsonschema-go both promote embedded fields, so the wire shape and the inferred schema stay flat. `withSource[In]()` still maps `documentIn` to the string-or-object schema.
- `assertIn`'s existing `Scope`, `Topics` and source fields move into `flatIn`. The JSON names don't change.
- `(f flatIn) foldSource(src *sourceIn) (*sourceIn, error)`: the existing logic, plus D1 and `harness`/`model` rows that follow the equal-or-refuse rule.
- `(f flatIn) foldContext(ctx *contextIn) (scope string, topics []string)`: `context.scope` wins over `scope`, and `context.topics` is joined with `topics`. These are today's `claim_assert` rules, moved here.

`claimAssert` and `synthesisCreate` each call both folds before doing anything else.

### D3. `harness` on `synthesis_create`

A top-level `harness` folds into `source.harness`. That also satisfies `needHarness`, the same as giving it in `source`. The field description says "shorthand for source.harness (defaults to the connected client)", so a model that leaves it out loses nothing.

## Risks / Trade-offs

- [`synthesis_create`'s schema gains 10 optional properties, most of them never sent] → They share descriptions and code with `claim_assert`. Keeping the two tools inconsistent would cost more, as the 8 refusals in 22 calls show.
- [Field-by-field merging could combine parts from two different documents] → Only when every field the two sides share is equal. With different `ref`s the merge is refused, so two documents with different references can't merge.
- [`flatIn` hides shorthand fields from someone reading `assertIn`] → A comment on the embedding names them and points to `foldSource` and `foldContext`.

## Migration Plan

Nothing to migrate. Release as a minor version, then re-run the bench against v0.18.0.

## Open Questions

None.
