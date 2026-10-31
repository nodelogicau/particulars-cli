## MODIFIED Requirements

### Requirement: Spec tool surface
The server SHALL expose tools named exactly `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect`, `lineage_trace`, and `knowledge_publish`, with the parameter names given in the DKF specification (`synthesis_create` additionally takes `particular_id`; `knowledge_publish` takes `claim_ids`, `scope`, `source`, and optional `reason`). `claim_assert` SHALL take a required `evidential` of `observed`, `inferred`, or `held`, with no default, refusing `confidence` together with `held`; it SHALL accept `source.document` as either a string or a mapping of `ref`, `author`, `hash`, and `quote`, accepting `uri` as a legacy alias for `ref` and refusing a mapping carrying both. Every tool taking `source` SHALL publish an input schema in which `source.document` is stated as either a string or an object with the properties `ref`, `uri`, `author`, `hash`, and `quote`. Every tool taking `source` SHALL accept `source.quote` and `source.hash` as shorthand for the document's `quote` and `hash`: the shorthand fills that field of the per-call document, a string document becoming its `ref`, and the written object SHALL be identical to one sent with the field nested. A shorthand field without a per-call `source.document`, or set both as shorthand and inside the document, SHALL be a usage error that names where the field belongs. `source.author` and `document.author` SHALL accept a particular id, URI, or bare name, resolved for writing per `source-attribution`: an unknown id or an ambiguous per-call name is an error result; a server-flag or workspace default that is ambiguous is written unchanged. `claim_retract` SHALL accept an optional `kind`. `synthesis_create` SHALL accept `method` only from the closed vocabulary. `knowledge_recall` SHALL accept an optional `author` — an id, URI, label, or alias — returning the objects asserted by or reported from that particular's merge class, each entry carrying `relations` (`asserted`, `reported`, or both), combinable with `particular_id`, `scope`, `topics`, and `include_retracted`. `knowledge_recall` SHALL accept an optional `query`, free text, that performs recall by query as defined for `recall --query` in `knowledge-query`, combinable with `author`, `scope`, `topics`, `include_retracted`, and `limit`, where `limit` counts particulars; passing both `query` and `particular_id` SHALL be a usage error, and `query` SHALL NOT be treated as a particular reference. Parameters that identify a particular SHALL accept an id, URI, label, or alias; `knowledge_publish` SHALL accept claim and synthesis ids only. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Query tools SHALL be annotated read-only; `particular_define` idempotent; no tool destructive.

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

### Requirement: Instructions and prompt carry the skill
The `initialize` response SHALL include `instructions` consisting of a short header naming the bound workspace (root and id) and stating that writes land as files for human review, followed by an MCP preamble, followed by the embedded skill's body. The preamble SHALL state that the skill's commands are not run over MCP — there is no shell and no `particulars` tool, and `--json` does not apply — and SHALL map each `particulars` command the skill names either to the tool that serves it or to having no MCP equivalent, including how `claim assert`'s document flags map to `source` parameters and that the register needs no call because the instructions carry it. Every tool the server registers SHALL appear in that mapping, and the mapping SHALL name no tool the server does not register. The instructions SHALL then carry the workspace register (per the `workspace-register` capability) under a heading naming it, so that a client which never reads the repository can see which particulars and topics the workspace already uses before it chooses either. When the workspace carries a conventions document (per the `workspace` capability), the instructions SHALL additionally carry its content after the register, under a heading naming the file — last, so that a human's stated convention is read after the facts it comments on and takes precedence over them. When the server limits what it delivers it SHALL deliver at least the first 16 KiB of the document's UTF-8 encoding, SHALL cut only on a character boundary — advancing past a character that straddles the limit, never cutting short of it — and SHALL append a note saying the text was truncated and naming the file. A configured conventions file that cannot be read SHALL be reported on stderr and omitted, never failing startup. The server SHALL also expose a prompt named `particulars-discipline` returning the same text.

#### Scenario: Instructions present
- **WHEN** a client initialises
- **THEN** `instructions` contains the workspace root and the phrase "Recall **before** you assert"

#### Scenario: The preamble maps commands to tools
- **WHEN** a client initialises
- **THEN** `instructions` contains, before the skill body, a statement that there is no `particulars` tool, a mapping naming `particular_resolve` for `particular resolve` and `knowledge_recall` for `recall`, and a statement that the register is already included

#### Scenario: Every registered tool is mapped
- **WHEN** the server's registered tools are listed
- **THEN** each tool name appears in the preamble's mapping, and every tool the mapping names is registered

#### Scenario: Workspace conventions delivered
- **WHEN** the workspace root holds `dkf.md` and a client initialises
- **THEN** `instructions` contains a heading naming `dkf.md` followed by the file's content, after the skill body

#### Scenario: No conventions, no section
- **WHEN** the workspace has no conventions document
- **THEN** `instructions` contains no conventions heading

#### Scenario: Truncation lands on a character boundary and honours the floor
- **WHEN** the document is longer than 16 KiB and a multi-byte character begins before and ends after the 16 KiB mark
- **THEN** the delivered text is valid UTF-8, contains that whole character, is at least 16 KiB long, and is followed by a note naming the file

#### Scenario: Prompt available
- **WHEN** a client lists prompts and gets `particulars-discipline`
- **THEN** the prompt's message text equals the instructions body

#### Scenario: The register travels with the instructions
- **WHEN** a client initializes against a workspace holding 39 particulars and 195 topics
- **THEN** `instructions` carries a register section listing them, between the skill body and any conventions document

#### Scenario: A workspace with no conventions document
- **WHEN** the workspace carries no conventions document
- **THEN** `instructions` still carries the register, because it is derived rather than authored
