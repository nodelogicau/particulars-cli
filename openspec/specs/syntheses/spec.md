# Syntheses

## Purpose

Recording syntheses that reconcile thesis and antithesis inputs, with mandatory `unresolved` and produced-by attribution.

## Requirements

### Requirement: Create a synthesis
`particulars synthesis create --subject <particular> (--content <text> | --content-file <path|->) --input <id>:<role>[:<weight>]... --unresolved <text> [--method <name>] [--author] [--harness] [--model] [--document <uri>] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new synthesis file with `type: synthesis`, `inputs` as a list of `{id, role, weight}`, `unresolved`, `source: {author?, harness, model?, document?}`, `method` (default `reconciliation`), and context/timestamp/confidence as for claims, and SHALL update `index.yaml`. The file SHALL NOT contain a `produced-by` key.

#### Scenario: Thesis and antithesis reconciled
- **WHEN** `synthesis create --subject "Project X" --content "..." --input clm_A:thesis --input clm_B:antithesis --unresolved "Compliance basis unsourced" --harness claude --model claude-sonnet-4-6` is run
- **THEN** a new `syn_` file exists with two inputs whose weights default to `primary`, `source.harness: claude`, `source.model: claude-sonnet-4-6`, `method: reconciliation`, no `produced-by` key, and an index entry listing both input ids

#### Scenario: Author recorded alongside harness
- **WHEN** `synthesis create ... --author ben --harness claude ...` is run
- **THEN** the synthesis has `source: {author: ben, harness: claude}`

### Requirement: Input validation
Each `--input` SHALL reference an existing claim or synthesis; `role` SHALL be `thesis` or `antithesis`; `weight` SHALL be `primary` or `qualifying` (default `primary`). At least one input SHALL be given. An input id that does not exist SHALL exit 3; an invalid role or weight SHALL exit 2. An input referencing a particular SHALL exit 2.

#### Scenario: Unknown input
- **WHEN** `synthesis create ... --input clm_missing:thesis ...` is run
- **THEN** the command exits with code 3 and writes nothing

#### Scenario: Invalid role
- **WHEN** `synthesis create ... --input clm_A:support ...` is run
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Qualifying weight
- **WHEN** `--input clm_C:thesis:qualifying` is given
- **THEN** the written input has `weight: qualifying`

### Requirement: Unresolved is mandatory
`--unresolved` SHALL be required and non-blank. The exact string `None identified` SHALL be the documented conventional value meaning the producer considered the question and found nothing outstanding; the CLI SHALL accept it like any other non-blank value and SHALL name it in `synthesis create --help`.

#### Scenario: Missing unresolved
- **WHEN** `synthesis create` is run without `--unresolved`
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Blank unresolved
- **WHEN** `synthesis create ... --unresolved "   "` is run
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Conventional empty value
- **WHEN** `synthesis create ... --unresolved "None identified"` is run
- **THEN** the synthesis is written with `unresolved: None identified`

### Requirement: Source attribution
A synthesis's `source.harness` SHALL be required, resolved from `--harness`, `DKF_HARNESS`, or `dkf.yaml` `defaults.source.harness`; `source.author`, `source.model`, and `source.document` are optional and resolved the same way (`document` from the flag only). A synthesis whose resolved `source` lacks `harness` SHALL be refused even when `author` is present.

#### Scenario: Harness unavailable
- **WHEN** `synthesis create` is run with no `--harness`, no `DKF_HARNESS`, and no default harness configured
- **THEN** the command exits with code 2 naming `DKF_HARNESS`

#### Scenario: Author alone is insufficient
- **WHEN** `synthesis create ... --author ben` is run with no harness resolvable
- **THEN** the command exits with code 2

### Requirement: Citing retracted inputs
Creating a synthesis whose input is retracted SHALL be permitted. The command SHALL include a `warnings` list in its result identifying each retracted input.

#### Scenario: Retracted thesis cited
- **WHEN** `clm_A` is retracted and `synthesis create ... --input clm_A:thesis --input clm_B:antithesis ...` is run
- **THEN** the synthesis is written and the result contains a warning naming `clm_A`

### Requirement: Synthesis is a claim
A synthesis SHALL be usable anywhere a claim is: as an input to another synthesis, as a target of `claim retract`, and in `recall` and `lineage` results.

#### Scenario: Synthesis as input
- **WHEN** `synthesis create ... --input syn_B:thesis ...` is run and `syn_B` exists
- **THEN** the new synthesis is written with `syn_B` as an input
