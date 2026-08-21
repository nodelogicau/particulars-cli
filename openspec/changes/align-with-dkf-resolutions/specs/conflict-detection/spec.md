## MODIFIED Requirements

### Requirement: Structural conflict detection
`particulars conflicts [<particular>] [--fail-on-conflicts]` SHALL compute, for the merge equivalence class of the given particular (or for every class when omitted, reported once per class keyed by its lowest particular id with `members` listed): `current` = the non-retracted synthesis whose `subject` is in the class with the greatest (`timestamp`, id), or none; `reconciled` = the transitive closure of `current.inputs`; `unsynthesised` = non-retracted claims and syntheses whose `subject` is in the class that are neither `current` nor in `reconciled`; `stale` = non-retracted syntheses whose `subject` is in the class that cite, directly or transitively through their inputs, at least one retracted object. A claim about another particular cited as an input SHALL NOT be considered synthesised for its own particular. The CLI SHALL NOT attempt to judge whether contents contradict; it reports structure only.

#### Scenario: Claim asserted after synthesis
- **WHEN** `syn_C` (inputs A, B) is current for Project X and a later claim `clm_D` is asserted about Project X
- **THEN** `conflicts "Project X"` reports `current: syn_C` and `unsynthesised: [clm_D]`

#### Scenario: Older claim never reconciled
- **WHEN** `clm_A`, `clm_B`, `clm_Z` exist about Project X and `syn_C` cites only A and B
- **THEN** `conflicts "Project X"` lists `clm_Z` as unsynthesised

#### Scenario: Stale synthesis
- **WHEN** `syn_C` cites `clm_A` and `clm_A` is retracted
- **THEN** `conflicts "Project X"` lists `syn_C` in `stale`

#### Scenario: Transitively stale synthesis
- **WHEN** `syn_D` cites `syn_C`, `syn_C` cites `clm_A`, and `clm_A` is retracted
- **THEN** `conflicts "Project X"` lists both `syn_C` and `syn_D` in `stale`

#### Scenario: Current chosen by timestamp
- **WHEN** `syn_1` has `timestamp` 2026-03-01 and `syn_2`, minted later, has `timestamp` 2026-01-01, both non-retracted about Project X
- **THEN** `current` is `syn_1` and `syn_2` is in `unsynthesised`

#### Scenario: Conflicts across a merge
- **WHEN** A and B are merged, `syn_C` about A cites `clm_A`, and `clm_B` about B is not an input of `syn_C`
- **THEN** `conflicts A` reports `current: syn_C` and `unsynthesised: [clm_B]`

#### Scenario: Cross-particular input is not synthesised for its own particular
- **WHEN** `clm_lib` (subject Library Y) is an input to `syn_proj` (subject Project X) and Y has one other claim
- **THEN** `conflicts "Library Y"` lists `clm_lib` among its unsynthesised claims
