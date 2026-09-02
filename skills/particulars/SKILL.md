---
name: particulars
description: Capture and recall knowledge as dialectical claims in a DKF workspace using the `particulars` CLI. Use when you learn a durable fact about the project, people, systems, or decisions you are working with; before acting on something you may already know; or when the user asks you to remember, recall, reconcile, or review knowledge.
license: MIT
compatibility: Requires the `particulars` binary on PATH and a DKF workspace (a directory containing dkf.yaml). Shell access required.
metadata:
  author: nodelogicau
  version: "dev"
  format: dkf/0.1
---

You are the author of a knowledge base that humans review through git pull
requests. `particulars` stores what you learn as YAML files; you do the
reasoning, it does the bookkeeping. Everything you write is a proposal until a
human merges it, so optimise for a reviewer: one falsifiable statement per
claim, evidence they can open, honest confidence, and honest `unresolved`.

Always pass `--json` and parse the result. Never prompt, never edit the YAML
files by hand, never touch `index.yaml`.

## The loop

Every time you are about to rely on or record knowledge about some thing X:

```
particulars particular resolve "X" --json        # does X exist? (exit 3 = no)
particulars recall "X" --json                    # what is already believed?
particulars conflicts "X" --json                 # what has not been reconciled?
      │
      ▼  reason about it yourself
      │
particulars claim assert …        # new fact, with evidence
particulars synthesis create …    # reconciliation of existing claims
```

Recall **before** you assert. Duplicated or contradicted claims are not errors —
contradiction is signal in this format — but an unreconciled pile is noise for
the reviewer. If what you learned extends or contradicts a `current` synthesis,
prefer a new synthesis over another loose claim.

## Setup

Find the workspace. Precedence is **`--workspace <dir>`, then `$DKF_WORKSPACE`,
then the nearest ancestor directory containing `dkf.yaml`** (or a `.dkf` pointer
at the same level) — so a `dkf.yaml` under your feet does *not* override an
already-set `$DKF_WORKSPACE`. Run `particulars workspace --json` when it matters:
`via` tells you which of the three won. A `$DKF_WORKSPACE` pointing at a
directory with no `dkf.yaml` is an error, not a fallback to the search.

If there is no workspace and the user wants one:

```sh
particulars init ./knowledge --author <user> --harness <your harness> --json
```

Set attribution once per session rather than per call:

```sh
export DKF_HARNESS=claude DKF_MODEL=<your model id>
# DKF_AUTHOR defaults to dkf.yaml defaults.source.author — the human you work for
```

If this skill is not yet installed for your harness, `particulars skill install`
does it (`--harness copilot|agents|cursor|agents-md` for others; `skill --help`).

A source needs at least one of `author` or `harness`, so an agent acting alone is
valid; syntheses always need `harness`.

**In zsh, brace every id you follow with a colon:** write `--input "${id}:thesis"`,
not `--input "$id:thesis"`. Quoting is not enough — zsh applies the `:t` history
modifier inside double quotes too, so `"$id:thesis"` silently becomes
`clm_0191abchesis` and the call fails on an unparseable input. `"${id}"` and
`"$id":thesis` are both safe; bash is unaffected. Pass one `--input` per input.

## Verbs

| Do | Command |
|---|---|
| Find a thing | `particulars particular resolve <id\|uri\|label\|alias> --json` → `{matches: [...]}`; exit 3 if none |
| Create a thing | `particulars particular define --label "<label>" [--uri <uri>] [--alias <a>]... --json` → `{particular, created}` |
| Record a fact | `particulars claim assert --subject "<thing>" --content "<statement>" --evidential observed\|inferred\|held --document <evidence> [--document-author <who>] [--topic <t>]... [--confidence 0..1] [--scope personal\|organisation\|public] --json` |
| Multi-line content | add `--content-file -` and pipe the text on stdin (instead of `--content`) |
| Reconcile | `particulars synthesis create --subject "<thing>" --input <id>:thesis --input <id>:antithesis [--input <id>:thesis:qualifying] --unresolved "<what remains open>" --content "<resolution with reasoning>" --json` |
| What is believed | `particulars recall "<thing>" [--topic <t>]... [--scope <s>] [--limit <n>] --json` → `{entries: [...]}`, oldest first, `current: true` on the live synthesis |
| Who said what | `particulars recall --author <who> --json` — everything that particular asserted (`relations: [asserted]`) or is reported for (`[reported]`); combinable with a subject |
| Across things | `particulars recall --topic <t> --json` |
| Which tags exist | `particulars topics ["<thing>"] --json` → `{topics: [{topic, assertions, particulars}]}`; check before inventing a new tag |
| Why is it believed | `particulars lineage <id> [--depth <n>] --json` → nested tree of inputs with roles |
| What needs work | `particulars conflicts ["<thing>"] --json` → `{reports: [{particular, current, unsynthesised, stale, priority}]}` |
| Show the shape | `particulars export --format mermaid --subject "<thing>" [--depth 2] [--include-retracted]` → a diagram to paste into a PR (`--format dot` for Graphviz) |
| Withdraw | `particulars retract <id> --reason "<why>" [--kind defect\|supersession\|provenance-failure] [--superseded-by <id>] --json` — claims, syntheses, merges, or promotions; append-only, never deletes (`claim retract` is a deprecated alias) |
| Share it wider | `particulars publish <id>... --scope organisation\|public [--reason "<why>"] --json` — ids only; widens, never narrows, never cascades to a synthesis's inputs |
| Same thing, two URIs | `particulars particular merge <a> <b> [--reason "<why>"] --json` — either side may be a bare URI with no local particular; undo with `retract <mrg_id>` |
| Health check | `particulars validate --json`; `particulars index --check --json` |

Exit codes: `0` ok · `1` runtime · `2` usage/ambiguous · `3` not found · `4` check failed · `5` no workspace.
On failure stderr carries `{"error": {"code", "message"}}`.

## Rules for good claims

- **The subject is the world, not the medium.** A claim is about the thing
  the fact concerns; what you read goes in `--document`. Two tests: strip the
  document — a claim that then teaches nothing was a citation, not knowledge;
  and file facts where someone will recall them — nobody recalls a feed. A
  document is a subject only when the knowledge is about the document itself
  (the feed moved; the paper was retracted).

  ```sh
  # ✗ a catalogue entry: write-only, recall finds nothing
  --subject "BBC news feed" --content "Article: 'Sydney records warmest winter' — https://bbc.com/news/123"
  # ✓ the knowledge, with the article as evidence
  --subject "Sydney" --content "Sydney recorded its warmest winter in 2026" \
    --evidential observed --document https://bbc.com/news/123 --document-author "BBC" \
    --quote "the warmest winter since records began"
  ```

- **One statement per claim.** "Service A times out above 500 rps" — not a paragraph of findings. Split compound observations.
- **Evidence or it didn't happen.** Put the file path, URL, or document in `--document`. Add `--quote "<the sentence that supports the claim>"` so a reviewer can check the claim against its source without leaving the diff, and `--hash-document` when the document is a file in the workspace, so `validate` can tell you later if it moved.
- **Quote the sentence, not the section.** A quote is reproduced verbatim inside the claim file, so it discloses its source *completely* — where a synthesis only summarises. Quote what supports the claim and nothing more, and remember that promoting the claim publishes the quote with it.
- **Never point at a line number.** `--document src/billing/cron.go:14` looks precise and rots on the next edit above line 14. A quote is content-addressed: it survives insertions and reformatting, and it says what you actually read.
- **Declare what backs the claim.** `--evidential` is required and has no
  default: `observed` — you (or a tool) actually looked; `inferred` — you
  reasoned it from other claims; `held` — nothing external backs it, it is a
  position. Do not reach for `observed` reflexively: it claims you looked, and
  a reviewer will read it that way.
- **Confidence is the probability you are wrong, inverted** — not how good the
  evidence is, and not how strongly you feel. It exists only for `observed`
  (the evidence may have been misread) and `inferred` (the reasoning may be
  invalid). A `held` claim takes no confidence — the tool refuses it — because
  a position is not on that scale; how strongly it is held belongs in
  `--content`, where reasoning lives.
- **Who told you goes in `--document-author`, never in the content.** "Jane
  said the split happened in Q2" is content "The split happened in Q2" with
  `--document "conversation with Jane, 2026-08-30" --document-author jane`
  and, ideally, `--quote` with her words. The claim stays `observed` — you
  observed the utterance — and `recall --author jane` finds it, labelled
  `reported`.
- **Pass `--author` only when the author differs from the default** (the human
  you work for, from `dkf.yaml`). It takes a particular id, URI, or name; a
  defined person is written as their URI, and an explicitly passed name that
  matches two particulars is refused. Never define a person's particular as a
  side effect — the human chooses the URI they are cited under, once, with
  `particular define --label "Their Name" --uri <chosen-uri> --alias <name>`.
- **Scope honestly.** Default `personal`. Use `organisation` for things others in the org should see; never `public` unless the user says so.
- **Backdate when recording history.** `--timestamp 2024-11-15T00:00:00Z` for a fact from a dated document; the id still records when you wrote it.
- **Topics are facets for recall, not descriptions.** A claim carries a few
  tags (2–4) from a small, stable, lowercase vocabulary — one per facet the
  workspace recalls by (a place, a domain, a system) — never a summary of its
  content. Check `particulars topics --json` before inventing one; a tag earns
  its place by being reused.
- **Compose, never compound.** `us` + `politics`, not `us-politics`;
  `billing` + `incident`, not `billing-incident`. `recall --topic a --topic b`
  is an AND, so composed tags recombine freely while a compound one can never
  be queried apart.
- **The subject is not a topic.** The particular already names what the claim
  is about; `kakapo` adds nothing over `conservation` + `new-zealand`, and a
  city is a subject, not a tag. Tag the facets a claim shares with claims about
  *other* things.
- **Retire tags, never rewrite them.** Claim files are append-only, so
  consolidating `weather` into `climate` means stop using the old tag and
  search both when it matters; the workspace's conventions file
  (`CONVENTIONS.md`, or `workspace.conventions` in dkf.yaml — delivered to MCP
  clients with the server instructions) is the place to record which tags won.

## Rules for particulars

- `define` mints the URI from the label, so **the same label always hits the same particular**. Use specific labels: "Auth Service" and "Auth Team", not "Auth".
- If `define` returns `created: false`, you hit an existing particular — check it is the one you meant.
- Things with a global identifier get it: `--uri https://www.wikidata.org/entity/Q42`, an ORCID, a GitHub URL for a repo or file.
- Ambiguous subjects exit 2 listing candidate ids; retry with the id.

## Rules for syntheses

- Roles say how inputs relate: the belief being challenged is `thesis`, the challenger is `antithesis`; supporting context is `thesis:qualifying`.
- `--method` names the kind of question at issue: `reconciliation` — the inputs
  disagreed about a fact, and this settles it (the default); `qualification` —
  each is true in a different context; `positions` — they disagree in a way no
  evidence settles. Two conflicting `held` claims still want a synthesis — a
  `positions` one. The label changes what the work is, not whether there is work.
- A synthesis declares no `--evidential`: it is backed by argument from its
  inputs, which is what a synthesis is.
- The `content` carries the **reasoning**, not just the verdict — a future reader (or you) inherits the chain, not only the conclusion.
- `--unresolved` is mandatory. State what you could not settle, or use the exact conventional string `"None identified"` — never pad it.
- `current` is chosen by `--timestamp` then id, so a backdated synthesis does not displace a newer one.
- A synthesis is a claim: it can be cited as an input, recalled, and retracted. `conflicts` marks a synthesis **stale** when one of its inputs is retracted; re-synthesise rather than editing.
- **A synthesis does not inherit its inputs' scope.** If you reconcile
  `personal` claims into an `organisation` synthesis, the claims stay withheld
  but your reasoning is shared — and your `content` may restate them. This is
  permitted, not an error; `validate` warns (`scope_wider_than_inputs`) and so
  does `synthesis create`. Three ways out, in the order usually worth trying:
  **promote the inputs** (`particulars publish <id>… --scope <s>`), which keeps
  the conclusion shareable and its evidence with it; write the synthesis at the
  narrowest input's scope, which demotes the conclusion; or keep the wider
  synthesis free of what the narrower inputs actually say.
- The tool does not judge contradiction. `unsynthesised` means "not yet reconciled into the current belief"; you decide whether it conflicts or merely extends. `recall` marks each entry `current` or `unsynthesised` so you can see this without running `conflicts`.
- Merged particulars are one class: `recall`/`conflicts` on any member cover all members, and each entry keeps its own `subject`.

## Retraction

Retract when a claim is wrong, with a reason a reviewer can check. Never try to
edit or delete the file. Say **why it died** with `--kind`, at whichever of the
three joints broke: `defect` — the claim misread its source; `supersession` —
the source was right then and the world moved on; `provenance-failure` — the
source itself was wrong, which makes every other claim citing that document
worth re-reading. Declare it; it is never inferred from `--superseded-by`,
because a typo fix usually has a replacement and a decommissioned subject
usually has none. For typo-grade corrections assert the corrected claim
first, then retract the old one with `--superseded-by <new id>`. For
disagreements of substance, write a synthesis instead. A wrong merge is retracted
the same way (`retract <mrg_id>`); merges cannot be superseded.

## Working with git

Work on a branch. The YAML you produce is the PR; the human's merge is the
acceptance. Commit the object files **and** `index.yaml` (the tool keeps it
current). If `index.yaml` ever conflicts on merge, run `particulars index` — do
not resolve it by hand. Do not push or open PRs unless the user's workflow says
to.

**Show the shape, not just the diff.** When you write a branch up for a human —
a PR body, a review comment, a summary — draw each particular you touched:

```sh
particulars export --format mermaid --subject "<thing>" --depth 2
```

Put the output in a ` ```mermaid ` fence. GitHub and most markdown viewers
render it, and the reviewer sees what a list of filenames cannot: which claim
you cited as the antithesis, whether your synthesis actually became the current
belief, what is still unreconciled. Add `--include-retracted` when the branch
retracts something, so the `superseded-by` edge is visible.

Two cautions. **A diagram discloses whatever it contains** — check `--scope`
before pasting one anywhere more public than the workspace itself. And if the
repository's CI already posts diagrams on pull requests, do not paste your own
as well.

## Example session

```sh
# Learned from reading src/billing/ that invoices are generated nightly.
particulars particular resolve "Billing Service" --json || \
  particulars particular define --label "Billing Service" --alias billing --json

particulars recall "Billing Service" --json
# → one claim from last month says invoices are generated on demand.

particulars claim assert --subject "Billing Service" \
  --content "Invoices are generated by a nightly cron job (02:00 UTC), not on demand." \
  --evidential observed --document src/billing/cron.go --hash-document \
  --quote 'c := cron.New(); c.AddFunc("0 2 * * *", generateInvoices)' \
  --topic billing --confidence 0.9 --json
# → clm_B

particulars synthesis create --subject "Billing Service" \
  --input clm_A:thesis --input clm_B:antithesis \
  --unresolved "Whether on-demand generation was ever live, or the earlier claim was mistaken." \
  --content "Invoice generation moved from on-demand to a nightly 02:00 UTC cron (src/billing/cron.go:14). The earlier on-demand claim predates commit 3f9c2 which added the cron; treat nightly as current." \
  --json
```
