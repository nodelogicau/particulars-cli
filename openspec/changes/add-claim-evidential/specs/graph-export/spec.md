## ADDED Requirements

### Requirement: The brief marks the warrant
A `held` claim rendered in an item's brief SHALL be marked as a position, and a claim whose warrant is undeclared SHALL be marked as such, inline where the existing unsynthesised marker appears; `observed` and `inferred` claims render unmarked. This is what the evidential exists for downstream: without the marker, a consumer citing the brief cannot distinguish a fluent position from an observed fact.

#### Scenario: A position in the brief
- **WHEN** an exported particular's supporting claims include one with `evidential: held`
- **THEN** that line carries a position marker and no confidence, and the observed claims' lines are unmarked
