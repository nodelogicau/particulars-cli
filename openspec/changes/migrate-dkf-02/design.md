## Context

After stages 1 and 2, a workspace whose `dkf.yaml` says `dkf/0.2` is fully supported:
- **Paths:** `Workspace.Path` derives them, and `Load` reads month directories and reports misplaced files.
- **Index:** `RebuildIndex` writes the root, sealed segments and legacy, preserving unknown entry types and fields from a committed index it can parse.
- **Guards:** they refuse writes into sealed months.

**What's missing** is a way to get there. `init` writes `dkf/0.1` (`dkf.WriteFormat`), `dkf.yaml` is only ever written by `Init` (`encodeConfig` of the struct), and there's no verb that moves files.

**The three dogfood workspaces**, measured on 2026-11-03:

| Workspace | Objects | Git | CI | Open PRs | Sealed at migration |
|---|---|---|---|---|---|
| `particulars-knowledge` | 272 | yes | `dkf-check.yml` pinned to v0.12.0 | none | August and September |
| `particulars-personal` | 375 | yes | none | – | – |
| `particulars-feeds` | 207 | **no** | – | – | – |

Stage 2's simulation on a copy of the knowledge workspace sealed August (205 entries) and September (32), passed `--check` and `validate`, and behaved correctly on a write and a retraction.

## Goals / Non-Goals

**Goals:**
- **Exact.** Migration changes nothing an object asserts: every file is renamed, byte-identical, so git records renames and every signature still verifies.
- **Resumable.** An interrupted migration is never silent. The workspace is either untouched, or declares `dkf/0.2` while still containing flat files, which every verb then reports as misplaced. Running `migrate` again completes it.
- **Default.** `init` creates `dkf/0.2`.

**Non-Goals:**
- Migrating back from `dkf/0.2` to `dkf/0.1`. Upstream defines no reverse migration, and git is the undo for a workspace in a repository.
- Rewriting files to canonical form during migration. A migration that re-encodes would change bytes and break signatures.
- Exposing `migrate` over MCP. It's a one-off human operation, like `init`.
- Migrating a workspace with unreadable files. It refuses instead (D2).

## Decisions

### D1. Order: format first, then files, then index

`migrate` works in four steps:
1. **Plan:** load the graph, compute every move, check the preconditions (D2).
2. **Rewrite `format`** in `dkf.yaml` to `dkf/0.2`.
3. **Rename each file** to its derived path, creating month directories as needed.
4. **Rebuild the index** in the sharded shape.

The point of this order is that a crash can't hide anything.
- **Format last** would be dangerous. A crash between moves and the format edit would leave a `dkf/0.1` workspace with files in subdirectories, and the flat loader skips subdirectories without a word, so objects would silently vanish from every verb.
- **Format first** makes the half-way state loud. A `dkf/0.2` workspace with files still at the top of a type directory reports each one as `misplaced_file`, and verbs refuse until it's fixed.

**Resuming** is the same verb. On a `dkf/0.2` workspace, `migrate` moves every top-level object file in a type directory to its derived path, then rebuilds the index. With nothing to move and a clean `index --check`, it reports "already dkf/0.2" and exits 0.

A resumed run can't load the graph through `Load`, because `Load` refuses misplaced files. So the plan reads the top-level files of each type directory directly: parse each one, check its id against its file name, and derive its path. That's the same per-file check `loadFile` already makes.

### D2. Preconditions refuse before anything is written

`migrate` refuses, listing every offending file, when any of these holds:
- **A file fails to load** (`parse_error`, `id_mismatch`, `type_mismatch`). Left in place it would become a stray file under `dkf/0.2`, and moving it to a derived path built from a name that disagrees with its id would put it in the wrong place.
- **A type directory has a subdirectory holding a `.yaml` file**, other than a month or `legacy` directory on a resumed run. `dkf/0.1` ignores such files, but `dkf/0.2` would report them as misplaced. The user decides what they are.
- **A target path already exists with different bytes.** The same bytes mean an earlier interrupted run already copied the file, so the source is removed and the run continues.

`--dry-run` runs the plan and the preconditions and reports. It always exits 0 unless a precondition fails, in which case it exits 1, the same as a real run.

### D3. `dkf.yaml` is edited as text

`dkf.yaml` may carry comments, key order and unknown keys (`workspace.conventions`, say) that a struct round-trip would lose.
- `migrate` replaces exactly one line matching `^format:\s*dkf/0\.1\s*(#.*)?$`, keeping any trailing comment, and writes the file atomically.
- If no such line exists, or more than one, it refuses (for example: a flow-style mapping, or a quoted value). The message says to set `format: dkf/0.2` by hand and run `migrate` again, which then resumes.

### D4. The index: rebuild with the old index's unknown rows

Before step 2, `migrate` parses the committed `dkf/0.1` `index.yaml`, keeping its unknown entry types and unknown entry fields. After the moves it runs the stage 2 sharded rebuild, seeded with those rows. Each preserved row then lands in the document its id places it: root, segment or legacy.

The flat file's `retracted: true` flags are dropped. In `dkf/0.2` retraction lives in the lists, and the rebuild regenerates them from the files.

No sealed segment can exist before migration, so this first rebuild writes every one of them.

### D5. `init --format`

`init` takes `--format`, accepting `dkf/0.2` (the default) or `dkf/0.1`, and refuses anything else as a usage error. `dkf.WriteFormat` becomes `dkf/0.2`.

The README explains when to choose `dkf/0.1`: a workspace whose CI or collaborators pin a binary older than this release. Such binaries refuse `dkf/0.2` by name, so the failure is loud, not silent.

### D6. Branches open across a migration

A branch created before migration adds flat files. Once merged after migration, they're stray files in a `dkf/0.2` workspace, so `validate` fails with `misplaced_file`.

**Remedy:** rebase the branch onto the migrated main and run `particulars migrate` on the branch. That's resuming (D1), which moves the stray files to their derived paths. If those paths are in a sealed month, `validate` then reports `late_merge`, and the remedy is the usual one: re-mint.

The docs advise merging or closing open branches before migrating. The knowledge workspace has none open.

### D7. Skill line

`SKILL.md`'s git section gains one sentence: if `validate` reports `late_merge`, the branch was open across two month boundaries; re-mint its objects (assert them again, and delete the never-merged files) and do not touch the index segments.

The skill doesn't mention `migrate`, so `cliTools` doesn't change. The installed copy (`.claude/skills/particulars/SKILL.md`) is regenerated with `skill install --force` from a binary built from HEAD, which is the check CI enforces.

## Risks / Trade-offs

- **[A migration interrupted mid-way]** It is loud by D1's order, and `migrate` resumes it. In a git repository, `git checkout -- . && git clean -fd` is the alternative.
- **[`particulars-feeds` has no git]** A failed migration there can't be undone by checkout. The dogfood task copies the directory first.
- **[`init`'s new default breaks older binaries reading new workspaces]** This is marked BREAKING in the changelog. `--format dkf/0.1` is the opt-out, and older binaries refuse by name rather than misreading.
- **[Two clocks at migration time]** Migration doesn't mint or date anything, so the guards don't apply. Sealing follows the newest id in the files.

## Migration Plan

1. Merge and release as v0.20.0. The minor version carries the BREAKING `init` default.
2. Run `particulars migrate --dry-run`, then `particulars migrate`, then `validate` and `index --check`, in `particulars-knowledge` on a branch. Bump `dkf-check.yml` to v0.20.0 in the same PR, merge it after CI, then comment the result on #11 and close it.
3. Repeat step 2 for `particulars-personal` (git, no CI).
4. Repeat for `particulars-feeds` after copying it. There's no git there, so the copy is the undo.

## Open Questions

None.
