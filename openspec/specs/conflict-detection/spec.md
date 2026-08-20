# Conflict Detection

## Purpose

Structural (non-semantic) detection of unsynthesised claims and stale syntheses per particular, with a reporting threshold, priority ordering, and a CI exit code.

## Requirements

### Requirement: Structural conflict detection
`particulars conflicts [<particular>] [--fail-on-conflicts]` SHALL compute, for the given particular (or every particular when omitted): `current` = the non-retracted synthesis about the particular with the highest id, or none; `reconciled` = the transitive closure of `current.inputs`; `unsynthesised` = non-retracted claims and syntheses about the particular that are neither `current` nor in `reconciled`; `stale` = non-retracted syntheses about the particular having at least one directly retracted input. The CLI SHALL NOT attempt to judge whether contents contradict; it reports structure only.

#### Scenario: Claim asserted after synthesis
- **WHEN** `syn_C` (inputs A, B) is current for Project X and a later claim `clm_D` is asserted about Project X
- **THEN** `conflicts "Project X"` reports `current: syn_C` and `unsynthesised: [clm_D]`

#### Scenario: Older claim never reconciled
- **WHEN** `clm_A`, `clm_B`, `clm_Z` exist about Project X and `syn_C` cites only A and B
- **THEN** `conflicts "Project X"` lists `clm_Z` as unsynthesised

#### Scenario: Stale synthesis
- **WHEN** `syn_C` cites `clm_A` and `clm_A` is retracted
- **THEN** `conflicts "Project X"` lists `syn_C` in `stale`

### Requirement: Reporting threshold
A particular SHALL be reported when `current` exists and `unsynthesised` is non-empty, or when `current` does not exist and `unsynthesised` has two or more members, or when `stale` is non-empty. Particulars not meeting the threshold SHALL be omitted.

#### Scenario: Single unsynthesised claim is not a conflict
- **WHEN** exactly one non-retracted claim exists about Project X and no synthesis
- **THEN** `conflicts` does not report Project X

#### Scenario: Two unsynthesised claims without a synthesis
- **WHEN** two non-retracted claims exist about Project X and no synthesis
- **THEN** `conflicts` reports Project X with both claims in `unsynthesised` and `current: null`

### Requirement: Priority ordering
Reported particulars SHALL be ordered by `|unsynthesised| + |stale|` descending, ties broken by particular id ascending. Each report SHALL include a numeric `priority` equal to that sum.

#### Scenario: Ordering
- **WHEN** Project X has three unsynthesised claims and Project Y has one unsynthesised claim and one stale synthesis
- **THEN** Project X (priority 3) is listed before Project Y (priority 2)

### Requirement: CI exit code
With `--fail-on-conflicts`, the command SHALL exit with code 4 if any particular is reported, and 0 otherwise. Without the flag it SHALL exit 0 regardless.

#### Scenario: Fail flag
- **WHEN** `conflicts --fail-on-conflicts` is run and at least one particular is reported
- **THEN** the command exits with code 4
