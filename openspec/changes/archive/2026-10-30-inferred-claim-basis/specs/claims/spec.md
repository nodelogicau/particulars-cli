## MODIFIED Requirements

### Requirement: Assert a claim
`particulars claim assert --subject <particular> (--content <text> | --content-file <path|->) --evidential observed|inferred|held [--basis <id>]... [--author <id|uri|name>] [--harness] [--model] [--document <uri>] [--document-author <id|uri|name>] [--document-hash <sha256:…> | --hash-document] [--quote <text> | --quote-file <path|->] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new claim file with `type: claim`, `subject` set to the resolved particular id, `source` populated from flags/environment/defaults, `context.scope` defaulting to `dkf.yaml` `defaults.scope`, `timestamp` defaulting to the current UTC time, and SHALL update `index.yaml`. `--evidential` is required, has no default, and accepts exactly `observed`, `inferred`, or `held`; without it the command SHALL exit 2 naming the three values and what each means. `--confidence` together with `--evidential held` SHALL be a usage error: a position is not on the scale confidence describes. The resolved `source` SHALL contain at least one of `author` or `harness`; if neither is available the command SHALL exit 2 naming both `DKF_AUTHOR` and `DKF_HARNESS`. `--author` and `--document-author` SHALL accept a particular id, URI, or bare name, resolved for writing per `source-attribution` — a defined particular is written as its `uri`; an unknown id exits 3; an explicitly given ambiguous name exits 2 listing candidates. The result SHALL include the full claim.

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

## ADDED Requirements

### Requirement: An inferred claim may be asserted with its basis
`claim assert` SHALL accept `--basis <id>` any number of times, and the MCP `claim_assert` tool SHALL accept `basis`, a list of ids. Each id names a claim or synthesis the new claim was inferred from. The ids SHALL be written as `basis`, in the order given, after `evidential` and before `confidence`. Ids SHALL NOT be resolved from labels, URIs or aliases. Both the CLI and MCP SHALL refuse, without writing a file:
- a basis whose evidential is not `inferred`, as a usage error;
- the same id given twice, as a usage error;
- an id that matches no object, as not-found (exit 3);
- an id that names a particular, merge or promotion, as a usage error.

A retracted id SHALL be accepted, and the result SHALL carry a `stale_claim` warning naming it. When `evidential` is `inferred` and no basis is given, the claim SHALL be written and the result SHALL carry an `inferred_without_basis` warning. When the new claim's scope is wider than the effective scope of any basis member, the claim SHALL be written and the result SHALL carry a `scope_wider_than_inputs` warning naming the narrower members.

#### Scenario: A rule of thumb with its instances
- **WHEN** `claim assert --subject certs --content "Renewals go wrong when the TTL is long" --evidential inferred --basis clm_a --basis clm_b --basis clm_c` is run and all three exist
- **THEN** the file carries `basis: [clm_a, clm_b, clm_c]` between `evidential` and `confidence`, and its index entry carries the same `basis`

#### Scenario: Basis on an observation
- **WHEN** `claim assert … --evidential observed --basis clm_a` is run
- **THEN** the command exits 2 and no file is written

#### Scenario: Unknown basis id
- **WHEN** `--basis clm_missing` is given and no such object exists
- **THEN** the command exits 3 and no file is written

#### Scenario: Basis names a particular
- **WHEN** `--basis par_x` is given and `par_x` is a particular
- **THEN** the command exits 2 and no file is written

#### Scenario: Duplicate basis id
- **WHEN** `--basis clm_a --basis clm_a` is given
- **THEN** the command exits 2 and no file is written

#### Scenario: A retracted basis member
- **WHEN** `--basis clm_a` is given and `clm_a` is retracted
- **THEN** the claim is written and the result carries a `stale_claim` warning naming `clm_a`

#### Scenario: Inferred without a basis
- **WHEN** `claim assert … --evidential inferred` is run with no `--basis`
- **THEN** the claim is written, the command exits 0, and the result carries an `inferred_without_basis` warning

#### Scenario: The same rules over MCP
- **WHEN** a client calls `claim_assert` with `evidential: held` and `basis: [clm_a]`
- **THEN** the result is a usage error naming `basis` and no claim is written
