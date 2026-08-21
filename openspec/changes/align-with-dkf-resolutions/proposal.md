## Why

On 2026-08-21 the DKF draft absorbed all ten pieces of feedback from this implementation ([nodelogicau/particulars@27743db](https://github.com/nodelogicau/particulars/commit/27743db), baseline specs under its `openspec/specs/`). It adopted most of what particulars-cli already does, but decided three points differently, so v0.1.1 now writes files the spec considers malformed (`produced-by` on syntheses), rejects files the spec considers valid (agent-only `source`), and lacks a record type the spec defines (merge records). A reference implementation that diverges from the spec it references is worse than none.

## What Changes

- **Source provenance**: a `source` needs at least one of `author` or `harness` (not `author` specifically) on claims, retractions, and merges. Syntheses carry `source` (with `harness` required) in place of `produced-by`, at the same position. Legacy `produced-by` files are read as `source` with a `validate` warning. `synthesis create` gains `--author` and `--document`. **BREAKING** for consumers of the synthesis JSON/YAML shape: `produced-by` → `source`.
- **Merge records**: new `mrg_` record type at `merges/mrg_<uuidv7>.yaml` (`id, type: merge, uris[2], reason?, source, timestamp, retracted?`), written by a new `particular merge <a> <b>` verb, indexed, retractable, and validated. Non-retracted merges form symmetric, transitive equivalence classes of particulars; `recall`, `conflicts`, and `lineage` operate over the class.
- **Retraction verb**: new top-level `retract <id>` accepting claims, syntheses, and merges; `claim retract` remains as an alias.
- **Conflict semantics**: `current` ordered by `timestamp` then id (was id only); `stale` includes transitively retracted inputs (was direct only); sets computed per equivalence class. `recall` entries gain `unsynthesised: true`.
- **Identifiers**: `mrg` prefix added to minting and the lenient read grammar; `validate` warns (`legacy_id`) on ids that are not canonical UUIDv7.
- **Workspace config**: `workspace.base-uri` must end in `/` — `init` normalises and reports it, `validate` errors otherwise, `MintURI` drops its `#`/`:` special-casing.
- **Lineage**: retracted nodes expose `superseded_by` (informational, never an input).
- **Docs**: README, skill (both copies), review workflow, and `SPEC-FEEDBACK.md` updated to record the resolutions; `None identified` documented as the conventional `unresolved` value.
- Release as **v0.2.0** (on-disk synthesis shape changes; everything v0.1.x wrote remains readable).

## Capabilities

### New Capabilities
- `merge-records`: The `mrg_` record type, the `particular merge` verb, equivalence-class semantics consumed by query verbs, and merge validation.

### Modified Capabilities
- `object-format`: `mrg` prefix; synthesis field order with `source`; merge record field order; source minimum is author-or-harness.
- `claims`: assert no longer requires `author` when `harness` is present; retraction generalised to a top-level `retract` verb covering merges, with `claim retract` as alias.
- `syntheses`: `source` replaces `produced-by` (harness required); legacy read; `None identified` convention; `--author`/`--document` flags.
- `knowledge-query`: recall and lineage operate over merge classes; recall marks `unsynthesised`; lineage shows `superseded_by`.
- `conflict-detection`: timestamp-then-id ordering for `current`, transitive staleness, per-class computation.
- `index`: merge entries with `uris`; `particular merge` and `retract` maintain the index.
- `validation`: source minimum, `legacy_produced_by` and `legacy_id` warnings, merge checks, base-uri trailing slash, unknown merge URI warning.
- `workspace`: base-uri normalisation at `init` and trailing-slash requirement.

## Impact

- `internal/dkf`: types (Synthesis.Source, new Merge), id grammar, codec (synthesis order, merge encoding, legacy `produced-by` decode), validation rules.
- `internal/store`: `merges/` directory, graph loads merges and builds equivalence classes, index entries for merges, retract accepts merges.
- `internal/query`: class-aware recall/conflicts/lineage, timestamp ordering, transitive stale, new findings.
- `internal/cli`: `particular merge`, `retract`, `synthesis create` flags, `init` normalisation, JSON shape change (`source` on syntheses).
- Main specs under `openspec/specs/` updated on archive; `SKILL.md` (both copies), README, docs, `SPEC-FEEDBACK.md`.
- The knowledge workspace at `~/IdeaProjects/particulars-knowledge` (four syntheses with `produced-by`) stays readable; `validate` there will show four `legacy_produced_by` warnings until re-synthesised — expected, not an error.
- Not in scope: `knowledge_publish`/feeds, signing, `.well-known` discovery, MCP transport, a `particular rehome` verb, migration/rewrite of existing files (immutability forbids it).
