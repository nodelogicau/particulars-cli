# Publishing knowledge to Microsoft 365 Copilot

Microsoft 365 Copilot runs in Microsoft's cloud. It cannot start a local MCP
server or read a workspace on your disk, so the integration is an **export**:
merged knowledge is pushed into Microsoft Graph as a
[synced Copilot connector](https://learn.microsoft.com/en-us/microsoft-365/copilot/extensibility/overview-copilot-connector),
where it is semantically indexed and citable from Copilot Chat, Microsoft
Search, Copilot in Excel, and the Researcher agent.

```
agent writes claims → PR → a human reviews the evidence → merge
                                                            │
                                        particulars export --format graph
                                                            │
                                     PUT /external/connections/{id}/items/{id}
                                                            ▼
                                   Copilot Chat · Search · Excel · Researcher
```

Only knowledge that a human merged ever reaches Copilot.

## What is indexed, and what is not

One item per **particular**, carrying the current belief rather than individual
claims. This is deliberate: in DKF the current belief is *computed* from the
lineage, never stored, so a flat index of claims would let Copilot cite a
retracted claim or an overturned thesis with full confidence. Each item's body
reads:

```
Project X  (https://nodelogic.au/particulars/project-x)

CURRENT BELIEF (syn_0191…, 2026-08-21T09:00:00Z, confidence 0.9)
Microservices 2022–2024, consolidated to a monolith in Nov 2024; auth remains
separately deployable for compliance.

NOT RECONCILED
The compliance basis for the separate auth service is asserted but unsourced.

SUPPORTING
- Uses Postgres 16, confidence 0.9 — evidence: docs/architecture.md
- Billing split back out in Q2 2026 — evidence: adr/0042.md   [unsynthesised]
```

Copilot therefore inherits the caveats along with the conclusion.

**Never exported:**

- **`personal` knowledge.** Only `organisation` and `public` **effective** scopes
  leave the workspace. `--scope personal` is rejected rather than honoured, so
  this cannot regress into a flag default.
- **Retracted claims, syntheses, and merges.** A retracted synthesis is not the
  belief; the newest standing one is.
- A particular with nothing exportable produces no item at all.

**A caveat the export cannot enforce.** Scope is declared per assertion and is
not inherited, so an `organisation` synthesis of `personal` claims is exported
while those claims are withheld — its `claimCount` reads 0, but its *content* may
summarise them. `particulars validate` warns (`scope_wider_than_inputs`) when a
synthesis is shareable more widely than an input it reasons from; the export
honours declared scope and cannot judge what a summary discloses. Treat the
warning as a prompt to read the synthesis before it is merged.

## Promoting a personal workspace

A workspace whose `defaults.scope` was `personal` exports nothing — every claim
is withheld, correctly. Claims are immutable, so the fix is not to rewrite them:

```sh
particulars publish clm_… syn_… --scope organisation --reason "cleared for the docs site"
```

That writes `publishes/pub_….yaml`. An object's **effective scope** is its
asserted scope widened by the non-retracted promotions covering it, so the same
files now export, keeping their ids and their lineage.

Three rules worth knowing before you promote:

- **Promotion may only widen.** A scope narrower than an object's asserted scope
  is refused. That fixes the direction in which the format fails: a consumer
  ignoring `/publishes/` withholds something authorised, but can never expose
  something restricted.
- **Promotion does not cascade.** Promoting a synthesis leaves its inputs where
  they are, so Copilot can receive a belief whose supporting claims are absent —
  `claimCount: 0`. Promote the inputs too when the evidence should travel with
  the conclusion. There is deliberately no flag that does both in one step.
- **Retraction stops future readers, not past ones.** Retracting the promotion
  reverts the effective scope and the next sync deletes the item, but nothing in
  this format can recall what an external consumer already fetched.

**Data movement:** exported content is copied into Microsoft Graph. If your
knowledge must stay in place, the alternative is a
[custom federated connector](https://learn.microsoft.com/en-us/microsoft-365/copilot/connectors/set-up-custom-federated-connectors),
which queries live over MCP — but it needs a remote HTTPS MCP server with OAuth
that you host, it has no semantic index, and it is **read-only by contract**, so
it costs a hosted service to arrive in the same place.

## Item shape

| Property | Notes |
|---|---|
| `title` | the particular's label (semantic label `title`) |
| `url` | the current synthesis's file in source control (semantic label `url`) |
| `particularUri` | the DKF URI; exact-match queryable |
| `scope` | `organisation` or `public`; refinable |
| `topics` | distinct topics, sorted; refinable |
| `authors` | distinct `source.author`; semantic label `authors` |
| `lastModifiedDateTime` | newest exported assertion, ISO 8601; refinable |
| `claimCount`, `openQuestions` | how much is known, and how much is unreconciled |
| `currentSynthesis` | the id backing the belief |

`content` is Microsoft's built-in property — it is not declared in the schema and
is what gets semantically indexed and summarised.

Because `url` points at the synthesis's YAML, a Copilot citation opens the file a
human approved, with its provenance and `unresolved` intact.

## One-time setup

1. **Emit the payloads.**

   ```sh
   particulars export --format graph --schema --connection particulars > setup.json
   ```

2. **Create the connection and register the schema** with the `connection` and
   `schema` objects from that file (Graph Explorer, or `POST /external/connections`
   then `PATCH /external/connections/{id}/schema`). Schema registration is
   asynchronous; wait for it to complete before pushing items.

3. **Create an Entra app** with the **application** permission
   `ExternalItem.ReadWrite.OwnedBy`, granted admin consent, and add a **federated
   credential** trusting your knowledge repository so the workflow needs no client
   secret. Record the tenant and client ids as `AZURE_TENANT_ID` and
   `AZURE_CLIENT_ID`.

4. **Add the workflow**: copy
   [`examples/graph-sync.yml`](examples/graph-sync.yml) into the knowledge
   repository's `.github/workflows/`.

Changing the schema later requires reingesting items — run the workflow with the
`full` input to re-PUT everything.

## Deletion

`--manifest` writes the exported ids. The workflow keeps the previous manifest as
an artifact and deletes ids absent from the current one, so a particular that is
wholly retracted disappears from Copilot on the next sync. If the artifact has
expired the workflow logs that deletions were skipped and continues — a withdrawn
particular may linger for one cycle, which the next run corrects.

## Trying it without a tenant

The export never contacts Microsoft, so it is safe to inspect:

```sh
particulars export --format graph --source-url https://github.com/you/knowledge/blob/main/ | jq .
particulars export --format graph | jq -r '.item.content.value'   # read the briefs
```

Confirm no `personal` content and nothing retracted appears before you wire up
the push.
