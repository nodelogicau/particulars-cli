## ADDED Requirements

### Requirement: Evidential findings
`validate` SHALL report: `confidence_on_held` as an **error** when a claim carries both `evidential: held` and `confidence` — the one mechanically checkable rule in this area, and the file stays readable by every query verb; `undeclared` at informational severity for every claim without an `evidential`, aggregated as a corpus fact, since every workspace written before the field existed carries it on every claim and it can never be cleared; `confidence_on_undeclared` at informational severity when an undeclared claim carries `confidence`, whose meaning cannot be established; and `unknown_method` as a warning when a synthesis's `method` is outside the closed vocabulary. An invalid `evidential` value in a file SHALL be an error.

#### Scenario: Undeclared is aggregate
- **WHEN** a workspace holds 88 claims written before the field existed
- **THEN** text output shows one `undeclared` line carrying the count, `--json` carries every finding, and the exit code is unaffected

#### Scenario: Held with confidence is the one error
- **WHEN** one claim carries `evidential: held` with `confidence: 0.9`
- **THEN** `validate` exits 4 reporting `confidence_on_held`, and `recall` still returns the claim
