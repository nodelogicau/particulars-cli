## ADDED Requirements

### Requirement: Recall by author
`particulars recall [--author <id|uri|label|alias>]` SHALL return the non-retracted claims and syntheses asserted by or reported from the resolved particular's merge equivalence class — an object is asserted by the particular its `source.author` resolves to and reported from the particular its `source.document.author` resolves to, per `source-attribution`. Each returned entry SHALL carry `relations`, a list containing `asserted`, `reported`, or both, never empty. `--author` SHALL be a sufficient selector on its own and SHALL be combinable with a subject, `--topic`, `--scope`, and `--include-retracted`, results satisfying every filter given. An author query resolving to no particular SHALL exit 3; one matching several SHALL exit 2 listing the candidate ids. Entries returned without `--author` SHALL NOT carry `relations`.

#### Scenario: Everything Jane said
- **WHEN** `recall --author jane` is run where one claim's `source.author` and another claim's `source.document.author` resolve to Jane's class
- **THEN** both are returned, the first with `relations: [asserted]`, the second with `relations: [reported]`

#### Scenario: Both relations on one object
- **WHEN** a claim's `source.author` and `source.document.author` both resolve to Ben's class and `recall --author ben` is run
- **THEN** that entry carries `relations: [asserted, reported]`

#### Scenario: Author with a subject
- **WHEN** `recall "Project X" --author jane` is run
- **THEN** only objects whose subject is in Project X's class and which are asserted by or reported from Jane are returned

#### Scenario: Attribution across a merge
- **WHEN** a person's URN and ORCID are merged, a claim carries the URN as `source.author`, and `recall --author <ORCID>` is run
- **THEN** the claim is returned, its file unchanged
