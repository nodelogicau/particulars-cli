## Why

I re-ran particulars-bench on v0.16.0 (#16, follow-up #18). `claim_assert` failed 157 times in 1,075 calls, down from 834 in 1,686. The bench kept 64 samples of those failures. Nearly all of them have the same cause as #16: the model shapes its arguments the way the CLI shapes flags, the SDK rejects them before any handler runs, and the model spends a turn fixing the call.

- **Top-level `topics`** (24 samples). This mirrors `--topic`. `scope` is already accepted at the top level as shorthand for `context.scope`, and `topics` isn't.
- **`source.document_author` or `source.document-author`** (19 samples). This mirrors `--document-author`, and both spellings occur.
- **The same quote both in `source.document.quote` and as `source.quote`** (9 samples). Every sample that could be checked carried identical values. Decision D1 of mcp-model-friction refuses the pair so that one of two differing quotes is never dropped. When the values are equal, nothing can be lost, so the refusal only costs a turn.

## What Changes

- `claim_assert` accepts top-level `topics` as shorthand for `context.topics`. When both are given, the written topics are their union, in order and without duplicates.
- Every tool that takes `source` also accepts `source.document_author` and `source.document-author` as shorthand for `source.document.author`. They follow the same rules as `source.quote`: each needs a per-call `document`, and the author resolves exactly as the nested form does.
- A shorthand field (`quote`, `hash`, or document author) that is also set inside `source.document` is refused only when the two values differ. Equal values are accepted and written once.
- The MCP preamble's flag mapping says `--topic` may also be given as top-level `topics`, and `--document-author` may also be given as `source.document_author`.
- The DKF file format doesn't change. Every call that was valid before is still valid and writes the same file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`: in *Spec tool surface*:
  - `topics` is added as a top-level shorthand on `claim_assert`;
  - the document-author shorthand is added to `source`;
  - the refusal of a field set twice is narrowed to values that differ.

## Impact

- `internal/mcp/server.go`:
  - `sourceIn` gains the two document-author fields;
  - `withShorthand` gains an author row and compares values before refusing;
  - the preamble's flag lines change.
- `internal/mcp/tools.go`: `assertIn` gains `Topics`, and `claimAssert` merges it into the topics it writes.
- Tests in `internal/mcp/server_test.go`.
- `docs/mcp.md` and `CHANGELOG.md`.
- After release, re-run particulars-bench at the #16 settings and compare against 157 errors in 1,075 calls.
