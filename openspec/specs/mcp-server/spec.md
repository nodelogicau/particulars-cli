# MCP Server

## Purpose

The stdio Model Context Protocol front-end: one server bound to one workspace, the DKF specification's tool surface plus two labelled extensions, results identical to the CLI's `--json`, error mapping, provenance defaulted from the client handshake, instructions and prompt delivery, and write serialisation.
## Requirements
### Requirement: Stdio MCP server bound to one workspace
`particulars serve --mcp [--workspace <dir>] [--author] [--harness] [--model]` SHALL run a Model Context Protocol server over stdio, bound at startup to exactly one workspace resolved as every verb does (flag, `DKF_WORKSPACE`, then `dkf.yaml`/`.dkf` discovery from the working directory). Without a workspace it SHALL exit with code 5 and a message on stderr. It SHALL write nothing to stdout except JSON-RPC. `serve` without `--mcp` SHALL be a usage error.

#### Scenario: Started in a project
- **WHEN** `particulars serve --mcp` is started in a directory whose ancestor has a `.dkf` pointing at a valid workspace
- **THEN** the server initialises and every tool operates on that workspace

#### Scenario: No workspace
- **WHEN** `particulars serve --mcp` is started with no resolvable workspace
- **THEN** the process exits with code 5 and stdout is empty

### Requirement: Spec tool surface
The server SHALL expose tools named exactly `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect`, `lineage_trace`, and `knowledge_publish`, with the parameter names given in the DKF specification (`synthesis_create` additionally takes `particular_id`; `knowledge_publish` takes `claim_ids`, `scope`, `source`, and optional `reason`). `claim_assert` SHALL take a required `evidential` of `observed`, `inferred`, or `held`, with no default, refusing `confidence` together with `held`; it SHALL accept `source.document` as either a string or a mapping of `ref`, `author`, `hash`, and `quote`, accepting `uri` as a legacy alias for `ref` and refusing a mapping carrying both. Every tool taking `source` SHALL publish an input schema in which `source.document` is stated as either a string or an object with the properties `ref`, `uri`, `author`, `hash`, and `quote`. Every tool taking `source` SHALL accept `source.quote` and `source.hash` as shorthand for the document's `quote` and `hash`, and `source.document_author` (or `source.document-author`) as shorthand for the document's `author`: the shorthand fills that field of the per-call document, a string document becoming its `ref`, and the written object SHALL be identical to one sent with the field nested, a shorthand author resolving as `document.author` does. A shorthand field without a per-call `source.document`, or set both as shorthand and inside the document with a different value, SHALL be a usage error that names where the field belongs; the same value given both ways SHALL be accepted and written once. `claim_assert` and `synthesis_create` SHALL accept the same top-level shorthand. Top-level `scope` and `topics` are shorthand for `context.scope` and `context.topics`: when both scopes are given `context.scope` SHALL win, and when both topic lists are given the written topics SHALL be their union, `context.topics` first, without duplicates. Top-level `document`, `quote`, `hash`, `document_author` (or `document-author`), `harness`, and `model` are shorthand for the same fields of `source`, folded into `source` before provenance resolves so that every rule above applies to them unchanged; a field given both at the top level and in `source` with a different value SHALL be a usage error, and the same value given both ways SHALL be accepted. A document given both at the top level and in `source` SHALL be merged field by field after each is read as a document — a string being `{ref: string}` — keeping a field set on either side, accepting a field equal on both, and refusing with a usage error a field that differs. The written object SHALL be identical to one sent through `source` and `context`. `source.author` and `document.author` SHALL accept a particular id, URI, or bare name, resolved for writing per `source-attribution`: an unknown id or an ambiguous per-call name is an error result; a server-flag or workspace default that is ambiguous is written unchanged. `claim_retract` SHALL accept an optional `kind`. `synthesis_create` SHALL accept `method` only from the closed vocabulary. `knowledge_recall` SHALL accept an optional `author` — an id, URI, label, or alias — returning the objects asserted by or reported from that particular's merge class, each entry carrying `relations` (`asserted`, `reported`, or both), combinable with `particular_id`, `scope`, `topics`, and `include_retracted`. `knowledge_recall` SHALL accept an optional `query`, free text, that performs recall by query as defined for `recall --query` in `knowledge-query`, combinable with `author`, `scope`, `topics`, `include_retracted`, and `limit`, where `limit` counts particulars; passing both `query` and `particular_id` SHALL be a usage error, and `query` SHALL NOT be treated as a particular reference. Parameters that identify a particular SHALL accept an id, URI, label, or alias; `knowledge_publish` SHALL accept claim and synthesis ids only. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Query tools SHALL be annotated read-only; `particular_define` idempotent; no tool destructive.

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

#### Scenario: A string document and the mapping it names
- **WHEN** a client calls `claim_assert` with `document: "session X"` and `source: {document: {ref: "session X", quote: "the sentence"}}`
- **THEN** the call succeeds and the written document is `{ref: "session X", quote: "the sentence"}`

#### Scenario: Harness at the top level
- **WHEN** a client calls `claim_assert` with top-level `harness: "bench"` and no `source`
- **THEN** the written claim's `source.harness` is `bench`

#### Scenario: A flattened synthesis
- **WHEN** a client calls `synthesis_create` with top-level `harness`, `scope: "personal"`, and `topics: ["goals"]`
- **THEN** the written synthesis carries that harness in `source`, and `scope` and `topics` in its context, identical to one sent through `source` and `context`

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

### Requirement: Conflict detection over a claim set
`conflict_detect` SHALL accept either `particular_id` or `claim_ids[]` (exactly one). With `claim_ids`, the given set SHALL be the universe: `current` is the most recent non-retracted synthesis in the set, `unsynthesised` the members not in its transitive inputs (following synthesis `inputs` only, never a claim's `basis`), `stale` the non-retracted member claims and syntheses that cite, directly or transitively through `inputs` and `basis`, a retracted object; `priority` is `|unsynthesised| + |stale|`. Staleness SHALL be followed through objects outside the set, since what a member rests on is not limited to what the caller named.

#### Scenario: Particular form
- **WHEN** `conflict_detect{particular_id}` is called
- **THEN** the result equals `particulars conflicts <particular> --json` for that particular (an empty `reports` list when below the reporting threshold)

#### Scenario: Claim-set form
- **WHEN** `conflict_detect{claim_ids: [clm_A, clm_B, syn_C]}` is called where `syn_C` cites only `clm_A`
- **THEN** the result has `current: syn_C` and `unsynthesised: [clm_B]`

#### Scenario: A stale rule in a claim set
- **WHEN** `conflict_detect{claim_ids: [clm_rule, clm_B]}` is called, `clm_rule`'s basis lists `clm_a`, which is not in the set, and `clm_a` is retracted
- **THEN** the result lists `clm_rule` in `stale`

#### Scenario: Both or neither
- **WHEN** `conflict_detect` is called with both parameters, or neither
- **THEN** the result is an error with code `usage`

### Requirement: Extension tools are labelled
The server SHALL expose `topics_list()` and `workspace_status()` as read-only tools whose descriptions begin with "(particulars extension, not part of the DKF tool set)". `workspace_status` SHALL return the workspace root, id, base-uri, object counts, the number of `validate` errors and warnings, the conflict reports, and — when the workspace root is inside a git checkout — the paths of workspace files that are modified or untracked, obtained read-only. It SHALL never run a git command that writes.

#### Scenario: Status after an assert
- **WHEN** a claim is asserted in a git-tracked workspace and `workspace_status` is called
- **THEN** the result lists the new claim file and the index under uncommitted files and reports `validate` with 0 errors

### Requirement: Unresolved listing tool
The server SHALL expose `unresolved_list(particular_id?, scope?, include_none?)` as a read-only tool whose description begins with "(particulars extension, not part of the DKF tool set)". Its structured result SHALL equal `particulars unresolved [<particular>] [--scope <s>] [--include-none] --json`: `{"entries": [...], "count": <n>}` with the entry shape and ordering defined by `unresolved-listing`. `particular_id` SHALL accept an id, URI, label, or alias; an unknown one SHALL be an error result with code `not_found`.

#### Scenario: Workspace-wide listing
- **WHEN** `unresolved_list{}` is called in a workspace whose current syntheses carry two real `unresolved` values and one `None identified`
- **THEN** the structured result has `count: 2`, ordered by the current syntheses' timestamps ascending

#### Scenario: Include the convention
- **WHEN** `unresolved_list{include_none: true}` is called in that workspace
- **THEN** the structured result has `count: 3`

#### Scenario: Unknown particular
- **WHEN** `unresolved_list{particular_id: "Nobody"}` is called
- **THEN** the result has `isError: true` and `error.code: "not_found"`

### Requirement: Results carry their data as text
A successful tool result SHALL carry, after its one-line summary text block, a second text block holding its structured content as JSON, so that a client which passes the model only text content still passes the data.

#### Scenario: Listing in a text-only client
- **WHEN** `unresolved_list` returns one entry
- **THEN** the first text block is `1 unresolved` and the second parses as JSON equal to the structured content, naming that entry's synthesis

### Requirement: Errors are tool results with CLI codes
A domain failure (not found, usage, invalid input, conflict, runtime) SHALL be returned as a tool result with `isError: true`, a text block `<code>: <message>`, and structured `{"error": {"code", "message"}}` using the CLI's error codes. Protocol-level failures only SHALL be JSON-RPC errors.

#### Scenario: Unknown input id
- **WHEN** `synthesis_create` names an input that does not exist
- **THEN** the result has `isError: true` and `error.code: "not_found"`

#### Scenario: Ambiguous particular
- **WHEN** a `particular_id` argument matches two particulars
- **THEN** the result has `isError: true`, `error.code: "usage"`, and the message lists both ids

### Requirement: Provenance defaults from the handshake
For each session the server SHALL default `source.harness` to the client's `clientInfo.name` from `initialize` when no harness is supplied by the call, server flags, environment, or `dkf.yaml`. `author` SHALL default from `--author`, `DKF_AUTHOR`, or `dkf.yaml`. The usual minimum (author or harness; harness for syntheses) SHALL apply after defaults.

#### Scenario: Agent-only client
- **WHEN** a client identifying as `claude-ai` calls `claim_assert` with no `source` in a workspace with no default author
- **THEN** the claim is written with `source.harness: claude-ai` and no author

#### Scenario: Call overrides handshake
- **WHEN** the same client passes `source: {harness: "other", author: "ben"}`
- **THEN** the claim records `harness: other` and `author: ben`

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

### Requirement: Conventions document is a resource
When the workspace carries a readable conventions document at startup, the server SHALL expose it as an MCP resource with a `file://` URI naming the document's absolute path, `name` equal to its workspace-relative path, `mimeType` `text/markdown`, whose content is read from the file at request time. When no document applies, no resource SHALL be listed.

#### Scenario: Resource listed and readable
- **WHEN** the root holds `dkf.md` and a client lists resources and reads the one named `dkf.md`
- **THEN** the listing contains exactly one resource, and its content equals the file's current bytes

#### Scenario: No document, no resource
- **WHEN** the workspace has no conventions document
- **THEN** the resource listing is empty

### Requirement: Concurrent writes keep the index consistent
Mutating tools SHALL be serialised by a server-wide lock so that concurrent calls never interleave load, write, and index update.

#### Scenario: Parallel asserts
- **WHEN** twenty `claim_assert` calls run concurrently
- **THEN** twenty claim files exist and `particulars index --check` passes

### Requirement: The tool surface teaches the register
The descriptions and parameter schemas of `claim_assert` and `particular_define` SHALL state the register that separates knowledge from catalogue: the subject is the thing in the world the fact is about, never the document or feed it was read in; content states the fact, and what was read goes in `source.document`. `particular_define`'s description SHALL give identity examples for global URIs (a person, a project) and SHALL NOT give reading matter as an example. The requirement pins presence of the register, not exact wording.

#### Scenario: Assert-time register present
- **WHEN** a client lists tools
- **THEN** `claim_assert`'s description or parameter schemas state that the subject is the thing in the world and that what was read belongs in `source.document`

#### Scenario: Define does not invite document particulars
- **WHEN** a client reads `particular_define`'s description
- **THEN** its URI examples name identities, not articles or feeds, and it states that a particular is a thing in the world, not a document being read

