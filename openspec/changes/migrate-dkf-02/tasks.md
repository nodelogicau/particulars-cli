## 1. Writing dkf/0.2 by default

- [x] 1.1 `dkf.WriteFormat` becomes `dkf/0.2`. `init` gains `--format` (`dkf/0.2` default, `dkf/0.1`, anything else a usage error). `init`'s `created` list and help text reflect the format. Update the `version` test (design D5)
- [x] 1.2 Tests:
  - a default `init` writes `format: dkf/0.2` and a `dkf/0.2` root;
  - `--format dkf/0.1` writes the flat index;
  - `--format dkf/0.3` exits 2 and creates nothing.

  Existing CLI tests that assume flat paths: either make them layout-agnostic or create their workspace with `--format dkf/0.1`, and list which in the PR

## 2. migrate

- [x] 2.1 `store.PlanMigration` reads each type directory's top-level files directly (parse, id against file name, derived path). It refuses on load failures, foreign subdirectories, conflicting targets, or a `dkf.yaml` without exactly one plain `format: dkf/0.1` line (unless already `dkf/0.2`). It returns the moves, the months that would seal, and the legacy ids (design D1, D2, D3)
- [x] 2.2 `Workspace.Migrate(plan)`:
  - parse and keep the old index's unknown rows;
  - edit the `format` line atomically;
  - rename each file into its month directory, treating an identical existing target as already moved;
  - rebuild the sharded index seeded with the kept rows (design D4)
- [x] 2.3 `particulars migrate [--dry-run]`: text and `--json` results (moved, sealed months, legacy ids, already-migrated), and exit codes as specified. Add it to the root help's verb list
- [x] 2.4 Tests:
  - a flat workspace with mints in two months, a retraction, a legacy id, a merge, a promotion, and an unknown index row migrates: every file byte-identical at its derived path, `dkf.yaml` differs only on the `format` line (including a comment and an unknown key), `index --check` clean, and `validate` findings equal to before apart from paths;
  - each precondition refuses and changes nothing;
  - `--dry-run` writes nothing;
  - resuming after a simulated crash (format rewritten, half the files moved) completes;
  - a second run reports already migrated and writes nothing

## 3. Skill and docs

- [x] 3.1 `SKILL.md`: add the `late_merge` line to "Working with git" (design D7). Regenerate `.claude/skills/particulars/SKILL.md` with `skill install --force` from a HEAD build, and check the skill tests pass
- [x] 3.2 README: make the layout section present `dkf/0.2` as the default, add a "Migrating a workspace" subsection (the order, resuming, branches open across a migration, `--format dkf/0.1`). Note in the `docs/examples/dkf-check.yml` pin comment that a `dkf/0.2` workspace needs v0.20.0 or later. Update `docs/review-workflow.md`'s example paths. Add a `CHANGELOG.md` Unreleased entry: `migrate`, and the **BREAKING** `init` default
- [x] 3.3 Run `go test ./...` and `golangci-lint run`

## 4. Release and dogfood (after merge, each step confirmed with the user)

- [ ] 4.1 Release v0.20.0, folding the Unreleased entries of stages 1–3 under one heading
- [ ] 4.2 `particulars-knowledge`: on a branch, `migrate --dry-run`, `migrate`, `validate`, `index --check`; bump `dkf-check.yml` to v0.20.0; open a PR, merge after CI
- [ ] 4.3 `particulars-personal`: migrate and commit
- [ ] 4.4 `particulars-feeds`: copy the directory, then migrate (no git)
- [ ] 4.5 Comment the results on #11 and close it
