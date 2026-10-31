## Why

particulars-bench ran Claude Haiku 5.5 as the ingest agent against `particulars serve --mcp` 0.15.0 (#16). 834 of 1,686 `claim_assert` calls failed, and 58 calls went to tools that don't exist. No knowledge was lost: each time, the model recovered on the next turn. But each failure costs a turn, and a smaller model hits both failures in almost every session. Both come from what the server tells the model.

- **`quote` beside `source.document`.** The model sends `source: {document: "…", quote: "…"}`. The inferred schema rejects that shape before the handler runs, so the model gets back `unexpected additional properties ["quote"]` and nothing more. The flat shape is what the model has been taught: the skill shows `--document X --quote Y` as sibling flags (`SKILL.md:132-133`). And `document` is typed `any`, so its description is the only thing that says `quote` goes inside it.
- **CLI commands in the instructions.** `instructions()` sends the skill body unchanged (`internal/mcp/server.go:103`). The body tells the model to "always pass `--json`" and gives about 30 `particulars …` command lines. With no shell to run them, the model calls tools named `particulars`, `particulars_register` and `particulars_particular_resolve`. One step needs no call at all: the register the skill says to read is already in the instructions.

## What Changes

- Every tool that takes `source` accepts `source.quote` and `source.hash` as shorthand for the same fields on the document, as top-level `scope` is shorthand for `context.scope`:
  - With a string `document`, the string becomes `ref`.
  - With a mapping `document`, the shorthand fills a field the mapping leaves empty.
  - Setting a field both ways is refused, and so is shorthand with no per-call `document`.
  - The written claim is identical to one sent in the nested form. The DKF file format doesn't change.
- `source.document` gets an explicit input schema: a string, or an object whose properties are `ref`, `uri`, `author`, `hash` and `quote`. Today the schema leaves it unconstrained, so the shape is stated only in prose.
- The instructions get an MCP preamble between the header and the skill body. It says:
  - there is no `particulars` tool and no shell;
  - `--json` doesn't apply, because every result is already structured;
  - which tool serves each `particulars <verb>` the skill names, and which verbs have no MCP equivalent;
  - how the `claim assert` flags map to parameters;
  - the register is already below, so reading it needs no call.

  The prompt `particulars-discipline` carries the same text. The skill body itself is unchanged.
- A test fails if a registered tool is missing from the mapping, or if the mapping names a tool that isn't registered.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`:
  - *Spec tool surface* adds the `source.quote` / `source.hash` shorthand with its refusals, and the explicit `source.document` schema.
  - *Instructions and prompt carry the skill* adds the MCP preamble and its position, between the header and the skill body.

## Impact

- `internal/mcp/server.go`:
  - `sourceIn` gains `Quote` and `Hash`.
  - `source()` merges the shorthand into the document.
  - `instructions()` writes the preamble.
  - A mapping table pairs CLI verbs with tools.
- `internal/mcp/tools.go`: tools that take `source` register with an input schema built with `TypeSchemas` for the document type. `github.com/google/jsonschema-go` moves from an indirect dependency to a direct one; it is already in the module graph through the SDK.
- Tests in `internal/mcp/server_test.go`.
- `docs/mcp.md` mentions the shorthand.
- `CHANGELOG.md` gets an entry. Nothing breaks: every call that was valid before is still valid and writes the same file.
- After release, re-run particulars-bench at the issue's settings (bench `4b5d19f`, oracle with 70 questions, Haiku 5.5 ingest) and record the error counts against the #16 baseline.
