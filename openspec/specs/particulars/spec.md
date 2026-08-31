# Particulars

## Purpose

Defining and resolving particulars — the identifiable things claims are about — including URI minting and idempotency on URI.

## Requirements

### Requirement: Define a particular
`particulars particular define --label <label> [--uri <uri>] [--alias <alias>]...` SHALL create a particular file containing `id`, `type: particular`, `uri`, `label`, and `aliases` (omitted when empty). The result SHALL include the particular and a `created` boolean.

#### Scenario: New particular with explicit URI
- **WHEN** `particular define --label "Project X" --uri https://www.wikidata.org/entity/Q1` is run and no particular has that URI
- **THEN** a new `par_` file is written with that `uri` and `label`, and the result has `created: true`

### Requirement: URI minting
When `--uri` is omitted, the CLI SHALL mint `uri = <base> + <slug>` where `base` is `workspace.base-uri` from `dkf.yaml` if set, otherwise `urn:dkf:<workspace.id>:`, and `slug` is the label lowercased, folded to ASCII, with runs of non-alphanumeric characters replaced by a single hyphen and leading/trailing hyphens removed. An empty slug SHALL be a usage error.

#### Scenario: Minted under base URI
- **WHEN** `dkf.yaml` has `base-uri: https://example.com/particulars/` and `particular define --label "Billing Service"` is run
- **THEN** the particular's `uri` is `https://example.com/particulars/billing-service`

#### Scenario: Minted under URN fallback
- **WHEN** `dkf.yaml` has no `base-uri`, `workspace.id` is `W`, and `particular define --label "Auth & Sessions"` is run
- **THEN** the particular's `uri` is `urn:dkf:W:auth-sessions`

#### Scenario: Empty slug rejected
- **WHEN** `particular define --label "!!!"` is run without `--uri`
- **THEN** the command exits with code 2

### Requirement: Define is idempotent on URI
If a particular with the resolved URI already exists, `define` SHALL NOT create a new one. It SHALL update the existing particular: `label` is replaced by the supplied label, and `aliases` becomes the union of the existing aliases, the supplied aliases, and the previous label if it differs from the new one. The result SHALL have `created: false` and the existing `id`.

#### Scenario: Same label, second session
- **WHEN** `particular define --label "Project X"` is run twice in the same workspace
- **THEN** only one particular file exists and the second result has `created: false` with the same id as the first

#### Scenario: Label change preserved as alias
- **WHEN** a particular has `label: ProjectX` and `define --uri <same-uri> --label "Project X"` is run
- **THEN** the particular's `label` is `Project X` and `aliases` contains `ProjectX`

### Requirement: Resolve a particular
`particulars particular resolve <query>` SHALL return all particulars whose `id` or `uri` equals the query exactly, or whose `uri` is joined to the query by non-retracted merge records, or whose `label` or any alias equals the query case-insensitively. The result SHALL be `{"matches": [...]}` in JSON mode. If no particular matches, the command SHALL exit with code 3.

#### Scenario: Resolve by alias, case-insensitive
- **WHEN** a particular has alias `project_x` and `particular resolve PROJECT_X` is run
- **THEN** that particular is returned with exit code 0

#### Scenario: Resolve a merged URI
- **WHEN** a non-retracted merge joins `urn:dkf:W:jane` with a particular's `uri` and `particular resolve urn:dkf:W:jane` is run
- **THEN** that particular is returned

#### Scenario: No match
- **WHEN** `particular resolve nothing-here` is run and nothing matches
- **THEN** the command exits with code 3 and, in JSON mode, stderr carries `error.code: "not_found"`

### Requirement: Subject references accept id, URI, or label
Wherever a verb takes a particular as a subject (`claim assert`, `synthesis create`, `recall`, `conflicts`), the CLI SHALL accept an `id`, `uri`, `label`, or alias and resolve it using the same rules as `particular resolve`. Zero matches SHALL exit 3; more than one match SHALL exit 2 with an error listing the candidate ids.

#### Scenario: Ambiguous subject
- **WHEN** two particulars both have alias `auth` and `recall auth` is run
- **THEN** the command exits with code 2 and the error lists both ids
