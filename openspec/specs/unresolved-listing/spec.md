# Unresolved Listing

## Purpose

Listing what each current synthesis admits it could not settle, oldest first.

## Requirements

### Requirement: List current unresolved questions
`particulars unresolved [<particular>] [--include-none] [--scope <s>]` SHALL return one entry per merge equivalence class that has a `current` synthesis — the non-retracted synthesis whose `subject` is in the class with the greatest (`timestamp`, id), as defined for `conflicts` and `recall` — for the class of the given particular, or for every class when omitted, keyed by the class's lowest particular id. Each entry SHALL carry `particular`, `label`, `uri`, `members` when the class has more than one member, `synthesis` (the current synthesis's id), `timestamp`, `unresolved` (the current synthesis's `unresolved` text verbatim), and `unsynthesised` (the number of non-retracted claims and syntheses in the class that are neither `current` nor in its transitive inputs, as `conflicts` defines it). Classes with no current synthesis SHALL be omitted. Superseded syntheses SHALL NOT contribute entries.

#### Scenario: Only the current synthesis is reported
- **WHEN** `syn_1` (timestamp 2026-01-01, unresolved "A is open") and `syn_2` (timestamp 2026-02-01, inputs including `syn_1`, unresolved "B is open") exist about Project X
- **THEN** `unresolved` lists one entry for Project X with `synthesis: syn_2` and `unresolved: "B is open"`

#### Scenario: No synthesis, no entry
- **WHEN** Project Y has three claims and no synthesis
- **THEN** `unresolved` does not list Project Y

#### Scenario: Unsynthesised count accompanies the question
- **WHEN** `syn_C` is current for Project X and `clm_D` about Project X is not among its transitive inputs
- **THEN** the Project X entry has `unsynthesised: 1`

#### Scenario: Retracted current falls back to its predecessor
- **WHEN** `syn_2` is retracted and `syn_1` about the same particular is not
- **THEN** the entry reports `syn_1` and its `unresolved`

#### Scenario: Merged class reported once
- **WHEN** A and B are merged and `syn_C` about B is the only synthesis in the class
- **THEN** `unresolved` lists one entry keyed by the lower of A and B with `members` naming both and `synthesis: syn_C`

#### Scenario: Single particular
- **WHEN** `unresolved "Project X"` is run
- **THEN** only the entry for Project X's class is returned, and an unknown particular exits 3

### Requirement: Ordering
Entries SHALL be ordered by the current synthesis's `timestamp` ascending, ties broken by synthesis id ascending, so the longest-standing question is listed first.

#### Scenario: Oldest first
- **WHEN** Project X's current synthesis has timestamp 2026-03-01 and Project Y's has 2026-01-15
- **THEN** Project Y is listed before Project X

### Requirement: The conventional empty value is filtered
Entries whose `unresolved` is exactly the string `None identified` SHALL be omitted unless `--include-none` is given.

#### Scenario: None identified hidden by default
- **WHEN** Project X's current synthesis has `unresolved: None identified`
- **THEN** `unresolved` does not list Project X, and `unresolved --include-none` does

#### Scenario: Only the exact convention is filtered
- **WHEN** a current synthesis has `unresolved: nothing`
- **THEN** it is listed

### Requirement: Scope filter
With `--scope <s>`, only entries whose current synthesis has effective scope `s` SHALL be returned.

#### Scenario: Promoted synthesis visible at its promoted scope
- **WHEN** `syn_C` is asserted `personal` and promoted to `organisation`
- **THEN** `unresolved --scope organisation` lists it and `unresolved --scope personal` does not

### Requirement: Output and exit
In JSON mode the result SHALL be `{"entries": [...], "count": <n>}`. In text mode each entry SHALL be printed as a block headed by particular id, label, and the synthesis date, followed by the synthesis id, the `unresolved` text, `members` when present, and `unsynthesised` only when greater than zero. An empty result SHALL exit 0.

#### Scenario: Nothing open
- **WHEN** no class has a current synthesis with a reportable `unresolved`
- **THEN** the command exits 0, JSON output is `{"entries": [], "count": 0}`, and text output says nothing is unresolved

#### Scenario: Text block shape
- **WHEN** Project X has a current synthesis with one unsynthesised claim
- **THEN** text output for Project X shows the synthesis id, its `unresolved` text, and `unsynthesised: 1`
