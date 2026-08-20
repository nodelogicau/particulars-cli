## Context

[DKF v0.1 (draft)](https://github.com/nodelogicau/particulars) defines three object types — `DPARTICULAR`, `DCLAIM`, `DSYNTHESIS` — stored as one YAML file each under `particulars/`, `claims/`, `syntheses/`, plus an `index.yaml` manifest. Claims are immutable; correction happens through synthesis, retraction is recorded rather than deleted, and the lineage graph is a DAG. The spec's reference interface is a set of eleven MCP tools; this change implements the eight non-federation ones as CLI verbs.

Operating model decided during exploration:

- **The author is an LLM agent; humans review.** The agent calls the CLI from a shell while working. Review happens through **git pull requests**: the agent's branch is the proposal, merge is acceptance, post-merge rejection is a retraction. The CLI therefore performs no git operations and needs no review state of its own.
- **Single native binary.** Must run anywhere with no runtime.
- **The spec is a draft.** Field names, ID format, and serialisation details may change. This implementation is expected to generate feedback upstream, so format concerns are isolated behind a small codec layer.

The repository is greenfield: no code, no toolchain installed, an OpenSpec scaffold only.

## Goals / Non-Goals

**Goals:**

- Read and write DKF v0.1 workspaces exactly as the spec describes, with every deviation or extension documented here and fed back upstream.
- Be pleasant for an agent to drive: every verb is non-interactive, supports `--json`, and returns meaningful exit codes.
- Be pleasant for a human to review in a PR: deterministic YAML, spec field order, files only ever created (never rewritten) except for the retraction block.
- Keep the binary "dumb": structural queries only. The LLM reasons; the tool stores, indexes, and reports.
- Structure the core so a future `serve --mcp` transport reuses it unchanged.

**Non-Goals:**

- Federation tools (`feed_index`, `particular_merge`, `knowledge_publish`), public discovery manifests, signing.
- MCP server transport (follow-on change).
- Git operations of any kind (branching, committing, PR creation).
- Semantic similarity, embeddings, or any LLM call inside the binary.
- Full-text search over claim content (agents can `grep`; topic filtering is provided).
- A `particular rehome` verb (changing a URI). Allowed by the rules below but not needed until publishing exists.

## Decisions

### D1. Language and build: Go, static binary, cross-compiled

**Choice:** Go module `github.com/nodelogicau/particulars-cli`, `CGO_ENABLED=0`, GoReleaser (or equivalent) producing darwin/linux/windows × amd64/arm64.

**Alternatives:** Rust — comparable binaries but `serde_yaml` is archived and the YAML ecosystem is unsettled, a real risk for a tool whose entire surface is YAML. GraalVM native-image (Kotlin/Java) — familiar stack, but cannot cross-compile (one build runner per OS), slow builds, larger binaries.

**Dependencies:** `spf13/cobra` (CLI), `gopkg.in/yaml.v3` (node-level control over key order), `github.com/google/uuid` (UUIDv7), nothing else in the core.

### D2. Package layout: core is transport-agnostic

```
cmd/particulars/          main; wires cobra to core
internal/cli/             cobra commands, flag parsing, text/JSON rendering
internal/dkf/             format layer: types, ID minting/parsing, YAML codec, validation rules
internal/store/           workspace discovery, dkf.yaml, file IO, index read/write
internal/query/           recall, lineage, conflicts (pure functions over loaded objects)
```

`internal/dkf` has no filesystem access; `internal/query` takes in-memory graphs. A future MCP server is a second front-end in `internal/mcp` calling the same `store` + `query` APIs. The codec is the only place that knows field names and order, so a spec rename is a one-package change.

### D3. Identifiers: UUIDv7 with spec prefixes

**Choice:** `<prefix>_<uuidv7>` where prefix ∈ `par|clm|syn` and the UUID is lowercase canonical hyphenated hex (RFC 9562), minted with the monotonic-counter variant so IDs created within one millisecond still sort in creation order. Example: `clm_019196a5-8b4c-7def-8abc-0123456789ab`. File name is `<id>.yaml`.

**Why:** The spec's examples look like truncated ULIDs and it says the ID format is subject to change. UUIDv7 gives the same time-sortable property with an IETF standard behind it — "it's just a UUID" is the strongest interoperability story for a format whose purpose is portability. Chronological sort on disk makes `ls claims/` a log and gives `recall` a cheap lineage order (inputs always precede the syntheses that cite them, because objects are immutable).

**Semantics:** The ID timestamp is the *minting* instant. A claim's `timestamp` field is the *assertion* time and may be earlier (e.g. recording a dated document). Validation does not require them to agree.

**Alternatives:** ULID — shorter, more readable, not an RFC. Random UUIDv4 — no ordering. Content hashes — break on any field change and say nothing about time.

### D4. Deterministic, create-only serialisation

**Choice:** The codec writes YAML with 2-space indentation, keys in spec order (not alphabetical), multi-line `content`/`unresolved`/`reason` as literal block scalars (`|`), timestamps as RFC 3339 UTC with `Z`, optional fields omitted when unset, no document markers. Encoding the same object twice yields identical bytes. New objects are written with create-exclusive semantics; an existing file is never rewritten by any command except `claim retract` (see D6) and `particular define` (particulars are mutable per spec — "create or update").

Spec field order:

| Object | Order |
|---|---|
| particular | `id, type, uri, label, aliases` |
| claim | `id, type, subject, content, source, context, timestamp, confidence, retracted` |
| synthesis | `id, type, subject, content, inputs, unresolved, produced-by, method, timestamp, context, confidence, retracted` |
| `source` | `author, harness, model, document` |
| `context` | `scope, topics` |
| `inputs[]` | `id, role, weight` |
| `retracted` | `timestamp, reason, source, superseded-by` |

**Why:** Humans read PR diffs. Noise-free diffs require byte-stable output; create-only writes guarantee that a PR touching an existing claim file is, by construction, a retraction.

### D5. Workspace: `dkf.yaml` as marker and config

**Choice:** `particulars init` creates:

```
<root>/
  dkf.yaml
  particulars/
  claims/
  syntheses/
  index.yaml
```

```yaml
# dkf.yaml
format: dkf/0.1
workspace:
  id: 019196a5-8b4c-7def-8abc-0123456789ab     # UUIDv7, minted at init
  base-uri: https://example.com/particulars/   # optional
defaults:
  scope: personal
  source:
    author: ben
    harness: claude                            # optional
```

Discovery precedence: `--workspace <dir>` flag → `DKF_WORKSPACE` env → walk up from the working directory until a directory containing `dkf.yaml` is found. No workspace → exit code 5.

Provenance defaults precedence for `source.author|harness|model`: explicit flag → `DKF_AUTHOR|DKF_HARNESS|DKF_MODEL` env → `dkf.yaml` defaults. The env vars exist so a harness can configure attribution once rather than per call.

**Why:** The spec has no config file, but URI minting (D7) needs a stable workspace identity and optional base URI, and the agent needs a default `author`. Making the config file the discovery marker avoids inventing a second one.

**Alternatives:** Use `index.yaml` as the marker — but it is derived (D9) and may legitimately be absent; `.dkf/` hidden dir — less reviewable.

### D6. Retraction: append-only block on the object file

**Choice:** `claim retract <id>` appends to the existing claim/synthesis file:

```yaml
retracted:
  timestamp: 2026-08-21T09:12:00Z
  reason: "Port is 8443, not 443 — deploy/config.yaml:12"
  source:
    author: ben
  superseded-by: clm_0192…     # optional, must exist
```

Rules: the block is the only legal modification to an existing claim/synthesis file; it is appended to the existing bytes (the file is then re-parsed to confirm validity and restored on failure); retraction is permanent — there is no un-retract; reinstatement is a new claim or synthesis citing the retracted one. Retracting a synthesis is allowed (it is a claim). Retracting an already-retracted object is an error. The index records `retracted: true`. `recall` excludes retracted objects unless `--include-retracted`.

**Why over a separate `ret_` object:** the consumer most likely to misuse a retracted claim is one that opens only the claim file — the marker must live there. It matches the spec wording ("mark a claim retracted"), keeps to three object types, and gives the best PR diff (the claim with its retraction beneath it). Future signing is defined over the object minus `retracted` and `signature`, so the append does not invalidate a signature.

**Trust cascade:** retracting an input does *not* modify syntheses that cite it (they are immutable and were already reasoned). Instead `conflicts` reports such syntheses as **stale**. Creating a new synthesis that cites a retracted input is permitted (that is how reasoning about a retraction is recorded) and produces a warning.

### D7. Particular URIs: minted from a slug under a base

**Choice:** `particular define --label L [--uri U] [--alias A]…`. If `--uri` is absent the tool mints `<base><slug>` where `base` is `workspace.base-uri` if configured, otherwise `urn:dkf:<workspace-id>:`, and `slug` is the label lowercased, Unicode-folded to ASCII, non-alphanumerics collapsed to single hyphens, trimmed. `define` is idempotent on URI: an existing particular is updated (label replaced, old label and new aliases unioned into `aliases`) and returned with `created: false`.

Rules: a URI may be changed only while a particular has never been published (no publish exists in v1, so the rule is latent); claims reference particulars by `par_` ID, never by URI, so rehoming is cheap.

**Why:** Agents are bad at re-inventing the same URI across sessions; deriving it from the label makes "same label → same particular" automatic, and `resolve` + aliases cover variant phrasings. Explicit `--uri` lets the agent use Wikidata/ORCID/GitHub URLs for things that have one, as the spec prefers. The `urn:dkf:` fallback is globally unique (workspace UUID) and never resolvable, which is honest for unpublished knowledge.

**Alternatives:** `urn:uuid:` per particular — unique but not idempotent on label. `tag:` URIs — standard, but require a DNS/email authority, i.e. configuration, which is what `base-uri` already is.

### D8. Conflict detection is structural

**Choice:** For a particular P:

- **current** = the most recent (highest ID) non-retracted synthesis with `subject == P`, or none.
- **reconciled** = transitive closure of `current.inputs`.
- **unsynthesised** = non-retracted claims and syntheses about P that are neither `current` nor in `reconciled`.
- **stale** = non-retracted syntheses about P with at least one directly retracted input.

`conflicts` reports P when (current exists and unsynthesised is non-empty) or (no current and |unsynthesised| ≥ 2) or stale is non-empty, ordered by `|unsynthesised| + |stale|` descending (the spec's "suggested synthesis priority"). `--fail-on-conflicts` exits 4 for CI use.

**Why:** The spec says "the LLM reasons; this tool stores." Whether two statements actually contradict is a semantic judgement the agent makes better than any heuristic; what the tool can know exactly is which statements have not been reconciled into the current belief. This keeps the binary free of any model dependency.

### D9. `index.yaml` is committed but derived

**Choice:** Every mutating command updates `index.yaml` incrementally; `particulars index` rebuilds it from the object files; `particulars index --check` exits 4 if the committed index differs from a rebuild. Entries follow the spec shape plus additive fields (`scope`, `topics`, `retracted: true`, `timestamp`) that make `recall --topic` and retracted filtering possible without opening files. Entries are sorted by ID so output is stable.

**Why:** Two branches adding claims will both touch `index.yaml`; if it is authoritative, every merge conflicts and a human must hand-merge YAML. If it is derived, conflicts are resolved by `particulars index`. It stays committed because public consumers (future federation) fetch it over HTTP. The spec's own description — "enables recall … without parsing every file" — reads as an optimisation.

### D10. Agent-facing CLI contract

- Never prompts, never reads a TTY. Multi-line content is accepted via `--content-file <path>` or `--content-file -` (stdin pipe).
- `--json` on every verb emits exactly one JSON object on stdout; in JSON mode errors are a JSON object on stderr. Text mode is for humans and may change; JSON shape is the contract.
- Exit codes: `0` success · `1` runtime error · `2` usage error · `3` not found (unresolvable subject/ID, `resolve` with no match) · `4` check failed (`validate`, `index --check`, `--fail-on-conflicts`) · `5` no workspace.
- Verbs: `init`, `particular define|resolve`, `claim assert|retract`, `synthesis create`, `recall`, `conflicts`, `lineage`, `index`, `validate`, `version`.

### D11. `validate` as a first-class verb

Not in the spec's tool list, but the most useful thing a reference implementation offers other implementers and the natural PR check. Checks: `dkf.yaml` well-formed; every file parses; ID ↔ filename ↔ directory ↔ `type` agree; required fields present; enums (`scope`, `role`, `weight`) valid; `confidence` in [0,1]; timestamps RFC 3339; `subject`, `inputs[].id`, `superseded-by` resolve; no cycles; `unresolved` non-empty on syntheses; particular URIs unique; index matches rebuild. Findings are `{severity, path, code, message}`; any `error` → exit 4.

## Risks / Trade-offs

- [Spec changes field names or ID format before v0.1] → All format knowledge lives in `internal/dkf`; a `format:` field in `dkf.yaml` and `index.yaml` allows a future migration verb. Accept churn; this is the point of a reference implementation.
- [Appending YAML text could corrupt a hand-edited file] → Retract re-parses after append and restores the original bytes on failure; `validate` catches anything that slips through.
- [Incremental index updates drift from files after manual edits or merges] → `index --check` in CI; `index` rebuild is always safe; no command trusts the index for correctness, only for speed.
- [Slug collisions: two different things with the same label map to one URI] → `define` returns `created: false` so the agent sees it hit an existing particular; the agent-facing instructions tell it to use a more specific label or an explicit `--uri`.
- [UUIDv7 diverges from whatever the spec finally chooses] → ID parsing is lenient on read (any `<prefix>_<opaque>` is accepted), strict only on mint.
- [Structural conflicts over-report (extensions counted as "unsynthesised")] → Intended: the agent decides, and a synthesis that merely incorporates an extension is cheap. `--fail-on-conflicts` is opt-in.
- [Large workspaces make full-file loads slow] → Out of scope for v1; the index exists precisely so a later SQLite cache can be layered under `store` without format change.

## Migration Plan

Greenfield; no migration. Rollback is deleting the binary — the YAML remains readable by anything.

## Open Questions

- **How does the agent learn the verbs?** Options: ship a `SKILL.md`/instructions file in the repo that harnesses can install; embed `particulars help --agent` that prints a compact usage guide; both. Leaning both; decide in the follow-on change alongside `serve --mcp`.
- **Does `recall` need text search?** v1 offers subject and `--topic` filtering only. Revisit if agents are observed `grep`ping the workspace routinely.
- **Should syntheses be required to share `subject` with their inputs?** Not enforced in v1 (cross-particular reasoning seems legitimate). Possible `validate` warning later.
- **Stale detection depth.** v1 checks direct inputs only; transitive staleness may be wanted.

## Spec Feedback (to raise upstream)

1. **ID format:** propose `<prefix>_<uuidv7>` (RFC 9562) over truncated ULIDs; define ID timestamp vs `timestamp` field semantics.
2. **Retraction representation:** specify the append-only `retracted` block `{timestamp, reason, source, superseded-by?}`, its permanence, and that signatures exclude it.
3. **`superseded-by`:** bless an optional lightweight correction pointer for typo-grade errors.
4. **URIs:** soften "globally resolvable" to "globally unique, resolvable once published"; bless `urn:dkf:<workspace-id>:<slug>` (or a `tag:` convention) for unpublished particulars.
5. **Workspace config:** propose `dkf.yaml` as the workspace marker/config convention.
6. **`index.yaml`:** state explicitly that it is derived and regenerable, and allow additive fields.
7. **Merge record:** `particular_merge` "produces a merge record" — an undefined object type.
8. **Conflict semantics:** document the structural definition (unsynthesised / stale) as the baseline consumers can rely on.
9. **`source` required fields:** the spec does not say which `source` fields are mandatory; this implementation requires `author`.
