## MODIFIED Requirements

### Requirement: Every new claim declares what backs it
A claim SHALL carry `evidential`, one of `observed` (someone or something looked), `inferred` (derived by reasoning from other claims), or `held` (nothing external backs it; it is a position), serialised between `timestamp` and `confidence`. Writers SHALL refuse to create a claim without it, and SHALL NOT choose a value on the caller's behalf: there is no default, because if absence meant `observed` the laziest path would produce the most authoritative-looking output. Any value outside the three SHALL be rejected wherever it appears. An `inferred` claim MAY name the claims and syntheses it was inferred from in `basis`, serialised after `evidential` and before `confidence`. The basis is the claim's evidence, in the sense that `source.document` is the evidence of an `observed` claim, and is never a set of synthesis inputs.

#### Scenario: Asserting with a warrant
- **WHEN** `claim assert --subject "Project X" --content "Uses Postgres 16" --evidential observed` is run
- **THEN** the claim file carries `evidential: observed` between `timestamp` and `confidence`

#### Scenario: An inference names what it rests on
- **WHEN** `claim assert … --evidential inferred --basis clm_a --confidence 0.7` is run
- **THEN** the file carries `evidential: inferred`, then `basis: [clm_a]`, then `confidence: 0.7`

#### Scenario: No default
- **WHEN** `claim assert` is run without `--evidential`
- **THEN** the command exits 2 naming the three values and what each means, and no file is written

#### Scenario: Unknown value
- **WHEN** `--evidential opinion` is given
- **THEN** the command exits 2
