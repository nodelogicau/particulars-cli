## Why

On v0.18.0 the flat CLI shape is no longer refused, and `claim_assert` errors dropped to 16 in 891 calls (#18). The bench still produced three kinds of error:

- **A top-level string document conflicting with `source.document` (8 of the 16 errors, all in one session).**
  - The model sent `document: "session X"` together with `source.document: {ref: "session X", quote: …}`.
  - The cause is my design decision D1 in mcp-top-level-source: it compares the two decoded values, so a string never equals a mapping.
  - Yet a string document *is* `{ref: string}`. These two values describe one document, and refusing the call protects nothing.
- **Top-level `harness` on `claim_assert` (3 samples).** This is the same flat CLI shape again, this time including `harness`.
- **Top-level `harness`, `topics` or `scope` on `synthesis_create` (8 errors in 22 calls).**
  - Models flatten a synthesis the way they flatten a claim.
  - `synthesis_create` accepts none of the shorthand `claim_assert` does, so more than a third of its calls failed.

## What Changes

- **Documents merge field by field.** A document given both at the top level and in `source` is decoded first, so a string becomes `{ref}`. The two are then merged one field at a time:
  - a field set on one side only is kept;
  - a field set to the same value on both sides is accepted;
  - a field set to different values on the two sides is refused.

  So `"session X"` combined with `{ref: "session X", quote: …}` writes `{ref: "session X", quote: …}`.
- **Top-level `harness` and `model`** become shorthand for `source.harness` and `source.model`. Top-level `author` stays out: at the top level it could mean either the person asserting or the document's author.
- **`synthesis_create` gets the same top-level shorthand as `claim_assert`:**
  - `scope` and `topics`, which fold into `context`;
  - `document`, `quote`, `hash`, `document_author`, `harness` and `model`, which fold into `source`.

  Both tools share one embedded input struct and one fold function, so the two can't drift apart.
- **No other changes.** The DKF file format is unchanged, and every call that was valid before is still valid and writes the same file.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `mcp-server`: three changes to *Spec tool surface*:
  - the top-level shorthand is defined once and covers both `claim_assert` and `synthesis_create`, and now includes `harness` and `model`;
  - a document given both ways merges field by field;
  - `synthesis_create` accepts `scope` and `topics` as shorthand for its `context` fields.

## Impact

- `internal/mcp/tools.go`:
  - a new embedded `flatIn` struct holds the shorthand fields, and both `assertIn` and `synthesisIn` embed it;
  - `foldSource` becomes a method on `flatIn`, gains the field-wise document merge, and handles harness and model;
  - a small `foldContext` helper applies the `scope` and `topics` rules for both tools.
- Tests in `internal/mcp/server_test.go`, plus updates to `docs/mcp.md` and `CHANGELOG.md`.
- After release, re-run particulars-bench at the #16 settings and compare against v0.18.0: 16 `claim_assert` errors in 891 calls, and 8 `synthesis_create` errors in 22 calls.
