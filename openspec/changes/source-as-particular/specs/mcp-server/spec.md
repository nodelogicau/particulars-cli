## MODIFIED Requirements

### Requirement: Spec tool surface
The server SHALL expose tools named exactly `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect`, `lineage_trace`, and `knowledge_publish`, with the parameter names given in the DKF specification (`synthesis_create` additionally takes `particular_id`; `knowledge_publish` takes `claim_ids`, `scope`, `source`, and optional `reason`). `claim_assert` SHALL take a required `evidential` of `observed`, `inferred`, or `held`, with no default, refusing `confidence` together with `held`; it SHALL accept `source.document` as either a string or a mapping of `ref`, `author`, `hash`, and `quote`, accepting `uri` as a legacy alias for `ref` and refusing a mapping carrying both. `source.author` and `document.author` SHALL accept a particular id, URI, or bare name, resolved for writing per `source-attribution`: an unknown id or an ambiguous per-call name is an error result; a server-flag or workspace default that is ambiguous is written unchanged. `claim_retract` SHALL accept an optional `kind`. `synthesis_create` SHALL accept `method` only from the closed vocabulary. `knowledge_recall` SHALL accept an optional `author` — an id, URI, label, or alias — returning the objects asserted by or reported from that particular's merge class, each entry carrying `relations` (`asserted`, `reported`, or both), combinable with `particular_id`, `scope`, `topics`, and `include_retracted`. Parameters that identify a particular SHALL accept an id, URI, label, or alias; `knowledge_publish` SHALL accept claim and synthesis ids only. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Query tools SHALL be annotated read-only; `particular_define` idempotent; no tool destructive.

#### Scenario: Assert then recall
- **WHEN** a client calls `particular_define{label: "Project X"}`, then `claim_assert{particular_id: "Project X", content: "Uses Postgres", evidential: "observed"}`, then `knowledge_recall{particular_id: "Project X"}`
- **THEN** the recall result contains the asserted claim with its evidential

#### Scenario: Assert without a warrant
- **WHEN** a client calls `claim_assert` without `evidential`
- **THEN** the tool returns a usage error naming the three values and writes nothing

#### Scenario: Publishing over MCP
- **WHEN** a client calls `knowledge_publish{claim_ids: [<clm>, <syn>], scope: "organisation"}`
- **THEN** one promotion record is written and the result matches the `publish` verb's `--json` output

#### Scenario: A verifiable document over MCP
- **WHEN** a client calls `claim_assert` with `source.document` as a mapping carrying `ref` and `quote`
- **THEN** the written claim carries the mapping form

#### Scenario: Reported testimony over MCP
- **WHEN** a client calls `claim_assert` with `source.document: {ref: "conversation with Jane", author: "jane"}` and exactly one particular has alias `jane`
- **THEN** the written document mapping carries that particular's `uri` as `author`

#### Scenario: Legacy uri key still accepted
- **WHEN** a client calls `claim_assert` with `source.document: {uri: "docs/db.md"}`
- **THEN** the written claim's document `ref` is `docs/db.md`

#### Scenario: Recall by author over MCP
- **WHEN** a client calls `knowledge_recall{author: "ben"}`
- **THEN** the result equals `particulars recall --author ben --json`, each entry carrying `relations`
