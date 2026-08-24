## Why

A claim's scope is fixed when it is written, because claims are immutable. That is correct and it is also a trap: a workspace whose `defaults.scope` was `personal` accumulates knowledge that can never be shared, and the only exits the format offered were rewriting the file (forbidden) or re-asserting the claim (a new id, orphaning everything that cited the old one). This implementation hit it in its own knowledge base — the first Graph export produced **zero items from 38 personal claims**, and it was that report that became [particulars#14](https://github.com/nodelogicau/particulars/issues/14).

The spec answered with promotion records, and then — in [#15](https://github.com/nodelogicau/particulars/issues/15), from this implementation's report — defined what promotion does to the `scope_wider_than_inputs` warning we shipped in v0.5.1. Both are now normative in the DKF README at `fd75a11`, and neither exists here. Until they do, `defaults.scope` remains a one-way door.

## What Changes

- **A fourth record type.** `publishes/pub_<uuidv7>.yaml` with `id`, `type: publish`, `claims`, `scope`, optional `reason`, `source`, `timestamp`, and the usual append-only `retracted` block. `claims` names at least one claim or synthesis. Retracted through the existing `retract` verb, like a merge; not superseded.
- **Effective scope.** The widest scope named by a non-retracted promotion covering an object, else its asserted `context.scope`. **Promotion may only widen** — a record naming a scope narrower than a covered object's asserted scope is invalid, which is what makes a partial implementation safe: a consumer that ignores `/publishes/` withholds something authorised, but can never expose something restricted. **Promotion does not cascade** to a synthesis's inputs.
- **A `publish` verb and the tenth MCP tool.** `particulars publish <id>… --scope <s> [--reason R]`, and `knowledge_publish(claim_ids[], scope, source, reason?)` — the one spec tool the server does not implement.
- **Every scope filter becomes an effective-scope filter**: `recall --scope`, `topics --scope`, `export --format graph`, and `export --format dot|mermaid --scope`. The Graph export is the motivating case — a workspace of personal claims becomes exportable by promotion, keeping ids and lineage, with no re-assertion.
- **`scope_wider_than_inputs` compares effective scope on both sides**, and is evaluated at the three points the spec names: when a synthesis is created, when a promotion is written, and during `validate`. Comparing asserted scope is wrong in both directions once promotions exist — it warns about an input that has been promoted to match, and stays silent when a synthesis is promoted past inputs that were never widened, which is **the condition the check exists to catch and in which neither file changes**.
- **`conflicts` ignores promotions entirely.** They are not knowledge, form no equivalence class, and change no conflict set.
- `pub` joins the id grammar — in **both** the lenient and canonical patterns, so promotions we mint are not reported as `legacy_id`.

## Capabilities

### New Capabilities
- `scope-promotion`: the promotion record and its file layout, effective scope and its ordering, the widen-only and no-cascade rules, the `publish` verb and its MCP tool, retraction and reversion, and validation of promotions.

### Modified Capabilities
- `validation`: `scope_wider_than_inputs` currently reads "declares a scope wider than one of its **direct inputs**" with two scenarios written in terms of asserted scope; it becomes an effective-scope comparison. New promotion-specific findings are added.
- `knowledge-query`: `recall --scope` and `topics --scope` filter on effective scope.
- `graph-export`: the export filters on effective scope; `--scope personal` stays refused.
- `visual-export`: `--scope` filters on effective scope.
- `index`: entries for promotions carry `claims` and `scope`, so a filter need not open every file.
- `object-format`: the `pub` identifier prefix and the promotion record's canonical field order.
- `mcp-server`: `knowledge_publish` joins the tool list.

## Impact

- **New**: `internal/dkf/` promotion type, codec and validation; `internal/store/` promotion writes and an effective-scope index; `internal/cli/cmd_publish.go`; `internal/mcp/` tool.
- **Modified**: `internal/query/` (recall, topics, validate, and the warning), `internal/graph/`, `internal/viz/`, `internal/store/index.go`, `internal/cli/cmd_retract.go` and `cmd_export.go`.
- **Docs**: `docs/graph.md` (a personal workspace is now publishable), `docs/visualise.md`, README verb table, and `skills/particulars/SKILL.md` — whose synthesis-scope note currently offers one remedy ("write the synthesis at the narrowest input's scope") where there will be two, and promoting the inputs is usually the better one because it keeps the conclusion shareable.
- **Compatibility**: a workspace with no `publishes/` directory behaves exactly as today. Effective scope collapses to asserted scope when nothing is promoted, so every existing filter returns what it returns now.
- **Not in scope**: an acknowledgement mechanism that lets an author suppress `scope_wider_than_inputs` deliberately. Upstream considered and **deferred** it as more machinery than a warning warrants; adding one here would fork the format on the exact point it just decided.
