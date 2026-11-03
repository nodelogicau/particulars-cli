## Why

Stages 1 and 2 of #11 (`read-dkf-02`, `write-dkf-02-index`; #24 and #25) taught the CLI to read, write and check a `dkf/0.2` workspace. But no `dkf/0.2` workspace exists. `init` still writes `dkf/0.1`, and nothing moves an existing workspace across.

Stage 3 closes the gap so that DKF v0.2, declared on 2026-11-03, has an implementation that produces it:
- a `migrate` verb;
- `init` writing `dkf/0.2`;
- migrating the three dogfood workspaces.

After stage 3, #11 can close and the work is releasable. As agreed on #11, no release happens before this.

The upstream reference is `workspace-layout` at the `v0.2` tag (`1366d3b`), *Migration changes nothing an object asserts*. Migration moves each object and record file to its derived path, regenerates the index in the `dkf/0.2` shape, and rewrites `format`. No id, field, `source`, `retracted` block or payload changes, so every signature that verified before verifies after. Migration is optional: a `dkf/0.1` workspace stays valid.

## What Changes

- **`particulars migrate`** moves a `dkf/0.1` workspace to `dkf/0.2`, in this order:
  1. It refuses unless every object file loads cleanly.
  2. It rewrites `format` in `dkf.yaml` with a one-line text edit, so comments and unknown keys survive.
  3. It renames each object and record file, byte-identical, to `<type-dir>/<YYYY-MM>/<id>.yaml`, or to `legacy/` for an id that isn't UUIDv7.
  4. It replaces `index.yaml` with the `dkf/0.2` root, sealed segments and legacy document, carrying over any unknown entry types and fields.

  Running it again finishes an interrupted migration, and on a fully migrated workspace it does nothing. `--dry-run` reports what it would move and seal without touching anything. The result reports the files moved, the months sealed, and any legacy ids.
- **`init` writes `dkf/0.2` by default.** `--format dkf/0.1` remains for a workspace that older binaries must read. `dkf.WriteFormat` becomes `dkf/0.2`, and `version` reports it.
- **Skill guidance.** The skill gains one line: when `validate` reports `late_merge`, re-mint on the branch. The installed copy is regenerated.
- **Docs.** The README, the example check workflow, and `docs/review-workflow.md` describe the sharded layout as the default, and explain how to migrate and what happens to branches left open across a migration. The changelog marks `init`'s new default as **BREAKING**: binaries before this release can't read a workspace that `init` now creates.
- **Dogfood, after release.** Migrate `particulars-knowledge`, `particulars-personal` and `particulars-feeds`, each in one commit or change, and move the knowledge workspace's CI pin from v0.12.0 to the release.

## Capabilities

### New Capabilities

None. Migration belongs to `workspace`.

### Modified Capabilities

- `workspace`:
  - *Workspace initialisation* gains `--format` and the `dkf/0.2` default;
  - *dkf.yaml structure* says the CLI writes `dkf/0.2`;
  - a new requirement, *Migration*, covers the verb.
- `cli-interface`: *Version verb* reports `format: "dkf/0.2"`.

## Impact

- **New code:**
  - `internal/store/migrate.go`: the plan, the moves, the `dkf.yaml` edit, and the index;
  - `internal/cli/cmd_migrate.go`: the verb.
- **Changed code:**
  - `internal/dkf/types.go`: `WriteFormat`;
  - `internal/cli/cmd_init.go`: `--format`.
- **Tests:** byte-identical moves, resuming, refusals, dry run, preserved `dkf.yaml` content, and the index after migration.
- **Other files:** `skills/particulars/SKILL.md` and its installed copy, `README.md`, `docs/`, `CHANGELOG.md`.
- **MCP:** `migrate` isn't exposed. If the skill names it, it joins the "no MCP equivalent" row in `cliTools`.
- **Outside this repository, after release:** the three workspace migrations, and the knowledge workspace's `dkf-check.yml` pin.
