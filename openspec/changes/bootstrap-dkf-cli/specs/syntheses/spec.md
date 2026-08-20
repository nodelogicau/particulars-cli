## ADDED Requirements

### Requirement: Create a synthesis
`particulars synthesis create --subject <particular> (--content <text> | --content-file <path|->) --input <id>:<role>[:<weight>]... --unresolved <text> [--method <name>] [--harness] [--model] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new synthesis file with `type: synthesis`, `inputs` as a list of `{id, role, weight}`, `unresolved`, `produced-by: {harness, model}`, `method` (default `reconciliation`), and context/timestamp/confidence as for claims, and SHALL update `index.yaml`.

#### Scenario: Thesis and antithesis reconciled
- **WHEN** `synthesis create --subject "Project X" --content "..." --input clm_A:thesis --input clm_B:antithesis --unresolved "Compliance basis unsourced" --harness claude --model claude-sonnet-4-6` is run
- **THEN** a new `syn_` file exists with two inputs whose weights default to `primary`, `produced-by.harness: claude`, `method: reconciliation`, and an index entry listing both input ids

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
`--unresolved` SHALL be required and non-empty. A synthesis that reconciles everything SHALL still state so explicitly (for example `--unresolved "None identified"`).

#### Scenario: Missing unresolved
- **WHEN** `synthesis create` is run without `--unresolved`
- **THEN** the command exits with code 2 and writes nothing

### Requirement: Produced-by attribution
`produced-by.harness` SHALL be required, resolved from `--harness`, `DKF_HARNESS`, or `dkf.yaml` `defaults.source.harness`; `produced-by.model` is optional and resolved the same way.

#### Scenario: Harness unavailable
- **WHEN** `synthesis create` is run with no `--harness`, no `DKF_HARNESS`, and no default harness configured
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
