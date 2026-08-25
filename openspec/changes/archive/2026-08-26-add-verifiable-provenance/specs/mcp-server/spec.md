## MODIFIED Requirements

### Requirement: Spec tool surface
The server SHALL expose tools named exactly `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect`, `lineage_trace`, and `knowledge_publish`, with the parameter names given in the DKF specification (`synthesis_create` additionally takes `particular_id`; `knowledge_publish` takes `claim_ids`, `scope`, `source`, and optional `reason`). `claim_assert` SHALL accept `source.document` as either a string or a mapping of `uri`, `hash`, and `quote`; `claim_retract` SHALL accept an optional `kind`. Parameters that identify a particular SHALL accept an id, URI, label, or alias; `knowledge_publish` SHALL accept claim and synthesis ids only. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Query tools SHALL be annotated read-only; `particular_define` idempotent; no tool destructive.

#### Scenario: Assert then recall
- **WHEN** a client calls `particular_define{label: "Project X"}`, then `claim_assert{particular_id: "Project X", content: "Uses Postgres"}`, then `knowledge_recall{particular_id: "Project X"}`
- **THEN** the recall result contains the asserted claim

#### Scenario: Publishing over MCP
- **WHEN** a client calls `knowledge_publish{claim_ids: [<clm>, <syn>], scope: "organisation"}`
- **THEN** one promotion record is written and the result matches the `publish` verb's `--json` output

#### Scenario: A verifiable document over MCP
- **WHEN** a client calls `claim_assert` with `source.document` as a mapping carrying `uri` and `quote`
- **THEN** the written claim carries the mapping form
