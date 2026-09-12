## Context

`Server.instructions()` assembles three pieces today: a header naming the bound workspace, `skill.Body()`, and the conventions document when one resolves, capped at 16 KiB with a rune-boundary cut. The comment there states the reason the conventions travel at all — an MCP-only client never reads the repository — which is the same reason the register must travel rather than being a verb the model is merely told about.

Both dogfood workspaces receive that identical skill body. The one with a hand-written registry in `dkf.md` has 46 topics with about a ninth used once; the one without has 195 topics with more than half used once. The registry section of `dkf.md` was written on 2026-09-05, fifteen days after that workspace began, so it documented a discipline rather than creating one. Its preventive value is untested — which is an argument for making it free and automatic rather than for assuming prose would have worked.

There is a second, mechanical asymmetry. In a shell the check loop is one call: resolve, recall, conflicts and topics chain in a single command. Over MCP each is a separate round trip with schema ceremony, so checking before asserting costs roughly four actions instead of one. A model holding a single fact it wants to record will pay one and not four. Delivering the register with the instructions removes that cost entirely rather than asking guidance to overcome it.

## Goals / Non-Goals

**Goals:**
- The names a workspace already uses are visible at the moment a subject or a tag is chosen, on both surfaces, without a tool call.
- The register is derived, so it cannot go stale the way `dkf.md`'s hand-typed "Nine particulars exist" can.
- One view carries both halves of curation: what exists to reuse, and what has accumulated claims without a belief.

**Non-Goals:**
- Warning or refusing at `define` or `--topic` time. Visibility first; enforcement is a separate change with its own evidence.
- Any merge, rename, or retirement mechanism for topics. The format makes tags permanent and this change does not challenge that.
- Replacing or editing the conventions document. The tool never writes a human's prose.
- Solving over-coarse particulars. The register surfaces the claim counts that make coarseness visible, but the grain question is judgment and stays in prose.

## Decisions

### The register is assembled at delivery, never written into `dkf.md`

The alternative is a generated section inside the conventions document, marked by ownership comments the way the `agents-md` skill preset already does. Rejected: `dkf.md` is a reviewed, human-authored artifact that travels to agents as written, and a tool that rewrites part of it produces diff noise on every claim added and a merge conflict on every branch that adds two. Keeping the generated and authored halves in separate files also lets the prose do what it is uniquely for — saying why this split, which tag won, where releases are filed — with the facts beneath it rather than tangled into it.

### It sits between the skill body and the conventions

The skill states the general discipline, the register states this workspace's facts, and the conventions state the human's judgment about them. Reading in that order means a convention that contradicts a generated fact — "these two tags are being consolidated; prefer the first" — is read last and wins, which is the right precedence for a file a person maintains deliberately.

### Truncation keeps the most-used entries

The conventions cap truncates at a byte offset because prose has no ranking. A register does: an entry used forty times is more likely to be the one worth reusing than an entry used once, and the singleton tail is exactly what a model should not be copying anyway. So the cap drops the tail, keeps the head, and states the count omitted. A workspace large enough to truncate is a workspace whose head is the useful part.

### Unsynthesised counts ride along

`conflict_detect` already computes and ranks them. The personal workspace has thirty-two particulars with no synthesis at all, and the report that would say so is correct, ranked, and never run, because a capture session ends when the fact is written. Putting the number beside the claim count costs one column and turns the register into the prompt that was missing, without inventing a second report.

### The verb exists as well as the instructions

MCP clients get the register whether or not they ask. Shell users need something to run, and the skill needs something to name. The same function serves both, which also keeps the two surfaces from drifting — the failure this change exists to address was, in part, two surfaces that looked identical and were not.

## Risks / Trade-offs

- [The register is another thing in an already long instructions blob, and could be skimmed past] → It is short, tabular and specific, where the skill body is prose. If it proves ignorable the next lever is the warning at mint time, which this change deliberately leaves on the table.
- [Injecting names could bias a model toward reusing a particular that does not fit] → That is the intended pressure, and it is the recoverable direction: a wrongly reused particular is visible in recall and fixable, while a wrongly minted topic is permanent.
- [It scales only to the low thousands of particulars] → True, and bounded by the cap. A workspace past that has the sharding problem first.
- [Evidence is two workspaces and a correlation, with the registry written after the discipline it describes] → Stated in the proposal rather than glossed. The change is cheap enough that being wrong costs a section of text; the alternative, enforcement at mint time, is the one that would want stronger evidence first.
