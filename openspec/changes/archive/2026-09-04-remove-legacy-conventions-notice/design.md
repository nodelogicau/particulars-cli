## Context

The notice was introduced in `align-workspace-conventions` (v0.13.1) as the migration path for the `CONVENTIONS.md` to `dkf.md` rename, explicitly scheduled for removal in the next release. Nothing else depends on the three store symbols it used.

## Goals / Non-Goals

**Goals:** honour the schedule; leave no dead code or dead spec text.

**Non-Goals:** reintroducing any reading of `CONVENTIONS.md`; changing the `conventions_invalid` warning, which is permanent spec behaviour.

## Decisions

**D1. Delete, do not deprecate further.** The symbols and the JSON field go in one step; a second grace period would contradict the promise the changelog made. Alternative considered: keep `LegacyConventions()` unexported for a future audit verb; rejected as speculative.

**D2. Docs keep the migration, lose the reminder.** The MCP guide still says the default is `dkf.md` and that a generic file is not read; the sentence saying `workspace` and `serve` will nag is removed, since they no longer do.

## Risks / Trade-offs

- [A workspace that never migrated stops being reminded] → its file was never read after v0.13.1 either; the changelog for v0.13.1 and this release both name the rename.

## Migration Plan

None. Rename `CONVENTIONS.md` to `dkf.md` if you have not.

## Open Questions

- None.
