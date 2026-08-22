## Context

M365 Copilot extensibility, as verified on 2026-08-22:

| Route | Reach | Writes | Requires |
|---|---|---|---|
| Synced connector | Copilot Chat, Search, Excel, Researcher; semantic index; citations | no | a push job; **no server** |
| Custom federated connector (MCP) | same surfaces, live, no data movement | **no — "exposes read-only tools"** | remote HTTPS MCP server, OAuth/Entra SSO, admin registration |
| Declarative agent + MCP plugin | only inside that agent | likely | same remote server, plus agent packaging |

The federated route is *not* gated for a tenant's own use — an admin adds it under *Copilot › Connectors › Gallery › Created by your org* — but it is read-only, so it costs a hosted service to reach the same place a file export reaches. Writes exist only via declarative agents, which is the hosted implementation the project has already scoped out.

Graph facts this design depends on: items are `PUT /external/connections/{connection-id}/items/{item-id}` with `{acl[], properties{}, content{value,type}}`, payload limit 30 MB, application permission `ExternalItem.ReadWrite.OwnedBy`. The schema is a flat property list with `isSearchable` / `isQueryable` / `isRetrievable` / `isRefinable` / `isExactMatchRequired`, plus semantic labels; **searchable and refinable are mutually exclusive**, `isExactMatchRequired` applies only to non-searchable properties, labelled properties must be retrievable, each label maps to exactly one property, collections need `@odata.type` specifiers, and `DateTime` values must be ISO 8601. `content` is built-in, is not declared in the schema, and is what gets semantically indexed and summarised.

## Goals / Non-Goals

**Goals:**
- Make merged, organisation-scoped knowledge available to M365 Copilot with no server to run.
- Never let Copilot cite retracted or superseded knowledge.
- Never let `personal` knowledge leave the workspace.
- Keep Microsoft specifics out of the CLI: emit payloads, let a workflow push them.

**Non-Goals:**
- Pushing, authenticating, or talking to Graph from the CLI; remote MCP (`serve --http`); write-back from Copilot; per-assertion items; Graph crawl APIs.

## Decisions

### D1. Export, don't push

`particulars export --format graph` writes NDJSON; a workflow does the HTTP. The CLI gains no MSAL/Graph dependency, the export runs offline in tests, and the same output can feed any other sink. The push is ~40 lines of `curl`, which is a fair trade for not coupling a knowledge tool to one vendor's SDK and auth stack.

### D2. One item per particular, not per assertion

A flat index of claims has no notion of *current belief* — in DKF that is computed, never stored — so a per-claim export would let Copilot cite a retracted claim or an overturned thesis with full confidence. Exporting one item per particular, built from the same `query.Recall`/`CurrentSynthesis` used everywhere else, means:

- retracted objects never enter the index at all;
- the item carries the reasoning (the synthesis content) and the caveats (`unresolved`, and which claims are unsynthesised);
- item count tracks particulars, not assertions, so churn and deletion are manageable.

The cost is coarser citations: Copilot cites "what we know about Project X" and links to the synthesis file, rather than an individual claim. Per-assertion items are deferred until a real question needs them (see Open Questions).

**Item id** is the particular id (`par_…`) — stable, unique, and already the workspace's identity for the thing.

### D3. The rendered brief

`content` (type `text`) is deterministic and reads as prose, because it is what Copilot summarises:

```
<label>  (<uri>)

CURRENT BELIEF (<syn id>, <timestamp>, confidence <c>)
<synthesis content>

NOT RECONCILED
<synthesis unresolved>

SUPPORTING
- <claim content> (conf <c>) — evidence: <source.document>   [unsynthesised]
```

Sections are omitted when empty; with no synthesis the belief section is replaced by `NO SYNTHESIS YET — <n> claims not yet reconciled`. Ordering follows `recall`'s lineage order so the output is stable for a given workspace, which keeps re-pushes idempotent.

### D4. Schema

| Property | Type | Flags | Label |
|---|---|---|---|
| `title` | String | searchable, retrievable | `title` |
| `url` | String | retrievable | `url` |
| `particularUri` | String | queryable, exactMatch | |
| `topics` | StringCollection | queryable, refinable, exactMatch | |
| `scope` | String | queryable, refinable | |
| `lastModifiedDateTime` | DateTime | queryable, retrievable, refinable | `lastModifiedDateTime` |
| `authors` | StringCollection | retrievable | `authors` |
| `claimCount`, `openQuestions` | Int64 | queryable, retrievable | |
| `currentSynthesis` | String | retrievable | |

`topics` is refinable and therefore not searchable (the flags are mutually exclusive); it stays findable because topics also appear in the brief. `lastModifiedDateTime` is the greatest timestamp among the particular's exported assertions. `authors` is the distinct set of `source.author`; harnesses are deliberately not exported as a property — provenance of that kind belongs in the file the citation links to.

### D5. Scope is the filter, and it is normative

`personal` assertions are excluded, and a particular with no exportable assertions produces no item. `organisation` and `public` produce `acl: [{type: everyone, value: everyone, accessType: grant}]`. A future `--acl-group <id>` could map scopes to Entra groups; not now. The spec states the exclusion as a requirement so it cannot regress into a flag default.

### D6. `url` points at the reviewed source

`--source-url <base>` (e.g. `https://github.com/nodelogicau/particulars-knowledge/blob/main/`) makes `url` the current synthesis's file, or the newest claim's file when there is no synthesis. A citation therefore opens the YAML a human approved, with its provenance and `unresolved` intact. Without `--source-url`, `url` is omitted and the `url` label is left unmapped.

### D7. Deletion needs memory

An item can only be deleted if we know it existed. The export emits, alongside the items, a manifest of ids (`--manifest`). The sample workflow keeps the previous manifest as an Actions artifact and deletes ids absent from the current one; if the artifact is missing it skips deletions and logs that a stale item may persist until the next run. Committing the manifest into the knowledge repository is the alternative — rejected because it puts vendor-specific state in a format-neutral repo, unlike `index.yaml` which the format itself defines.

### D8. Setup as a command

`export --format graph --schema --connection <id> --name <n> --description <d>` emits the `externalConnection` creation body and the `schema` registration body. Schema changes require reingestion per Microsoft's docs, so the workflow supports a `--full` push that re-PUTs everything.

## Risks / Trade-offs

- [Copilot cites knowledge that has since been retracted] → items are rebuilt from current belief on every merge and deleted when a particular stops qualifying; the window is one sync, not indefinite.
- [Data movement: organisation knowledge is copied into Graph] → stated in `docs/graph.md`; the federated connector is the no-movement alternative and its cost (hosted server, OAuth, still read-only) is documented beside it.
- [Deletion depends on an artifact that can expire] → degradation is a stale item, logged; `--full` re-push plus a periodic reconcile covers it.
- [Coarse citations may frustrate "which claim said that?"] → the brief names claim evidence inline and the link lands in the repo; revisit with per-assertion items if it bites.
- [Schema drift requires reingestion] → `--full` mode; the schema is small and chosen against the documented constraints up front.
- [We cannot verify against a real tenant] → the change ships with unit tests against documented shapes and a documented dry-run procedure; the proposal says plainly that product verification is the user's to close. This is weaker than every other change in this repository and should not be papered over.

## Migration Plan

Additive. Release v0.5.0. First use is: run `--schema` once, create the connection, register the schema, then enable the workflow.

## Open Questions

- Per-assertion items as a second connection (`particulars-claims`) for "which claim cites `adr/0042`?" — deferred; would double citations if merged into one connection.
- Whether `unresolved` deserves its own retrievable property so Copilot can filter on "beliefs with open questions", rather than only living in the brief.
- Whether to emit `iconUrl`/`containerName` labels (particular URI as container?) — cosmetic until someone sees the result cards.
