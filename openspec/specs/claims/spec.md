# Claims

## Purpose

Asserting claims with provenance and retracting claims or syntheses via an append-only, permanent retraction block.

## Requirements

### Requirement: Assert a claim
`particulars claim assert --subject <particular> (--content <text> | --content-file <path|->) --evidential observed|inferred|held [--author <id|uri|name>] [--harness] [--model] [--document <uri>] [--document-author <id|uri|name>] [--document-hash <sha256:…> | --hash-document] [--quote <text> | --quote-file <path|->] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new claim file with `type: claim`, `subject` set to the resolved particular id, `source` populated from flags/environment/defaults, `context.scope` defaulting to `dkf.yaml` `defaults.scope`, `timestamp` defaulting to the current UTC time, and SHALL update `index.yaml`. `--evidential` is required, has no default, and accepts exactly `observed`, `inferred`, or `held`; without it the command SHALL exit 2 naming the three values and what each means. `--confidence` together with `--evidential held` SHALL be a usage error: a position is not on the scale confidence describes. The resolved `source` SHALL contain at least one of `author` or `harness`; if neither is available the command SHALL exit 2 naming both `DKF_AUTHOR` and `DKF_HARNESS`. `--author` and `--document-author` SHALL accept a particular id, URI, or bare name, resolved for writing per `source-attribution` — a defined particular is written as its `uri`; an unknown id exits 3; an explicitly given ambiguous name exits 2 listing candidates. The result SHALL include the full claim.

`--document-hash`, `--hash-document`, `--quote`, `--quote-file`, and `--document-author` SHALL require `--document` and SHALL be usage errors without it. `--hash-document` SHALL compute the hash from the local file the document resolves to, and SHALL be a usage error when it does not resolve. `--document-hash` and `--hash-document` SHALL be mutually exclusive, as SHALL `--quote` and `--quote-file`.

#### Scenario: Claim with evidence
- **WHEN** `claim assert --subject "Project X" --content "Uses Postgres 16" --evidential observed --document docs/db.md` is run
- **THEN** a claim file is written whose `source.document` is the scalar `docs/db.md` and whose `evidential` is `observed`

#### Scenario: Verifiable claim
- **WHEN** the same command adds `--hash-document --quote "Postgres 16 in every environment"`
- **THEN** `source.document` is a mapping carrying `ref`, a `sha256:` hash computed from the local file, and the quote

#### Scenario: Reported testimony
- **WHEN** `claim assert … --document "conversation with Jane, 2026-08-30" --document-author jane --quote "we went microservices in Q2"` is run and exactly one particular has alias `jane`
- **THEN** `source.document` is a mapping whose `author` is that particular's `uri`, in the order `ref`, `author`, `quote`

#### Scenario: Document author without a document
- **WHEN** `--document-author jane` is given without `--document`
- **THEN** the command exits 2

#### Scenario: Locator without a document
- **WHEN** `--quote` is given without `--document`
- **THEN** the command exits 2

#### Scenario: A position carries no confidence
- **WHEN** `claim assert --evidential held --confidence 0.8` is run
- **THEN** the command exits 2 and no file is written

### Requirement: Retract a claim or synthesis
`particulars retract <id> --reason <text> [--author] [--harness] [--model] [--superseded-by <id>]` SHALL append a `retracted` block containing `timestamp` (current UTC), `reason`, `source` (at least one of `author`/`harness`), and optional `superseded-by` to the existing object file without altering any existing bytes, then re-parse the file to confirm validity, restoring the original bytes on failure. It SHALL update the object's index entry with `retracted: true`. The target MAY be a claim, a synthesis, or a merge record (see `merge-records`). `particulars claim retract` SHALL remain available as an alias with identical behaviour.

#### Scenario: Retract a claim
- **WHEN** `retract clm_A --reason "Port was 8443"` is run
- **THEN** `claims/clm_A.yaml` consists of its previous bytes followed by a `retracted` block, and the index entry for `clm_A` has `retracted: true`

#### Scenario: Retract a synthesis
- **WHEN** `retract syn_B --reason "Inputs were misread"` is run
- **THEN** `syntheses/syn_B.yaml` gains a `retracted` block

#### Scenario: Alias still works
- **WHEN** `claim retract clm_A --reason r` is run
- **THEN** the behaviour is identical to `retract clm_A --reason r`

#### Scenario: Superseded-by must exist
- **WHEN** `retract clm_A --reason r --superseded-by clm_missing` is run
- **THEN** the command exits with code 3 and `clm_A` is unchanged

#### Scenario: Double retraction refused
- **WHEN** `retract clm_A --reason r` is run and `clm_A` already has a `retracted` block
- **THEN** the command exits with code 1 and the file is unchanged

#### Scenario: Reason required
- **WHEN** `retract clm_A` is run without `--reason`
- **THEN** the command exits with code 2

#### Scenario: Agent retraction
- **WHEN** `retract clm_A --reason r` is run with `DKF_HARNESS=claude` and no author anywhere
- **THEN** the `retracted.source` is `{harness: claude}` and the retraction is valid

### Requirement: Retraction is permanent
The CLI SHALL provide no command that removes or edits a `retracted` block. Reinstatement is expressed by asserting a new claim or creating a synthesis that cites the retracted object.

#### Scenario: No un-retract verb
- **WHEN** the user inspects the CLI command tree
- **THEN** no verb exists that removes a `retracted` block
