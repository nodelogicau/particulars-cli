## Context

Both define front ends mint the URI themselves, then call `Workspace.UpsertParticular(uri, label, aliases)`:
- the CLI at `internal/cli/cmd_particular.go`
- MCP at `internal/mcp/tools.go`

`UpsertParticular` looks up `g.ParticularByURI(uri)`. On a hit it replaces the label and keeps the old label as an alias; on a miss it creates a particular. It doesn't know whether the URI was minted, and it doesn't look through merge records. The empty-slug refusal already exists in both front ends as a usage error that comes before the lookup.

Upstream `particular-uris` (nodelogicau/particulars#64) now requires a label match across the merge class before a minted URI resolves. Without a match the call is refused. On the minted path the label never changes.

## Goals / Non-Goals

**Goals:**
- The collision rule lives once in `internal/store` and is shared by the CLI and MCP.
- A refusal writes nothing.
- The explicit-URI behaviour stays byte-for-byte as it is today.
- Minted define resolves through merge records instead of creating a second particular at a merged URN.

**Non-Goals:**
- Detecting or repairing particulars already relabelled by a past collision.
- Resolving a minted define by label or alias when the minted URI names nothing. For example, `define --label C-lang`, where `C-lang` is an alias of the particular at `…:c`, still creates a new particular at `…:c-lang`. Upstream doesn't ask for this, and it would make minting depend on label lookup. Upstream's *Matching an existing alias* scenario doesn't say what URI the particular carries. Our delta pins it to `…:c-lang`, the only reading under which it holds, and we will raise this upstream.
- Changing the slug algorithm or any URI that mints today.

## Decisions

**D1. A new store method, `DefineMinted(uri, label, aliases) (*dkf.Particular, bool, error)`, alongside `UpsertParticular`.** The front ends call `DefineMinted` when they minted the URI and `UpsertParticular` when `--uri` was given. The alternative was a `minted bool` parameter on `UpsertParticular`. The two paths differ in every branch (match check, never relabel, union-only aliases, merge-class lookup), so a flag would split the function in two anyway. Separate methods keep the explicit path untouched and easy to see.

**D2. Look up the merge class with `g.MergedURIOwners(uri)`.** It already returns the class's particulars, sorted by id, for a URI carried by a particular or named in a non-retracted merge, and returns empty when the URI is unknown. Empty means create. Otherwise, match the trimmed label against each member's `label` and `aliases`. Choosing among matches: prefer the member carrying the URI (`g.ParticularByURI(uri)`), else the lowest id, which is the first in the sorted order. The refusal names the URI's carrier if there is one, else the lowest id.

**D3. Case folding uses `strings.EqualFold`.** It applies Unicode simple case folding, which is what resolve already uses for labels. Full folding (`golang.org/x/text/cases.Fold`) differs only for multi-rune folds like `ß`/`SS`. Those labels slug differently, so they never reach this comparison through the same minted URI. Not worth the extra call per comparison.

**D4. The refusal is a typed store error, `*store.MintedCollisionError{Label, Existing *dkf.Particular, URI}`, which `errors.Is`-matches a new sentinel `store.ErrMintedCollision`.** `apperr` can't be imported by `store` (the dependency runs the other way), so `apperr.Classify` maps the sentinel to code `conflict`, exit 1, beside `ErrAlreadyExists`, and `IsDomain` recognises it. A dedicated code such as `slug_collision` was considered. `conflict` already means "refused because of what is already there", and agents read the message for the id. A new code would widen the error vocabulary for one case. The message:

  > `label "C#" mints urn:dkf:W:c, which names par_… ("C"); for a different thing pass --uri, or refer to the same thing by its uri or id`

  The MCP front end swaps `--uri` for `uri`. The message is built in the front ends from the typed error's fields, so each names its own flag.

**D5. Write only when something changes.** If no supplied alias is new, return the existing particular with `created: false` and write nothing. That keeps a repeat define from touching the file or the index. Today's `UpsertParticular` always rewrites. That doesn't change here (D1).

**D6. Wording.** The `particular_define` description and the skill's *Rules for particulars* drop "the same label always resolves to the same particular". They now say:
- Labels that slug alike are refused, not merged.
- Labels that are symbols or entirely in a non-Latin script need an explicit URI.
- On a refusal, use the named id or URI if it's the same thing, or pass a URI for a different thing.
- `define` without a URI never renames; pass the URI to rename.

## Risks / Trade-offs

- [Scripts that renamed a particular by re-running a minted define with a new casing now see no rename] → Documented as **BREAKING** in the proposal and release notes. Rename with `--uri <existing-uri>`.
- [An agent that hits the refusal may loop or invent a URI] → The message names the existing id and says what to do in each case. The skill gives the same instruction.
- [Concurrent CLI processes racing a define] → Same as today. The check and the write happen within one `Load` and write, and MCP serialises under `s.mu`. No new window.
- [The installed skill copy drifts from the source] → Regenerate with `skill install --force` from a binary built from HEAD. CI checks the body.

## Migration Plan

No data migration. Ship in the next minor release, with a release note under "Changed" that names the conflict refusal and the never-relabel rule. Rollback means reverting the release. No files are written in a new shape.

## Open Questions

- Filed upstream as nodelogicau/particulars#65: *Matching an existing alias* in `particular-uris` should state the particular's URI (see Non-Goals).
