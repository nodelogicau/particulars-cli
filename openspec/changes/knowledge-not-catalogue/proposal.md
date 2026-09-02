## Why

Models differ in register when ingesting sources. Pointed at an RSS feed, one model extracts the knowledge — subject `Sydney`, content "warmest winter on record", the article in `source.document` — while another records the *catalogue*: a particular for the feed, claims whose content is an article title and URL. The catalogue is valid by every rule we state ("one falsifiable statement" — "article X exists" is falsifiable) and useless by the one that matters: it is write-only knowledge. `recall Sydney` finds nothing; `recall --topic climate` finds nothing; the claim collapses the format's `claim → source → world` chain by putting the source in the world's position. Nothing in the skill, the tool descriptions, or the parameter schemas says the subject is not the medium — and `particular_define`'s description ("Prefer an existing global URI (Wikidata, ORCID, a GitHub URL)") actively reads, to a model holding an article URL, as an invitation.

This cannot be a validity rule: "the feed moved to a new URL" is a legitimate claim whose subject genuinely is the feed. It is a register to teach, on every surface a model actually reads — above all the tool and parameter descriptions, the only text guaranteed in context at the moment the model chooses a subject (several MCP clients never surface `instructions` at all).

## What Changes

- **The skill teaches the register**, in *Rules for good claims*: the subject is the thing in the world the fact is about, never the medium it arrived in; a document becomes a subject only when the knowledge is about the document itself. Two litmus tests — the *deletion test* (remove `source.document`; if the claim no longer teaches anything, it was a citation, not a claim) and *nobody recalls a feed* (file knowledge where someone will look for it) — plus one worked contrast example (✗ feed-as-subject with a title-and-URL content / ✓ fact-with-document), because a side-by-side example steers weaker models where principle does not.
- **Tool descriptions carry the same register at the point of choice.** `claim_assert`: content states the fact; what was read goes in `source.document`, who produced it in `document.author` — if the content names an article or URL, the source has leaked into the claim. `particular_define`: a particular is a thing in the world, not a document being read; the global-URI advice names identities (Wikidata, ORCID, a person's or project's GitHub page), not reading matter. Parameter schemas (`particular_id`, `content`) get one clause each.
- **`validate` whispers**: an aggregated informational note, `url_in_content`, counting claims whose content contains a URL — a catalogue smell, never an error, aggregated because its discovery value is spent on first sight and legitimate cases exist (a claim *about* an endpoint). Reported in the corpus-fact style, listable with `--notes`.
- **The conventions path is named as the per-workspace lever**: docs point out that a workspace ingesting feeds should say so in its conventions file ("extract the facts; the feed is never a subject"), which `serve --mcp` now delivers to every client.

Nothing is **BREAKING**; no file, config, or existing behaviour changes.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `validation`: the `url_in_content` informational note, aggregated.
- `mcp-server`: the tool surface's register requirement — descriptions steer the subject to the world and the source to `source.document` (pinned the way the instructions' "Recall **before** you assert" phrase already is).

## Impact

- `skills/particulars/SKILL.md` (rules block + contrast example; installed copy regenerated from a HEAD build), `internal/mcp/tools.go` and `server.go` (descriptions and schemas), `internal/query/validate.go` + `internal/cli/cmd_maint.go` (the note), `docs/mcp.md` (conventions guidance), README verb-table wording if touched.
- Measurable: since `source-as-particular`, every claim records `source.harness`/`model`, so the catalogue rate per model is observable before and after (`recall --json` filtered for URL-bearing content, grouped by model) — the change's effect can actually be checked against the misbehaving model.
- Dogfood: no existing objects change; the news workspace gains a conventions line rather than any file rewrite.
