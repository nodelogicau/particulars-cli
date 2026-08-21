# Changelog

## v0.2.1 — the skill ships in the binary

- New `skill` verb: `skill show` prints the embedded agent skill; `skill install`
  writes it to `./.claude/skills/particulars/SKILL.md` (default), `~/.claude/…`
  (`--user`), or `--dir <path>`. The text is stamped with the binary version and
  an ownership marker; files the tool did not write are refused without `--force`;
  `--check` verifies without writing (exit 4 on drift), ignoring the stamped version.
- The repository's own `.claude/skills/particulars/SKILL.md` is now generated
  (`make skill`) and verified in CI.
- Docs: one-command agent setup in the README; a sample Claude Code
  `SessionStart` hook in `docs/examples/claude-settings.json`.

## v0.2.0 — align with the resolved DKF v0.1 draft

Implements the decisions recorded in
[nodelogicau/particulars@27743db](https://github.com/nodelogicau/particulars/commit/27743db)
(2026-08-21), which resolved all ten feedback issues from this implementation.

**Breaking (JSON and on-disk shape of syntheses)**

- Syntheses carry `source` (author?, harness, model?, document?) in place of
  `produced-by`. `synthesis create` writes `source`; `--json` output uses `source`.
  Files written by v0.1.x are still read (their `produced-by` becomes `source`) and
  `validate` reports them as `legacy_produced_by` warnings. No rewrite.

**Relaxed**

- A `source` needs at least one of `author` or `harness` — on claims, syntheses,
  retractions, and merges. Agent-only provenance is valid. (v0.1.x required `author`.)

**New**

- Merge records: `particular merge <a> <b>` writes `merges/mrg_<uuidv7>.yaml`.
  Non-retracted merges define equivalence classes that `recall`, `conflicts`, and
  `lineage` operate over; `recall --json` lists `class`, `conflicts` lists `members`.
- Top-level `retract <id>` for claims, syntheses, and merges. `claim retract` remains
  as an alias.
- `synthesis create --author`, `--document`.
- `recall` entries carry `source` and `unsynthesised`; `lineage` shows
  `superseded_by` on retracted nodes.
- `validate` findings: `conflicting_provenance`, `invalid_merge`, `invalid_base_uri`
  (errors); `legacy_produced_by`, `legacy_id`, `unknown_merge_uri`, `duplicate_merge`
  (warnings). `stale_synthesis` is now transitive.

**Changed**

- `current` synthesis is chosen by `timestamp` then id (was id only).
- `stale` includes syntheses citing a retracted input transitively (was direct only).
- `workspace.base-uri` must end in `/`: `init` appends it and says so; an existing
  `dkf.yaml` without it fails to open. `MintURI` no longer special-cases `#`/`:`.
- `init` creates `merges/`. Identifier grammar accepts `mrg_`.

## v0.1.1

- `topics` verb; agent-facing `skills/particulars/SKILL.md`; issue links in
  `SPEC-FEEDBACK.md`.

## v0.1.0

- Initial release: `init`, `particular define|resolve`, `claim assert|retract`,
  `synthesis create`, `recall`, `lineage`, `conflicts`, `index`, `validate`.
