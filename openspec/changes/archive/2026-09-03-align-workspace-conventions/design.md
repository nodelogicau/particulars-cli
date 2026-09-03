## Context

The v0.12.0 feature and the blessed spec differ in four places, listed in particulars-cli#8 and verified against the upstream README ("dkf.md", "What the server tells the model") and our code. The store resolves `CONVENTIONS.md` by default; `Config.Validate` refuses an invalid key, and validation runs on every workspace open; the server slices `content[:16*1024]`; no resources are exposed. The spec's governing argument, repeated twice in its text, is one-tool/next-tool consistency: a blessed filename so rules survive a change of tool, and a lenient key so a workspace opens under old and new tools alike.

## Goals / Non-Goals

**Goals:** conform to the blessed text; never deliver a file nobody meant for this purpose; keep a v0.12.0 user from losing their conventions silently; fix the byte-slice defect.

**Non-Goals:** a realpath or symlink check (the spec says lexical, explicitly); constraining the document's content; any change to tool descriptions.

## Decisions

**D1. Rename without fallback, with a notice.** `ConventionsFile` becomes `dkf.md`. A fallback to `CONVENTIONS.md` would reintroduce the silent-delivery failure the rename exists to prevent (aider's file, or any generic conventions document, delivered to every session). The migration path is a notice, not delivery: `Workspace.LegacyConventions()` reports true when the key is unset, `dkf.md` is absent, and `CONVENTIONS.md` exists at the root; `serve` prints one stderr line ("CONVENTIONS.md is no longer read; rename it to dkf.md or set workspace.conventions") and `workspace` prints the same and carries `conventions_legacy: "CONVENTIONS.md"` in JSON. Scheduled for removal in the release after this one — recorded in the CHANGELOG entry so it is not forgotten.

**D2. Lenience lives in an accessor, not in `Validate`.** `Config.Validate` keeps structural errors (`format`, `base-uri`, `defaults.scope`) and drops the conventions check. A new `Config.ConventionsPath() (rel, warning string)` applies the same lexical rule — `path.Clean(filepath.ToSlash(v))`, refused when `filepath.IsAbs`, leading `/`, `..`, or `../` prefix — and returns `("", warning)` on failure. `Workspace.Conventions()` calls it, so an invalid key behaves exactly as an absent one everywhere; `serve` and `workspace` call it once to surface the warning (`workspace --json`: `conventions_invalid: "<value>"`). Alternative considered: a `Warnings []string` on `Config` populated at load; rejected because only one key can warn today and an accessor keeps the rule and its consequence in one place.

**D3. Round up to the boundary.** The spec's two MUSTs — at least the first 16 KiB, cut only on a character boundary — can only both hold when a straddling character is included. The cut index starts at 16 KiB and advances while `!utf8.RuneStart(content[i])`, at most three bytes. Invalid UTF-8 in the file is passed through as-is (the loop terminates on any byte that is not a continuation byte). The note text is unchanged.

**D4. The resource is the file, read on demand.** Registered at startup with `sdk.AddResource` only when `Conventions()` resolves a readable document; URI `file://<absolute path>`, `name` the relative path, `title` "Workspace conventions", `mimeType` `text/markdown`. The handler reads the file at request time, so a client attaching it after an edit sees the edit, whereas instructions are read once at startup like the workspace binding — the difference is documented. A document that appears after startup is not listed; restart, as for every other startup-time binding.

**D5. Docs say what the spec now says.** `.dkf` is blessed, not an extension. `AGENTS.md` is a good value for `workspace.conventions` when the workspace directory is its own agent scope, because harnesses that read the repository then pick it up with no DKF support; it is a bad default for the same reason `CONVENTIONS.md` was.

## Risks / Trade-offs

- [A v0.12.0 user with `CONVENTIONS.md` loses delivery on upgrade] → the notice on both `serve` and `workspace`, plus the CHANGELOG; nothing is deleted and the fix is a rename.
- [Rounding up delivers 1–3 bytes over the constant] → intended; the constant is a floor, and a test with a straddling multi-byte character pins the direction.
- [Resource content can differ from instructions after an edit] → documented; instructions are a startup snapshot everywhere else too.

## Migration Plan

Rename `CONVENTIONS.md` to `dkf.md`, or set `workspace.conventions: CONVENTIONS.md` to keep the name. The notice says so. Remove `LegacyConventions()` and its notice in the following release.

## Open Questions

- None.
