## Why

The particulars-bench re-run on v0.17.0 (#18) cut `claim_assert` errors from 157 in 1,075 calls to 85 in 962, and none of the shapes #18 covered recurred. A different shape took their place. 80 of the 85 errors came from 9 of the 70 sessions. In those sessions the model sent the source fields at the root of the call, with no `source` object at all:

```
{particular_id, content, evidential, document, quote, document_author, timestamp, scope, topics}
```

These are the CLI's `claim assert` flags, written one-to-one as arguments. The SDK refuses them with `unexpected additional properties ["document" "document_author" "quote"]`, and these sessions then failed 5–14 times each. On v0.16.0 only one session did this.

The v0.17.0 preamble may have encouraged it. It says the fields "may also be given directly on `source`", next to "or top-level `topics`". `claim_assert` already accepts top-level `scope` and `topics` as shorthand for their `context` fields, so the source fields are the only flags with no flat spelling.

## What Changes

- `claim_assert` accepts these top-level fields as shorthand for the matching `source` fields:
  - `document`, as a string or a mapping;
  - `quote`;
  - `hash`;
  - `document_author`, or `document-author`.

  Each one is folded into `source` before provenance is resolved, so every rule that applies to `source` applies unchanged. In particular, `quote`, `hash` and `document_author` need a `document`. The written claim is identical to one sent through `source`.
- A field given both at the top level and in `source` follows the same rule as the shorthand inside `source`: equal values are accepted and written once, and values that differ are refused.
- The MCP preamble goes back to stating only the nested form for `claim assert`'s flags. The shorthand is a fallback that the docs and schema descriptions mention. The preamble doesn't teach it.
- The DKF file format doesn't change. Every call that was valid before is still valid and writes the same file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`: *Spec tool surface* gains top-level source shorthand on `claim_assert`.

## Impact

- `internal/mcp/tools.go`: `assertIn` gains `Document`, `Quote`, `Hash`, `DocumentAuthor` and `DocumentAuthorHyphen`, and `claimAssert` folds them into `Source`.
- `internal/mcp/server.go`: the preamble's flag lines change.
- Tests in `internal/mcp/server_test.go`, and updates to `docs/mcp.md` and `CHANGELOG.md`.
- After release, re-run particulars-bench at the #16 settings and compare against 85 errors in 962 calls, and against the 9 sessions that sent the fields at the top level.
