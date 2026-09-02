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

### Requirement: Conflict detection over a claim set
`conflict_detect` SHALL accept either `particular_id` or `claim_ids[]` (exactly one). With `claim_ids`, the given set SHALL be the universe: `current` is the most recent non-retracted synthesis in the set, `unsynthesised` the members not in its transitive inputs, `stale` the member syntheses citing a retracted object; `priority` is `|unsynthesised| + |stale|`.

#### Scenario: Particular form
- **WHEN** `conflict_detect{particular_id}` is called
- **THEN** the result equals `particulars conflicts <particular> --json` for that particular (an empty `reports` list when below the reporting threshold)

#### Scenario: Claim-set form
- **WHEN** `conflict_detect{claim_ids: [clm_A, clm_B, syn_C]}` is called where `syn_C` cites only `clm_A`
- **THEN** the result has `current: syn_C` and `unsynthesised: [clm_B]`

#### Scenario: Both or neither
- **WHEN** `conflict_detect` is called with both parameters, or neither
- **THEN** the result is an error with code `usage`

### Requirement: Extension tools are labelled
The server SHALL expose `topics_list()` and `workspace_status()` as read-only tools whose descriptions begin with "(particulars extension, not part of the DKF tool set)". `workspace_status` SHALL return the workspace root, id, base-uri, object counts, the number of `validate` errors and warnings, the conflict reports, and — when the workspace root is inside a git checkout — the paths of workspace files that are modified or untracked, obtained read-only. It SHALL never run a git command that writes.

#### Scenario: Status after an assert
- **WHEN** a claim is asserted in a git-tracked workspace and `workspace_status` is called
- **THEN** the result lists the new claim file and the index under uncommitted files and reports `validate` with 0 errors

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
The `initialize` response SHALL include `instructions` consisting of a short header naming the bound workspace (root and id) and stating that writes land as files for human review, followed by the embedded skill's body. When the workspace carries a conventions document (per the `workspace` capability), the instructions SHALL additionally carry its content after the skill body, under a heading naming the file, truncated at 16 KiB with a note naming the file when longer. A configured conventions file that cannot be read SHALL be reported on stderr and omitted, never failing startup. The server SHALL also expose a prompt named `particulars-discipline` returning the same text.

#### Scenario: Instructions present
- **WHEN** a client initialises
- **THEN** `instructions` contains the workspace root and the phrase "Recall **before** you assert"

#### Scenario: Workspace conventions delivered
- **WHEN** the workspace root holds `CONVENTIONS.md` and a client initialises
- **THEN** `instructions` contains a heading naming `CONVENTIONS.md` followed by the file's content, after the skill body

#### Scenario: No conventions, no section
- **WHEN** the workspace has no conventions document
- **THEN** `instructions` contains no conventions heading

#### Scenario: Prompt available
- **WHEN** a client lists prompts and gets `particulars-discipline`
- **THEN** the prompt's message text equals the instructions body

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
