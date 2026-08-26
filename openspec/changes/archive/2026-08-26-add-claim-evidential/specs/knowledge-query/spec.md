## ADDED Requirements

### Requirement: Recall entries carry the warrant
Each recall entry for a claim SHALL include `evidential` when the file declares one, and SHALL omit it when the file predates the field — so a consumer sees exactly the three-values-or-absent that the file says, and `undeclared` remains a validator's report rather than a value that travels.

#### Scenario: Declared and undeclared side by side
- **WHEN** `recall` returns one claim carrying `evidential: held` and one written before the field existed
- **THEN** the first entry carries `"evidential": "held"` and the second carries no `evidential` key
