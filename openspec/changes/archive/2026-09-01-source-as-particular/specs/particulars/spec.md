## MODIFIED Requirements

### Requirement: Resolve a particular
`particulars particular resolve <query>` SHALL return all particulars whose `id` or `uri` equals the query exactly, or whose `uri` is joined to the query by non-retracted merge records, or whose `label` or any alias equals the query case-insensitively. The result SHALL be `{"matches": [...]}` in JSON mode. If no particular matches, the command SHALL exit with code 3.

#### Scenario: Resolve by alias, case-insensitive
- **WHEN** a particular has alias `project_x` and `particular resolve PROJECT_X` is run
- **THEN** that particular is returned with exit code 0

#### Scenario: Resolve a merged URI
- **WHEN** a non-retracted merge joins `urn:dkf:W:jane` with a particular's `uri` and `particular resolve urn:dkf:W:jane` is run
- **THEN** that particular is returned

#### Scenario: No match
- **WHEN** `particular resolve nothing-here` is run and nothing matches
- **THEN** the command exits with code 3 and, in JSON mode, stderr carries `error.code: "not_found"`
