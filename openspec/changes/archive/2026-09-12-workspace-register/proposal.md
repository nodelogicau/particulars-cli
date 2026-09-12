## Why

A model choosing a subject or a topic cannot see what the workspace already uses, so it mints. Two dogfood workspaces of comparable size, both served the same guidance, show the cost:

| | particulars-knowledge | particulars-personal |
|---|---|---|
| claims | 174 | 137 |
| particulars | 9 | 39 |
| topics | 46 | 195 |
| topics used exactly once | ~5 (11%) | 110 (56%) |
| particulars with no synthesis | ~1 | 32 of 39 |
| conventions document | `dkf.md`, 5,623 bytes | none |

The topic vocabulary in the second is larger than its claim count, and its near-misses are the compounding the skill already forbids: `adolescent-psychology` beside `psychology`, `analytic-philosophy` beside `philosophy`, `being-in-the-world` beside `being`, `digital-wellbeing` beside `wellbeing`. This matters more than particular duplication because **topics have no repair path** — claim files are append-only, so the skill's own rule is to retire a tag, never rewrite it. Particulars have `particular merge`; a topic vocabulary is permanent.

Guidance is not the missing piece. `serve --mcp` already writes the entire skill body into its `initialize` instructions, so both surfaces deliver identical norms. What differs is that one workspace carries a conventions document whose first two sections enumerate its nine particulars and its topic vocabulary — a registry, hand-typed. That file opens by saying it holds "only what the tool cannot know", and then spends both sections on what the tool knows exactly and more accurately than prose can: which particulars exist, which tags are in use, how often. It already says "Nine particulars exist" as a hand-maintained count.

The CLI can generate that, always current, and put it where the choice is made.

## What Changes

- **A new `particulars register` verb** reports what the workspace already uses: each particular with its label, aliases, whether its URI is global or workspace-minted, its claim count, and its unsynthesised count; then the topic vocabulary with per-topic counts, singletons marked. Deterministically ordered, `--json` like every other verb.
- **`serve --mcp` carries the register in its `initialize` instructions**, after the skill body and before the conventions document, so a human's prose keeps the last word and can explain or override what the generated section states. This is the same channel that already delivers conventions for exactly this reason: an MCP-only client never reads the repository.
- **The register is size-bounded and truncates by usefulness.** A cap, with the highest-count entries kept and a line naming how many were omitted, so the most reusable names survive truncation. At present sizes the whole thing is about 3 KB for either workspace.
- **The skill's loop gains one step before the subject is chosen.** Today it opens at `particular resolve "X"`, by which point X is already decided, and it can never reveal that `psychology` exists or that a sibling particular has eight claims. The register is read once per session; the existing per-subject loop is unchanged.
- Reporting the unsynthesised count per particular puts the other half of curation in the same view. `conflicts` already computes it, already ranks it, and is already ignored because nothing invokes it at a moment that matters.

Nothing is **BREAKING**. No object file, index, or existing verb changes shape; the instructions gain a section.

## Capabilities

### New Capabilities
- `workspace-register`: what the register contains, how it is ordered, how it is bounded, and the guarantee that it is derived from the files rather than authored.

### Modified Capabilities
- `mcp-server`: the `initialize` instructions requirement gains the register between the skill body and the conventions document.
- `skill-distribution`: the skill body gains the register step, which changes the embedded text and therefore the installed copy.

## Impact

- New `internal/cli/cmd_register.go` and a query in `internal/query`; `internal/mcp/server.go` `instructions()`; `skills/particulars/SKILL.md` and the regenerated `.claude/skills/particulars/SKILL.md`; tests; `docs/mcp.md`; `CHANGELOG.md`.
- Not in scope, and deliberately: a near-miss warning at `define` and `--topic` time. The register makes the alternatives visible at the moment of choosing, which is the cheap half. Whether a warning or a refusal is also needed is a separate question, better answered after watching whether visibility alone moves the numbers.
