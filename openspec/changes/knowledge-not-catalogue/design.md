## Context

The register failure happens at one moment: a model holding a document chooses what to pass as `particular_id` and what to write as `content`. Everything else — instructions, skill, conventions — may or may not be in context by then; the tool and parameter descriptions always are. The change is therefore mostly text placed where the choice is made, with one mechanical whisper after the fact. The existing precedent for pinning agent-facing prose in specs is the instructions requirement's "Recall **before** you assert" phrase; the precedent for a smell that is never an error is `quoted_source` / `unverified_document`.

## Goals / Non-Goals

**Goals:** every surface a model reads at assert time states the same register; a workspace can measure whether it worked; the catalogue shape stays representable for the cases where it is genuinely meant.

**Non-Goals:** any validity rule against document-subjects; semantic detection of cataloguing (only the URL-in-content proxy); changes to what Fable-like models already do.

## Decisions

**D1. The tool surface is the primary lever, one clause per site.** `claim_assert`'s description gains: content states the fact about the world — what was read goes in `source.document`, and if the content names an article or URL, the source has leaked into the claim. `particular_id`'s schema text gains "the thing in the world the fact is about — never the document or feed it was read in". `content`'s schema text gains "about the world, not about a document". `particular_define`'s description replaces "a GitHub URL" with identity examples (a person's or project's page) and gains "a particular is a thing in the world, not a document being read; what you read belongs in claim_assert's source.document". One clause each: descriptions that lecture get skimmed.

**D2. The skill teaches by contrast.** One new rule bullet carrying the two litmus tests — the deletion test (strip `source.document`; a claim that then teaches nothing was a citation) and nobody-recalls-a-feed (file knowledge where someone will look) — followed by a compact ✗/✓ example: feed-as-subject with title-and-URL content versus subject `Sydney`, content the fact, document `ref`/`author`/`quote`. A worked pair steers weak models where principle does not. The rule closes with the legitimate case: a document is a subject only when the knowledge is about the document itself.

**D3. `url_in_content` is an aggregated informational note.** Detection is a scheme match (`https?://`) in claim content — deliberately narrow: `urn:` or bare hostnames in prose are not the smell being chased. Info severity, on the corpus-fact whitelist, one line with a count (uniform message, so the existing severity+code grouping suffices — no per-value keying needed). Never an error and never per-object by default: legitimate claims about endpoints exist, and the note's discovery value is spent the first time it is seen; `--notes` lists the objects. Applied to claims only, not syntheses — a synthesis quoting its inputs' evidence is not the pattern.

**D4. Conventions are the per-workspace escalation.** Docs (not spec) say: a workspace that ingests feeds should state its ingestion register in the conventions file, which `serve --mcp` already delivers. The generic text stays register-only; vocabulary-grade rules ("this workspace never makes article particulars") are workspace policy.

**D5. Effect is measured, not assumed.** `source.model` rides on every claim, so the catalogue rate per model is `recall --json` filtered by the same URL proxy, grouped by model. The change's success criterion is that rate falling for the misbehaving model, checked in the dogfood-adjacent workspace that observed it.

## Risks / Trade-offs

- [False positives on endpoint/URL-subject claims] → info, aggregated, never affects exit codes; the message names the register rather than accusing.
- [Description bloat degrades every call] → one clause per site; the contrast example lives in the skill, not the descriptions.
- [A model that ignores all text] → conventions can restate per workspace; beyond that, measurement (D5) tells you the model needs replacing rather than prompting.

## Migration Plan

Text plus one read-only finding; nothing rewrites. Existing catalogue-shaped claims surface as one aggregate note; cleaning them up (retract + reassert) is the workspace owner's call, not the tool's.

## Open Questions

- None.
