## Context

`unresolved` is mandatory on every synthesis and read by nothing. `conflicts` already computes the two things this verb needs — the class enumeration (one report per merge equivalence class, keyed by its lowest particular) and `current` (the non-retracted synthesis with the greatest `(timestamp, id)` in the class, via `CurrentForClass`) — but applies a reporting threshold that hides any class that is merely *settled with open questions*, which is the normal state of a healthy particular. The dogfood workspace makes the distinction concrete: 37 syntheses, 7 current; 29 superseded ones whose `unresolved` text describes a past state. Only current syntheses carry live questions, and every predecessor's questions are either restated by its successor or answered by it.

## Goals / Non-Goals

**Goals:** one command that lists the workspace's admitted open questions, oldest first, with enough structural context to see where new evidence may already exist; MCP parity; a single definition of `current` shared with `conflicts` and `recall`.

**Non-Goals:** item-level tracking of questions across syntheses (`unresolved` is one prose string per synthesis; splitting it is a format question for upstream, not a CLI heuristic); any judgement of whether an unsynthesised claim answers a question; a reporting threshold or CI exit code (an open question is not a failure).

## Decisions

**D1. A new verb, not a flag on `conflicts`.** `conflicts` answers "what is structurally unreconciled" and omits classes below its threshold; this verb answers "what does the current belief say it does not know" and wants every class with a current synthesis. Bolting `--unresolved` onto `conflicts` would either break its threshold or produce a list missing most particulars. The two verbs share code, not a surface. Alternative considered: a workspace-wide `recall --current`, rejected because `recall` deliberately requires a selector and its entries are full objects, not a backlog view.

**D2. Reuse `conflicts`' machinery verbatim.** `Unresolved(g, subject)` in `internal/query` enumerates classes the way `Conflicts` does (or the one class of `subject`), calls `CurrentForClass`, and counts `unsynthesised` by the same rule as `Analyse` (non-retracted class assertions that are neither current nor in `Closure(current)`). The simplest implementation calls `Analyse` per class and reads `Current` and `len(Unsynthesised)` off the report, then fetches the synthesis for `unresolved` and `timestamp`. No second definition of `current` anywhere. Classes with no current synthesis are skipped: nothing has been settled, so nothing has been admitted open — that case belongs to `conflicts`.

**D3. Oldest current synthesis first.** Sort ascending by `(timestamp, id)` of the current synthesis. The question that has stood longest without a re-synthesis is the one most likely to be either stale or neglected; `conflicts`' priority ordering is available on the other verb for the "busiest" view.

**D4. `None identified` is filtered by default.** Exact string match, as the syntheses spec defines the conventional value; `--include-none` includes those entries. Trimmed comparison is deliberate: `synthesis create` already rejects blank values, and the spec names the exact string. No other value is treated as empty — a producer who writes "nothing" instead of the convention is reported, which is the right nudge.

**D5. Entry shape mirrors `conflicts.Report` plus the synthesis fields.** `particular`, `label`, `uri`, `members` (when class > 1), `synthesis` (id), `timestamp`, `unresolved`, `unsynthesised` (an integer count, not the id list — the ids are one `conflicts <particular>` away and the count is what a reader scans for). Keying by the class's lowest particular id keeps results stable across `recall`, `conflicts`, and this verb. Text mode prints one block per entry: `<id>  <label>  <date>`, then `synthesis:`, `unresolved:`, and `unsynthesised: N` only when N > 0; `members:` when present.

**D6. Scope filters on the current synthesis's effective scope**, as `topics` does with `EffectiveScope`, so a promoted synthesis is visible under the scope it was promoted to. Alternative considered: filtering by asserted scope; rejected because every other read-side scope filter uses effective scope.

**D7. Empty is exit 0.** `unresolved` with nothing to show prints `Nothing unresolved.` and `{"entries": [], "count": 0}`. Exit 3 is for a selector that resolves to nothing; an optional `[<particular>]` argument that does not resolve still exits 3, as every other verb does.

**D8. MCP tool `unresolved_list(particular_id?, scope?, include_none?)`**, read-only, description prefixed "(particulars extension, not part of the DKF tool set)", structured result equal to the verb's `--json`. Registered beside `topics_list`; the DKF tool set is unchanged.

**D9. The skill names it as the second half of "what needs work".** The query table gains a row and the review-flow diagram gains a line under `conflicts`. The installed copy is regenerated from a HEAD build with `skill install --force`, never copied by hand.

## Risks / Trade-offs

- [A class whose current synthesis is old but whose questions are still accurate ranks first] → that is the intent: age without re-synthesis is the signal; the `unsynthesised` count tells the reader whether evidence has accrued since.
- [Prose `unresolved` bundles several questions; the verb cannot separate them] → out of scope by design; noted as an upstream format question, not solved by heuristics here.
- [`--include-none` output is mostly noise in a mature workspace] → default excludes it; the flag exists for audit ("did every settled particular actually consider the question").

## Migration Plan

Read-only addition; nothing rewrites. Ship in the next minor release; no rollback concerns.

## Open Questions

- None.
