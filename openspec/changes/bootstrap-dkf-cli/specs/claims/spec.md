## ADDED Requirements

### Requirement: Assert a claim
`particulars claim assert --subject <particular> (--content <text> | --content-file <path|->) [--author] [--harness] [--model] [--document <uri>] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new claim file with `type: claim`, `subject` set to the resolved particular id, `source` populated from flags/environment/defaults, `context.scope` defaulting to `dkf.yaml` `defaults.scope`, `timestamp` defaulting to the current UTC time, and SHALL update `index.yaml`. The result SHALL include the full claim.

#### Scenario: Minimal assertion
- **WHEN** `claim assert --subject "Project X" --content "Uses Postgres 16"` is run with `DKF_AUTHOR=agent`
- **THEN** a new `clm_` file exists with `subject` equal to Project X's id, `source.author: agent`, `context.scope` equal to the workspace default, a current `timestamp`, and `index.yaml` contains an entry for it

#### Scenario: Backdated assertion
- **WHEN** `claim assert ... --timestamp 2024-08-20T09:00:00Z` is run
- **THEN** the claim's `timestamp` is `2024-08-20T09:00:00Z` while its id encodes the current time

#### Scenario: Missing author
- **WHEN** `claim assert` is run with no `--author`, no `DKF_AUTHOR`, and no `defaults.source.author` in `dkf.yaml`
- **THEN** the command exits with code 2 and writes nothing

### Requirement: Retract a claim or synthesis
`particulars claim retract <id> --reason <text> [--author] [--harness] [--model] [--superseded-by <id>]` SHALL append a `retracted` block containing `timestamp` (current UTC), `reason`, `source`, and optional `superseded-by` to the existing object file without altering any existing bytes, then re-parse the file to confirm validity, restoring the original bytes on failure. It SHALL update the object's index entry with `retracted: true`. The target MAY be a claim or a synthesis.

#### Scenario: Retract a claim
- **WHEN** `claim retract clm_A --reason "Port was 8443"` is run
- **THEN** `claims/clm_A.yaml` consists of its previous bytes followed by a `retracted` block, and the index entry for `clm_A` has `retracted: true`

#### Scenario: Retract a synthesis
- **WHEN** `claim retract syn_B --reason "Inputs were misread"` is run
- **THEN** `syntheses/syn_B.yaml` gains a `retracted` block

#### Scenario: Superseded-by must exist
- **WHEN** `claim retract clm_A --reason r --superseded-by clm_missing` is run
- **THEN** the command exits with code 3 and `clm_A` is unchanged

#### Scenario: Double retraction refused
- **WHEN** `claim retract clm_A --reason r` is run and `clm_A` already has a `retracted` block
- **THEN** the command exits with code 1 and the file is unchanged

#### Scenario: Reason required
- **WHEN** `claim retract clm_A` is run without `--reason`
- **THEN** the command exits with code 2

### Requirement: Retraction is permanent
The CLI SHALL provide no command that removes or edits a `retracted` block. Reinstatement is expressed by asserting a new claim or creating a synthesis that cites the retracted object.

#### Scenario: No un-retract verb
- **WHEN** the user inspects the CLI command tree
- **THEN** no verb exists that removes a `retracted` block
