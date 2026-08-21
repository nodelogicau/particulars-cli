## 1. Format Layer (`internal/dkf`)

- [x] 1.1 Add `TypeMerge`, prefix `mrg`, and the `Merge` type (`ID, URIs, Reason, Source, Timestamp, Retracted`); add a `Retractable` interface shared by assertions and merges; extend lenient and canonical id regexes
- [x] 1.2 Replace `Synthesis.ProducedBy` with `Synthesis.Source` plus an unserialised `LegacyProducedBy bool`; update `Assertion` helpers and JSON tags
- [x] 1.3 Encoder: synthesis order with `source`; merge encoding (`id, type, uris, reason, source, timestamp, retracted`); never emit `produced-by`
- [x] 1.4 Decoder: map legacy `produced-by` into `Source` and set the flag; detect both/neither; decode `type: merge`
- [x] 1.5 Validation: `checkSource` = author-or-harness; synthesis requires `source.harness`; `ValidateMerge` (two distinct URIs, source, timestamp); `ValidateRetracted` uses the new minimum; new codes `conflicting_provenance`, `invalid_merge`
- [x] 1.6 `MintURI` becomes plain concatenation; add `NormaliseBaseURI` (append `/`) and `ValidBaseURI`
- [x] 1.7 Tests: golden round-trip for synthesis-with-source and merge; legacy `produced-by` decode + flag; both/neither errors; source minimum matrix; base-uri helpers; `mrg` id minting/parsing

## 2. Store Layer (`internal/store`)

- [x] 2.1 `Config.Validate` rejects `base-uri` without trailing `/`; `Init` creates `merges/`; `NewConfig`/`Init` accept a pre-normalised base-uri
- [x] 2.2 `dirFor(TypeMerge) = "merges"`; `Load` reads merges into `Graph.Merges` and records legacy flags; `Read`/`Create` handle merges
- [x] 2.3 Union-find over URIs from non-retracted merges at load; `Graph.ClassOf(particularID) []string` (sorted, includes self) and `Graph.ClassByURI`
- [x] 2.4 `Retract` accepts merges (any `Retractable`), rejects `superseded-by` on merges; `UpsertIndex` handles merges
- [x] 2.5 Index: merge entries (`uris`, `timestamp`, `retracted`); `EntryFor` for merges; rebuild/check include `merges/`
- [x] 2.6 `CreateMerge(uriA, uriB, reason, source)`: sorted URIs, duplicate/self checks against non-retracted merges, mint `mrg_` id
- [x] 2.7 Tests: init creates `merges/`; config rejects bad base-uri; class computation (direct, transitive via foreign URI, retracted edge removed); merge create/dup/self; merge retract; index parity

## 3. Query Layer (`internal/query`)

- [x] 3.1 `CurrentSynthesis(g, class)` ordered by `(timestamp, id)`; class-aware candidate collection helper
- [x] 3.2 `Recall`: operate over the class; add `Unsynthesised bool` and `Source` to `Entry`; add `Class []string` to the result when >1 member
- [x] 3.3 `Conflicts`: per-class computation; transitive stale via memoised ancestor walk; all-particulars mode reports each class once with `Members`; keep cross-particular rule
- [x] 3.4 `Lineage`: `SupersededBy` on retracted nodes (not expanded)
- [x] 3.5 `Validate`: new errors (`invalid_base_uri`, `conflicting_provenance`, `invalid_merge`, merge referential checks, input-references-merge) and warnings (`legacy_produced_by`, `legacy_id`, `unknown_merge_uri`, `duplicate_merge`); `stale_synthesis` becomes transitive; suppress `non_canonical` when the only difference is legacy provenance
- [x] 3.6 Tests for every scenario in the delta specs for `knowledge-query`, `conflict-detection`, `validation`, `merge-records` (classes, timestamp ordering, transitive stale, superseded_by, legacy warnings)

## 4. CLI Layer (`internal/cli`)

- [x] 4.1 `init`: normalise `--base-uri`, report `normalised: true` in JSON and a line in text
- [x] 4.2 `claim assert`: author-or-harness check with an error naming both env vars
- [x] 4.3 `synthesis create`: `--author`, `--document`; write `source`; help text documents `None identified` and timestamp-ordered `current`
- [x] 4.4 New top-level `retract` verb; `claim retract` delegates to it (alias, same flags); merge targets reject `--superseded-by`; provenance minimum author-or-harness
- [x] 4.5 `particular merge <a> <b>` with resolution rules (local match, else bare URI, else exit 3), `--reason`, provenance flags; JSON result with record and per-side local ids; text output echoing argument order
- [x] 4.6 `recall`: emit `unsynthesised`, `source`, `class`; text renderer marks `[unsynthesised]`; `conflicts` text shows `members` when >1
- [x] 4.7 `lineage` text shows `[retracted → clm_Y]`
- [x] 4.8 Root help and README exit-code/verb tables updated; `version` unchanged (format stays `dkf/0.1`)
- [x] 4.9 In-process CLI tests for every new/changed scenario: agent-only assert, synthesis source shape and legacy read, retract alias and merge retract, merge verb paths (local, foreign, dup, self, ambiguous), recall across merge with `class`, conflicts transitive stale and timestamp ordering, init normalisation and bad-config open, validate codes

## 5. Documentation, Specs, Release

- [x] 5.1 README: verbs table (`retract`, `particular merge`), source rules, merge records, `None identified`, v0.2.0 JSON change note
- [x] 5.2 `skills/particulars/SKILL.md` and `.claude/skills/particulars/SKILL.md`: `retract` verb (mark `claim retract` deprecated), `particular merge`, author-or-harness, `None identified` convention, zsh quoting note for repeated `--input`
- [x] 5.3 `docs/review-workflow.md`: merges in the reviewer checklist; `legacy_produced_by` warnings on older workspaces
- [x] 5.4 `SPEC-FEEDBACK.md`: mark each item resolved with the spec decision; add the merge field-order discrepancy as a new item and raise it upstream
- [x] 5.5 Add `CHANGELOG.md` with a v0.2.0 entry naming the synthesis `source` rename and new verbs; bump `PARTICULARS_VERSION` to v0.2.0 in `docs/examples/dkf-check.yml` and `docs/review-workflow.md`
- [x] 5.6 Run `validate` against `~/IdeaProjects/particulars-knowledge` with the new binary: expect 0 errors and four `legacy_produced_by` warnings; record the result in the knowledge workspace as a claim
- [x] 5.7 Tag `v0.2.0`, verify the release assets, and confirm the published binary reads a v0.1.1 workspace
