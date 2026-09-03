## ADDED Requirements

### Requirement: Unresolved listing tool
The server SHALL expose `unresolved_list(particular_id?, scope?, include_none?)` as a read-only tool whose description begins with "(particulars extension, not part of the DKF tool set)". Its structured result SHALL equal `particulars unresolved [<particular>] [--scope <s>] [--include-none] --json`: `{"entries": [...], "count": <n>}` with the entry shape and ordering defined by `unresolved-listing`. `particular_id` SHALL accept an id, URI, label, or alias; an unknown one SHALL be an error result with code `not_found`.

#### Scenario: Workspace-wide listing
- **WHEN** `unresolved_list{}` is called in a workspace whose current syntheses carry two real `unresolved` values and one `None identified`
- **THEN** the structured result has `count: 2`, ordered by the current syntheses' timestamps ascending

#### Scenario: Include the convention
- **WHEN** `unresolved_list{include_none: true}` is called in that workspace
- **THEN** the structured result has `count: 3`

#### Scenario: Unknown particular
- **WHEN** `unresolved_list{particular_id: "Nobody"}` is called
- **THEN** the result has `isError: true` and `error.code: "not_found"`
