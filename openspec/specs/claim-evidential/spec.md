# Claim Evidential

## Purpose

What backs a claim, declared: the evidential axis and its three values, required on write with no default and lenient on read with `undeclared` as a reader's report, the definition of `confidence` and its exclusion from positions, the asymmetry by which a synthesis declares nothing because it is backed by argument from its inputs, and the closed `method` vocabulary naming the kind of question a synthesis settled. In a format written mostly by agents, the highest-risk artifact is a fluent judgement presented — for want of another register — as fact.

## Requirements

### Requirement: Every new claim declares what backs it
A claim SHALL carry `evidential`, one of `observed` (someone or something looked), `inferred` (derived by reasoning from other claims), or `held` (nothing external backs it; it is a position), serialised between `timestamp` and `confidence`. Writers SHALL refuse to create a claim without it, and SHALL NOT choose a value on the caller's behalf: there is no default, because if absence meant `observed` the laziest path would produce the most authoritative-looking output. Any value outside the three SHALL be rejected wherever it appears.

#### Scenario: Asserting with a warrant
- **WHEN** `claim assert --subject "Project X" --content "Uses Postgres 16" --evidential observed` is run
- **THEN** the claim file carries `evidential: observed` between `timestamp` and `confidence`

#### Scenario: No default
- **WHEN** `claim assert` is run without `--evidential`
- **THEN** the command exits 2 naming the three values and what each means, and no file is written

#### Scenario: Unknown value
- **WHEN** `--evidential opinion` is given
- **THEN** the command exits 2

### Requirement: Readers are lenient and report undeclared
A claim without `evidential` SHALL be read as valid, recallable, and citable; its warrant SHALL be reported as `undeclared`. `undeclared` is not a fourth value, SHALL NOT be writable, and is not a synonym for `observed`: it means the warrant cannot now be established. Claims are immutable, so there SHALL be no backfill mechanism — the distinction ages out as new claims are written.

#### Scenario: A pre-evidential workspace stays valid
- **WHEN** `validate` runs over a workspace written before the field existed
- **THEN** it reports no errors for the missing field, and reports `undeclared` in aggregate

#### Scenario: Undeclared is not writable
- **WHEN** `--evidential undeclared` is given
- **THEN** the command exits 2

### Requirement: Confidence is defined, and a position carries none
`confidence` is the inverse probability that the claim is mistaken. It applies to `observed` claims, whose evidence may have been misread, and `inferred` claims, whose reasoning may be invalid; it is undefined for `held`, because a position is not on that scale. Writers SHALL refuse `--confidence` together with `--evidential held`; `validate` SHALL report a file carrying both as an error, `confidence_on_held`, while still reading the file. Confidence on an `undeclared` claim SHALL be reported as unverified (`confidence_on_undeclared`), never rejected. There SHALL be no field recording strength of conviction; where it matters it belongs in `content`.

#### Scenario: Refused where the mistake is made
- **WHEN** `claim assert --evidential held --confidence 0.9` is run
- **THEN** the command exits 2 and no file is written

#### Scenario: An existing file carrying both
- **WHEN** a claim file carries `evidential: held` and `confidence` and `validate` runs
- **THEN** `validate` reports `confidence_on_held` as an error and exits 4, and the claim remains readable by every query verb

#### Scenario: Confidence on an undeclared claim
- **WHEN** a pre-evidential claim carries `confidence: 0.9`
- **THEN** `validate` reports `confidence_on_undeclared` as an informational note, not an error

### Requirement: A synthesis declares no evidential
A synthesis SHALL NOT carry `evidential` and writers SHALL NOT accept one for it: a synthesis is backed by argument from its inputs — that is what a synthesis is — so the value is implied and cannot vary. This mirrors the existing asymmetry by which `source.harness` is required on syntheses but not claims.

#### Scenario: Synthesis files are unchanged
- **WHEN** a synthesis is created
- **THEN** its file carries no `evidential` key

### Requirement: Method is a closed vocabulary
A synthesis's `method` SHALL be one of `reconciliation` (the inputs disagreed about a fact, and this settles it), `qualification` (each input is true in a different context), or `positions` (the inputs disagree in a way no evidence settles). Writers SHALL refuse other values; the default remains `reconciliation`. Readers SHALL accept files carrying other strings and `validate` SHALL warn `unknown_method`. A `held` claim is not exempt from reconciliation: two conflicting positions are still unsynthesised and still want a synthesis — a `positions` one.

#### Scenario: Positions synthesis
- **WHEN** `synthesis create --method positions` reconciles two held claims
- **THEN** the file carries `method: positions` and conflict reporting for the particular clears exactly as for any synthesis

#### Scenario: Unknown method refused at write
- **WHEN** `--method consensus` is given
- **THEN** the command exits 2

#### Scenario: Unknown method read leniently
- **WHEN** a file carries `method: consensus`
- **THEN** the file is read and `validate` reports an `unknown_method` warning
