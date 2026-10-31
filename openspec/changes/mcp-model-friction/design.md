## Context

Two things the server tells the model cause #16:

**The source input shape.**
- `sourceIn` (`internal/mcp/server.go:138`) has the fields `author`, `harness`, `model` and `document`.
- `Document` is typed `any`, because a schema inferred from a struct would reject the string form. With that type, the schema says nothing about what goes inside `document`.
- The SDK validates arguments against the inferred schema before any handler runs, and `sourceIn` sets `additionalProperties: false`. So a flat `source.quote` is refused with the SDK's generic message, and `documentFrom` never sees the call.
- `sourceIn` is shared by `claim_assert`, `synthesis_create`, `claim_retract`, `knowledge_publish` and `particular_merge`.
- Precedent for shorthand: `assertIn.Scope` (`tools.go:291`) is shorthand for `context.scope`, and the context value wins silently when both are set.

**The instructions.**
- `instructions()` writes four parts in order: a header, `skill.Body()` unchanged, the register, and the conventions document.
- The header's second line names the DKF tool families, but nothing maps the skill's `particulars <verb> … --json` lines to tools.
- The skill is shared with every CLI harness, and CI checks the installed copy against the embedded one, so its text has to stay the same for both audiences.

## Goals / Non-Goals

**Goals:**
- The flat `source: {document, quote}` shape writes the same claim as the nested shape.
- The schema states the shape of `document` where a model reads it.
- Over MCP, every CLI command in the skill can be traced to a tool, or to "no equivalent", without the model guessing.
- When a tool is added, the mapping can't silently fall behind.

**Non-Goals:**
- Changing `SKILL.md`, or producing an MCP-specific version of it.
- Registering decoy tools such as `particulars` to catch wrong guesses.
- Rewriting SDK validation errors into hints, which was option (c) in #16. Once the shorthand exists, the flat shape no longer errors.
- Shorthand for `document.author`. `source.author` already means the asserting person, so `author` can't be shorthand.
- The CLI's handling of `--quote` without `--document` (`internal/prov/prov.go:29`, where it is ignored). That's a separate question.

## Decisions

### D1. Shorthand is merged in `source()`, before provenance resolves
`sourceIn` gains:

```go
Quote string `json:"quote,omitempty" jsonschema:"shorthand for source.document's quote …"`
Hash  string `json:"hash,omitempty"  jsonschema:"shorthand for source.document's hash …"`
```

After `documentFrom` returns, `source()` applies the shorthand:
- **No per-call `document`:** usage error, `source.quote needs source.document: the reference it quotes`, and likewise for `hash`. The server's default provenance never supplies a document, so a quote here could only be dropped or misattributed.
- **Document field already set:** usage error, `pass quote once: in source.document or as source.quote, not both`. A quote is evidence, so silently picking one value would throw away the other. This differs from `scope`, where `context.scope` wins silently. A wrong scope gets corrected; a lost quote doesn't come back.
- **Otherwise:** the shorthand fills the field. A blank `quote` gets the refusal `documentFrom` already gives (`is blank; omit it`), so `HasQuote()` is checked after the merge, not inside `documentFrom` alone.

The merge doesn't depend on the document's form. A string document is a `Document{Ref: s}` by the time the merge runs, so "with a string document, the string becomes `ref`" holds with no special case.

*Alternative:* allow the shorthand only with a string document, as #16 suggests. That adds a refusal for a shape whose meaning is clear, and costs a turn the change exists to save.

### D2. The document schema comes from `TypeSchemas`, not a hand-edited tool schema
1. Change `Document any` to `Document documentIn`, where `type documentIn interface{}`. It is a named empty interface, so `encoding/json` decodes into it exactly as it does into `any`, and `documentFrom` keeps its type switch.
2. Build one schema for that type:

   ```
   {anyOf: [{type: string},
            {type: object, properties: {ref, uri, author, hash, quote: {type: string, description: …}}}]}
   ```

3. Infer the input schema of each tool that takes `source` with `jsonschema.For[In](&jsonschema.ForOptions{TypeSchemas: {reflect.TypeFor[documentIn](): docSchema}})`, and pass it as `Tool.InputSchema`. A small generic helper `withSource[In]()` keeps the five registrations one line each.

- The object branch deliberately leaves out `additionalProperties: false`. An unknown key then reaches `documentFrom`, which names the key and the fields allowed. That message is better than the SDK's `anyOf` failure.
- `anyOf` is used rather than `oneOf`. The two branches are disjoint by type, and `anyOf` keeps validator errors simpler.

*Alternative:* infer the schema, then patch `Properties["source"].Properties["document"]`. That hard-codes the property path in five places, and a renamed field would fail silently. `TypeSchemas` follows the type wherever it appears.

### D3. The preamble is generated from one table, and a test checks it against `ListTools`
`server.go` gets a table:

```go
var cliTools = []struct{ CLI, Tool, Note string }{
	{"particular resolve", "particular_resolve", ""},
	…
	{"register", "", "already below — no call needed; topics_list refreshes the topic counts"},
	{"export / index / init / skill install", "", "no MCP equivalent"},
}
```

`instructions()` writes the header, then the preamble, then the skill body. The header's existing line naming the DKF tool families moves into the preamble.

Preamble text, to be adjusted only for wording:

> ## Using this skill over MCP
>
> The skill below is written for the `particulars` command line. Over MCP there is no shell and no `particulars` tool: each command it names is one of this server's tools, listed here. Ignore `--json`, because every tool result is already structured. Flags become parameters. For `claim assert`:
> - `--subject` → `particular_id`
> - `--document` → `source.document`
> - `--document-author`, `--quote` and `--document-hash` → `source.document.author`, `.quote` and `.hash`; `quote` and `hash` may also be given directly on `source`
> - `--topic` → `context.topics`
> - `--hash-document` has no MCP equivalent: pass `hash` yourself, or omit it
>
> | Command in the skill | Tool |
> |---|---|
> | … generated from `cliTools` … | |
>
> Tool names follow the DKF specification; `topics_list`, `unresolved_list` and `workspace_status` are particulars extensions.

Test: list the server's tools, then assert that the set of non-empty `Tool` values equals the set of registered tool names. Adding a tool without a row fails, and so does a row naming a tool that was removed.

*Alternative:* rewrite the skill's command lines into tool calls at serve time. Regex rewriting of prose and code blocks is fragile, and a tool call with JSON arguments doesn't fit the skill's one-line command examples. *Alternative:* keep a second, MCP-only skill. Two texts that say the same thing will drift, which is the drift the installed-skill check exists to stop.

### D4. Order within the instructions
The order becomes: header, MCP preamble, skill body, register, conventions. The preamble comes before the skill so that the model reads the translation before the commands it translates. The conventions document stays last, and still takes precedence.

## Risks / Trade-offs

- [A model reads the preamble and still calls `particulars`] → The bench re-run measures this. If the rate stays high, the next step is a single sentence at the very top of the header, which a model is likeliest to read. A decoy tool remains out of scope.
- [A `TypeSchemas` entry interacts with the field's `jsonschema` tag description] → Test that the inferred `source.document` schema has both the `anyOf` and a description. If the tag's description is lost, put it in `docSchema`.
- [The shorthand makes two spellings legitimate, and examples in the wild will mix them] → Both are input-only and write identical files. `docs/mcp.md` names the nested form as canonical.
- [The instructions get longer] → The preamble is about 30 lines next to a skill of about 290. It is small compared with the register.

## Migration Plan

Nothing to migrate. Every call that was valid before is still valid and writes the same file. Release it in the next minor version, then re-run particulars-bench against the #16 baseline.

## Open Questions

None blocking. Whether `--hash-document` should get an MCP equivalent (the server hashing a workspace-local `ref`) is a separate, small feature, so the preamble says it has none.
