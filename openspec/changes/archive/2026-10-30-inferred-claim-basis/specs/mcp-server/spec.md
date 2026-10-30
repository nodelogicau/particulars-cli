## MODIFIED Requirements

### Requirement: Conflict detection over a claim set
`conflict_detect` SHALL accept either `particular_id` or `claim_ids[]` (exactly one). With `claim_ids`, the given set SHALL be the universe: `current` is the most recent non-retracted synthesis in the set, `unsynthesised` the members not in its transitive inputs (following synthesis `inputs` only, never a claim's `basis`), `stale` the non-retracted member claims and syntheses that cite, directly or transitively through `inputs` and `basis`, a retracted object; `priority` is `|unsynthesised| + |stale|`. Staleness SHALL be followed through objects outside the set, since what a member rests on is not limited to what the caller named.

#### Scenario: Particular form
- **WHEN** `conflict_detect{particular_id}` is called
- **THEN** the result equals `particulars conflicts <particular> --json` for that particular (an empty `reports` list when below the reporting threshold)

#### Scenario: Claim-set form
- **WHEN** `conflict_detect{claim_ids: [clm_A, clm_B, syn_C]}` is called where `syn_C` cites only `clm_A`
- **THEN** the result has `current: syn_C` and `unsynthesised: [clm_B]`

#### Scenario: A stale rule in a claim set
- **WHEN** `conflict_detect{claim_ids: [clm_rule, clm_B]}` is called, `clm_rule`'s basis lists `clm_a`, which is not in the set, and `clm_a` is retracted
- **THEN** the result lists `clm_rule` in `stale`

#### Scenario: Both or neither
- **WHEN** `conflict_detect` is called with both parameters, or neither
- **THEN** the result is an error with code `usage`
