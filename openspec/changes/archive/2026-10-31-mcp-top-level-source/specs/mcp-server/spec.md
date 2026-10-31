## MODIFIED Requirements

### Requirement: Spec tool surface
The server SHALL expose tools named exactly `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect`, `lineage_trace`, and `knowledge_publish`, with the parameter names given in the DKF specification (`synthesis_create` additionally takes `particular_id`; `knowledge_publish` takes `claim_ids`, `scope`, `source`, and optional `reason`). `claim_assert` SHALL take a required `evidential` of `observed`, `inferred`, or `held`, with no default, refusing `confidence` together with `held`; it SHALL accept `source.document` as either a string or a mapping of `ref`, `author`, `hash`, and `quote`, accepting `uri` as a legacy alias for `ref` and refusing a mapping carrying both. Every tool taking `source` SHALL publish an input schema in which `source.document` is stated as either a string or an object with the properties `ref`, `uri`, `author`, `hash`, and `quote`. Every tool taking `source` SHALL accept `source.quote` and `source.hash` as shorthand for the document's `quote` and `hash`, and `source.document_author` (or `source.document-author`) as shorthand for the document's `author`: the shorthand fills that field of the per-call document, a string document becoming its `ref`, and the written object SHALL be identical to one sent with the field nested, a shorthand author resolving as `document.author` does. A shorthand field without a per-call `source.document`, or set both as shorthand and inside the document with a different value, SHALL be a usage error that names where the field belongs; the same value given both ways SHALL be accepted and written once. `claim_assert` SHALL accept top-level `topics` as shorthand for `context.topics`; when both are given the written topics SHALL be their union, `context.topics` first, without duplicates. `claim_assert` SHALL also accept top-level `document`, `quote`, `hash`, and `document_author` (or `document-author`) as shorthand for the same fields of `source`, folded into `source` before provenance resolves so that every rule above applies to them unchanged; a field given both at the top level and in `source` with a different value SHALL be a usage error, and the same value given both ways SHALL be accepted. The written claim SHALL be identical to one sent through `source`. `source.author` and `document.author` SHALL accept a particular id, URI, or bare name, resolved for writing per `source-attribution`: an unknown id or an ambiguous per-call name is an error result; a server-flag or workspace default that is ambiguous is written unchanged. `claim_retract` SHALL accept an optional `kind`. `synthesis_create` SHALL accept `method` only from the closed vocabulary. `knowledge_recall` SHALL accept an optional `author` — an id, URI, label, or alias — returning the objects asserted by or reported from that particular's merge class, each entry carrying `relations` (`asserted`, `reported`, or both), combinable with `particular_id`, `scope`, `topics`, and `include_retracted`. `knowledge_recall` SHALL accept an optional `query`, free text, that performs recall by query as defined for `recall --query` in `knowledge-query`, combinable with `author`, `scope`, `topics`, `include_retracted`, and `limit`, where `limit` counts particulars; passing both `query` and `particular_id` SHALL be a usage error, and `query` SHALL NOT be treated as a particular reference. Parameters that identify a particular SHALL accept an id, URI, label, or alias; `knowledge_publish` SHALL accept claim and synthesis ids only. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Query tools SHALL be annotated read-only; `particular_define` idempotent; no tool destructive.

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

#### Scenario: Quote given beside a string document
- **WHEN** a client calls `claim_assert` with `source: {document: "chat session 1", quote: "I lead a team of 4 engineers."}`
- **THEN** the call succeeds and the written claim's document is the mapping `{ref: "chat session 1", quote: "I lead a team of 4 engineers."}`, identical to the nested form

#### Scenario: Shorthand fills a mapping document
- **WHEN** a client calls `claim_assert` with `source: {document: {ref: "docs/a.md", author: "jane"}, quote: "the sentence"}`
- **THEN** the written document carries `ref`, the resolved `author`, and `quote`

#### Scenario: Quote set twice
- **WHEN** a client calls `claim_assert` with `source: {document: {ref: "docs/a.md", quote: "one"}, quote: "two"}`
- **THEN** the tool returns a usage error saying to pass the quote once, and writes nothing

#### Scenario: Quote without a document
- **WHEN** a client calls `claim_assert` with `source: {quote: "the sentence"}` and no `document`
- **THEN** the tool returns a usage error saying the quote belongs with `source.document`, and writes nothing

#### Scenario: Shorthand on another tool taking source
- **WHEN** a client calls `synthesis_create` with `source: {document: "review notes", quote: "the sentence"}`
- **THEN** the written synthesis's document is the mapping `{ref: "review notes", quote: "the sentence"}`

#### Scenario: The same quote given both ways
- **WHEN** a client calls `claim_assert` with `source: {document: {ref: "docs/a.md", quote: "one"}, quote: "one"}`
- **THEN** the call succeeds and the written document carries `quote: "one"` once

#### Scenario: Document author as a flat field
- **WHEN** a client calls `claim_assert` with `source: {document: "conversation with Jane", document_author: "jane"}` and exactly one particular has alias `jane`
- **THEN** the written document is identical to one sent as `{ref: "conversation with Jane", author: "jane"}`, carrying that particular's `uri` as `author`

#### Scenario: Document author without a document
- **WHEN** a client calls `claim_assert` with `source: {document_author: "jane"}` and no `document`
- **THEN** the tool returns a usage error saying the author belongs with `source.document`, and writes nothing

#### Scenario: Topics at the top level
- **WHEN** a client calls `claim_assert` with `topics: ["billing", "ports"]` and `context: {topics: ["ports", "infra"]}`
- **THEN** the written claim's `context.topics` is `["ports", "infra", "billing"]`

#### Scenario: CLI flags as top-level fields
- **WHEN** a client calls `claim_assert{particular_id: "Billing", content: "443", evidential: "observed", document: "chat session 1", quote: "listens on 443", document_author: "jane"}` with no `source`
- **THEN** the written claim's source document is identical to one sent as `source: {document: {ref: "chat session 1", quote: "listens on 443", author: "jane"}}`

#### Scenario: Top-level and source document disagree
- **WHEN** a client calls `claim_assert` with `document: "docs/a.md"` and `source: {document: "docs/b.md"}`
- **THEN** the tool returns a usage error saying to pass the document once, and writes nothing

#### Scenario: Top-level quote without a document
- **WHEN** a client calls `claim_assert` with top-level `quote` and no `document` at the top level or in `source`
- **THEN** the tool returns a usage error saying the quote needs `source.document`, and writes nothing

#### Scenario: The document's shape is in the schema
- **WHEN** a client lists tools
- **THEN** the input schema of `claim_assert`, `synthesis_create`, `claim_retract`, `knowledge_publish`, and `particular_merge` states `source.document` as either a string or an object whose properties include `ref` and `quote`, and states `source.quote`

#### Scenario: Reported testimony over MCP
- **WHEN** a client calls `claim_assert` with `source.document: {ref: "conversation with Jane", author: "jane"}` and exactly one particular has alias `jane`
- **THEN** the written document mapping carries that particular's `uri` as `author`

#### Scenario: Legacy uri key still accepted
- **WHEN** a client calls `claim_assert` with `source.document: {uri: "docs/db.md"}`
- **THEN** the written claim's document `ref` is `docs/db.md`

#### Scenario: Recall by author over MCP
- **WHEN** a client calls `knowledge_recall{author: "ben"}`
- **THEN** the result equals `particulars recall --author ben --json`, each entry carrying `relations`

#### Scenario: Recall by query over MCP
- **WHEN** a client calls `knowledge_recall{query: "what is the session timeout?"}`
- **THEN** the result equals `particulars recall --query "what is the session timeout?" --json`, grouped by particular with `resolution` on each

#### Scenario: A name passed as query
- **WHEN** a client calls `knowledge_recall{query: "Project X"}` and a particular is labelled `Project X`
- **THEN** that particular is the first in `particulars` with `resolution: label`, not an error and not a flat `entries` list

#### Scenario: Query and particular together over MCP
- **WHEN** a client calls `knowledge_recall{particular_id: "Project X", query: "timeout"}`
- **THEN** the tool returns a usage error result
