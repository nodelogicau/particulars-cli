## ADDED Requirements

### Requirement: The skill teaches an inference to name its basis
The embedded skill body SHALL direct its reader, before asserting an `inferred` claim, to recall what the workspace holds and to pass the claims and syntheses it reasoned from as `--basis` (or `basis` over MCP). It SHALL state that a rule of thumb is an inferred claim whose basis is its instances, that a later counterexample is reconciled as a `qualification` synthesis citing the rule and the counterexample rather than by changing the rule, and that a basis is evidence and does not synthesise what it names. It SHALL NOT tell the reader to choose `held` to avoid naming a basis. The installed copy at `.claude/skills/particulars/SKILL.md` SHALL be regenerated from the binary so the CI check passes.

#### Scenario: An inference after recall
- **WHEN** a reader following the skill concludes something from three claims it recalled
- **THEN** it asserts the conclusion with `--evidential inferred` and those three ids as `--basis`

#### Scenario: A counterexample
- **WHEN** a reader following the skill finds a claim that contradicts a rule of thumb
- **THEN** it creates a `qualification` synthesis citing both, and leaves the rule unchanged
