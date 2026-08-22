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

Find the workspace: it is the nearest ancestor directory containing `dkf.yaml`,
or `$DKF_WORKSPACE`, or `--workspace <dir>`. If there is none and the user wants
one:

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
| Record a fact | `particulars claim assert --subject "<thing>" --content "<statement>" --document <evidence> [--topic <t>]... [--confidence 0..1] [--scope personal\|organisation\|public] --json` |
| Multi-line content | add `--content-file -` and pipe the text on stdin (instead of `--content`) |
| Reconcile | `particulars synthesis create --subject "<thing>" --input <id>:thesis --input <id>:antithesis [--input <id>:thesis:qualifying] --unresolved "<what remains open>" --content "<resolution with reasoning>" --json` |
| What is believed | `particulars recall "<thing>" [--topic <t>]... [--scope <s>] [--limit <n>] --json` → `{entries: [...]}`, oldest first, `current: true` on the live synthesis |
| Across things | `particulars recall --topic <t> --json` |
| Which tags exist | `particulars topics ["<thing>"] --json` → `{topics: [{topic, assertions, particulars}]}`; check before inventing a new tag |
| Why is it believed | `particulars lineage <id> [--depth <n>] --json` → nested tree of inputs with roles |
| What needs work | `particulars conflicts ["<thing>"] --json` → `{reports: [{particular, current, unsynthesised, stale, priority}]}` |
| Withdraw | `particulars retract <id> --reason "<why>" [--superseded-by <id>] --json` — claims, syntheses, or merges; append-only, never deletes (`claim retract` is a deprecated alias) |
| Same thing, two URIs | `particulars particular merge <a> <b> [--reason "<why>"] --json` — either side may be a bare URI with no local particular; undo with `retract <mrg_id>` |
| Health check | `particulars validate --json`; `particulars index --check --json` |

Exit codes: `0` ok · `1` runtime · `2` usage/ambiguous · `3` not found · `4` check failed · `5` no workspace.
On failure stderr carries `{"error": {"code", "message"}}`.

## Rules for good claims

- **One statement per claim.** "Service A times out above 500 rps" — not a paragraph of findings. Split compound observations.
- **Evidence or it didn't happen.** Put the file path, URL, command output, or document in `--document`. Reviewers verify claims in seconds when they can open the evidence.
- **Calibrated confidence.** `0.9+` you saw it directly in code or data; `0.6–0.8` inferred or second-hand; below that, say why in the content.
- **Scope honestly.** Default `personal`. Use `organisation` for things others in the org should see; never `public` unless the user says so.
- **Backdate when recording history.** `--timestamp 2024-11-15T00:00:00Z` for a fact from a dated document; the id still records when you wrote it.
- **Topics are for recall.** Use a few stable, lowercase tags (`architecture`, `auth`, `incident`) so `recall --topic` works later.

## Rules for particulars

- `define` mints the URI from the label, so **the same label always hits the same particular**. Use specific labels: "Auth Service" and "Auth Team", not "Auth".
- If `define` returns `created: false`, you hit an existing particular — check it is the one you meant.
- Things with a global identifier get it: `--uri https://www.wikidata.org/entity/Q42`, an ORCID, a GitHub URL for a repo or file.
- Ambiguous subjects exit 2 listing candidate ids; retry with the id.

## Rules for syntheses

- Roles say how inputs relate: the belief being challenged is `thesis`, the challenger is `antithesis`; supporting context is `thesis:qualifying`.
- The `content` carries the **reasoning**, not just the verdict — a future reader (or you) inherits the chain, not only the conclusion.
- `--unresolved` is mandatory. State what you could not settle, or use the exact conventional string `"None identified"` — never pad it.
- `current` is chosen by `--timestamp` then id, so a backdated synthesis does not displace a newer one.
- A synthesis is a claim: it can be cited as an input, recalled, and retracted. `conflicts` marks a synthesis **stale** when one of its inputs is retracted; re-synthesise rather than editing.
- **A synthesis does not inherit its inputs' scope.** If you reconcile
  `personal` claims into an `organisation` synthesis, the claims stay withheld
  but your reasoning is shared — and your `content` may restate them. `validate`
  warns (`scope_wider_than_inputs`); either write the synthesis at the narrowest
  input's scope, or keep the wider one free of what the narrower inputs say.
- The tool does not judge contradiction. `unsynthesised` means "not yet reconciled into the current belief"; you decide whether it conflicts or merely extends. `recall` marks each entry `current` or `unsynthesised` so you can see this without running `conflicts`.
- Merged particulars are one class: `recall`/`conflicts` on any member cover all members, and each entry keeps its own `subject`.

## Retraction

Retract when a claim is wrong, with a reason a reviewer can check. Never try to
edit or delete the file. For typo-grade corrections assert the corrected claim
first, then retract the old one with `--superseded-by <new id>`. For
disagreements of substance, write a synthesis instead. A wrong merge is retracted
the same way (`retract <mrg_id>`); merges cannot be superseded.

## Working with git

Work on a branch. The YAML you produce is the PR; the human's merge is the
acceptance. Commit the object files **and** `index.yaml` (the tool keeps it
current). If `index.yaml` ever conflicts on merge, run `particulars index` — do
not resolve it by hand. Do not push or open PRs unless the user's workflow says
to.

## Example session

```sh
# Learned from reading src/billing/ that invoices are generated nightly.
particulars particular resolve "Billing Service" --json || \
  particulars particular define --label "Billing Service" --alias billing --json

particulars recall "Billing Service" --json
# → one claim from last month says invoices are generated on demand.

particulars claim assert --subject "Billing Service" \
  --content "Invoices are generated by a nightly cron job (02:00 UTC), not on demand." \
  --document src/billing/cron.go:14 --topic billing --confidence 0.9 --json
# → clm_B

particulars synthesis create --subject "Billing Service" \
  --input clm_A:thesis --input clm_B:antithesis \
  --unresolved "Whether on-demand generation was ever live, or the earlier claim was mistaken." \
  --content "Invoice generation moved from on-demand to a nightly 02:00 UTC cron (src/billing/cron.go:14). The earlier on-demand claim predates commit 3f9c2 which added the cron; treat nightly as current." \
  --json
```
