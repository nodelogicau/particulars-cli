# Conflict Detection

## Purpose

Structural (non-semantic) detection of unsynthesised claims and stale syntheses per particular, with a reporting threshold, priority ordering, and a CI exit code.
## Requirements
### Requirement: Structural conflict detection
`particulars conflicts [<particular>] [--fail-on-conflicts]` SHALL compute, for the merge equivalence class of the given particular (or for every class when omitted, reported once per class keyed by its lowest particular id with `members` listed): `current` = the non-retracted synthesis whose `subject` is in the class with the greatest (`timestamp`, id), or none; `reconciled` = the transitive closure of `current.inputs`, following synthesis `inputs` only and never a claim's `basis`; `unsynthesised` = non-retracted claims and syntheses whose `subject` is in the class that are neither `current` nor in `reconciled`; `stale` = non-retracted claims and syntheses whose `subject` is in the class that cite, directly or transitively through `inputs` and `basis`, at least one retracted object. A claim about another particular cited as an input SHALL NOT be considered synthesised for its own particular. An object named in a claim's `basis` SHALL NOT thereby be considered synthesised, for its own particular or any other. The CLI SHALL NOT attempt to judge whether contents contradict; it reports structure only. The MCP `conflict_detect` tool SHALL compute the same sets, including over a claim set.

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

#### Scenario: Retracted basis member
- **WHEN** `clm_rule` declares `inferred` with `basis: [clm_a, clm_b]` and `clm_a` is retracted
- **THEN** `conflicts` lists `clm_rule` in `stale` for its particular, and its file is unchanged

#### Scenario: Staleness through a rule
- **WHEN** `syn_1` cites `clm_rule` as an input and a member of `clm_rule`'s basis is retracted
- **THEN** both `clm_rule` and `syn_1` are in `stale` for their particulars

#### Scenario: A basis member about another particular
- **WHEN** `clm_rule` about Project X has in its basis `clm_lib` about Library Y, and `clm_lib` is retracted
- **THEN** `clm_rule` is in `stale` for Project X

#### Scenario: Leaving the stale set
- **WHEN** a stale inferred claim is retracted with `superseded-by` pointing at a claim with a revised basis
- **THEN** it is no longer in `stale`, and its replacement is `unsynthesised` like any other

#### Scenario: Instances of a rule stay unsynthesised
- **WHEN** `clm_rule` about Project X lists `clm_a` and `clm_b`, also about Project X, in its basis, and Project X has no synthesis
- **THEN** `unsynthesised` for Project X is `[clm_a, clm_b, clm_rule]`

#### Scenario: A synthesis over a rule reconciles the rule only
- **WHEN** the current synthesis for Project X cites `clm_rule` as an input but not `clm_a` or `clm_b`
- **THEN** `clm_rule` is not unsynthesised, and `clm_a` and `clm_b` remain `unsynthesised`

#### Scenario: Current chosen by timestamp
- **WHEN** `syn_1` has `timestamp` 2026-03-01 and `syn_2`, minted later, has `timestamp` 2026-01-01, both non-retracted about Project X
- **THEN** `current` is `syn_1` and `syn_2` is in `unsynthesised`

#### Scenario: Conflicts across a merge
- **WHEN** A and B are merged, `syn_C` about A cites `clm_A`, and `clm_B` about B is not an input of `syn_C`
- **THEN** `conflicts A` reports `current: syn_C` and `unsynthesised: [clm_B]`

#### Scenario: Cross-particular input is not synthesised for its own particular
- **WHEN** `clm_lib` (subject Library Y) is an input to `syn_proj` (subject Project X) and Y has one other claim
- **THEN** `conflicts "Library Y"` lists `clm_lib` among its unsynthesised claims

### Requirement: Reporting threshold
A particular SHALL be reported when `current` exists and `unsynthesised` is non-empty, or when `current` does not exist and `unsynthesised` has two or more members, or when `stale` is non-empty. Particulars not meeting the threshold SHALL be omitted.

#### Scenario: Single unsynthesised claim is not a conflict
- **WHEN** exactly one non-retracted claim exists about Project X and no synthesis
- **THEN** `conflicts` does not report Project X

#### Scenario: Two unsynthesised claims without a synthesis
- **WHEN** two non-retracted claims exist about Project X and no synthesis
- **THEN** `conflicts` reports Project X with both claims in `unsynthesised` and `current: null`

### Requirement: Priority ordering
Reported particulars SHALL be ordered by `|unsynthesised| + |stale|` descending, ties broken by particular id ascending. Each report SHALL include a numeric `priority` equal to that sum. An object in both sets SHALL count once in each, since each set asks for a different action: a stale inferred claim needs a revised rule, and as an unsynthesised claim it needs a synthesis.

#### Scenario: Ordering
- **WHEN** Project X has three unsynthesised claims and Project Y has one unsynthesised claim and one stale synthesis
- **THEN** Project X (priority 3) is listed before Project Y (priority 2)

#### Scenario: A stale, unsynthesised rule
- **WHEN** Project X has no synthesis and its only claims are `clm_rule` and `clm_other`, and `clm_rule` is stale
- **THEN** Project X is reported with priority 3

### Requirement: CI exit code
With `--fail-on-conflicts`, the command SHALL exit with code 4 if any particular is reported, and 0 otherwise. Without the flag it SHALL exit 0 regardless.

#### Scenario: Fail flag
- **WHEN** `conflicts --fail-on-conflicts` is run and at least one particular is reported
- **THEN** the command exits with code 4

