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
The server SHALL expose tools named exactly `particular_define`, `particular_resolve`, `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`, `knowledge_recall`, `conflict_detect`, `lineage_trace`, and `knowledge_publish`, with the parameter names given in the DKF specification (`synthesis_create` additionally takes `particular_id`; `knowledge_publish` takes `claim_ids`, `scope`, `source`, and optional `reason`). Parameters that identify a particular SHALL accept an id, URI, label, or alias; `knowledge_publish` SHALL accept claim and synthesis ids only. Each tool's structured result SHALL equal the corresponding CLI verb's `--json` output. Query tools SHALL be annotated read-only; `particular_define` idempotent; no tool destructive.

#### Scenario: Assert then recall
- **WHEN** a client calls `particular_define{label: "Project X"}`, then `claim_assert{particular_id: "Project X", content: "Uses Postgres"}`, then `knowledge_recall{particular_id: "Project X"}`
- **THEN** the recall result contains the asserted claim

#### Scenario: Publishing over MCP
- **WHEN** a client calls `knowledge_publish{claim_ids: [<clm>, <syn>], scope: "organisation"}`
- **THEN** one promotion record is written and the result matches the `publish` verb's `--json` output

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
The `initialize` response SHALL include `instructions` consisting of a short header naming the bound workspace (root and id) and stating that writes land as files for human review, followed by the embedded skill's body. The server SHALL also expose a prompt named `particulars-discipline` returning the same text.

#### Scenario: Instructions present
- **WHEN** a client initialises
- **THEN** `instructions` contains the workspace root and the phrase "Recall **before** you assert"

#### Scenario: Prompt available
- **WHEN** a client lists prompts and gets `particulars-discipline`
- **THEN** the prompt's message text equals the instructions body

### Requirement: Concurrent writes keep the index consistent
Mutating tools SHALL be serialised by a server-wide lock so that concurrent calls never interleave load, write, and index update.

#### Scenario: Parallel asserts
- **WHEN** twenty `claim_assert` calls run concurrently
- **THEN** twenty claim files exist and `particulars index --check` passes
