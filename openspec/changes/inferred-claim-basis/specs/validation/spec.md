## MODIFIED Requirements

### Requirement: Referential checks
`validate` SHALL report an error when a claim or synthesis `subject` does not resolve to a particular file; when a synthesis input id, a claim `basis` id, or a `retracted.superseded-by` id does not resolve to a claim or synthesis file; when a synthesis input or a basis id references a particular, a merge, or a promotion; when the graph formed by synthesis `inputs` and claim `basis` together contains a cycle; when a claim's `basis` lists the claim's own id (`self_basis`); when a `basis` is an empty list (`empty_basis`); when two particulars share a `uri`; or when a merge record's two URIs are equal.

#### Scenario: Dangling subject
- **WHEN** a claim's `subject` is `par_missing`
- **THEN** `validate` reports an error with code `dangling_reference`

#### Scenario: Duplicate URI
- **WHEN** two particular files have the same `uri`
- **THEN** `validate` reports an error with code `duplicate_uri` naming both ids

#### Scenario: Cycle
- **WHEN** hand-edited files make `syn_A` cite `syn_B` and `syn_B` cite `syn_A`
- **THEN** `validate` reports an error with code `cycle`

#### Scenario: Dangling basis id
- **WHEN** a claim's `basis` lists `clm_missing` and no such file exists
- **THEN** `validate` reports a `dangling_reference` error against that claim

#### Scenario: Basis names a particular
- **WHEN** a hand-written claim's `basis` lists a particular id
- **THEN** `validate` reports an error against that claim

#### Scenario: A claim cites itself
- **WHEN** a hand-written claim's `basis` lists its own id
- **THEN** `validate` reports a `self_basis` error and no `cycle` error for the same cause

#### Scenario: Cycle through a basis
- **WHEN** hand-edited files make `clm_A`'s basis list `syn_B`, and `syn_B` cite `clm_A` as an input
- **THEN** `validate` reports an error with code `cycle`

#### Scenario: Empty basis
- **WHEN** a claim file carries `basis: []`
- **THEN** `validate` reports an `empty_basis` error and exits 4, and the claim is still read

### Requirement: Evidential findings
`validate` SHALL report: `confidence_on_held` as an **error** when a claim carries both `evidential: held` and `confidence` — the file stays readable by every query verb; `undeclared` at informational severity for every claim without an `evidential`, aggregated as a corpus fact, since every workspace written before the field existed carries it on every claim and it can never be cleared; `confidence_on_undeclared` at informational severity when an undeclared claim carries `confidence`, whose meaning cannot be established; `unknown_method` as a warning when a synthesis's `method` is outside the closed vocabulary; `basis_not_inferred` as an **error** when a `basis` appears on a claim whose `evidential` is not `inferred`, on a claim with no `evidential`, or on a synthesis — the file stays readable and citable, and the encoder never writes the field there; and `inferred_without_basis` at informational severity for every `inferred` claim without a `basis`, aggregated as a corpus fact, since no existing claim can be given one. An invalid `evidential` value in a file SHALL be an error.

#### Scenario: Undeclared is aggregate
- **WHEN** a workspace holds 88 claims written before the field existed
- **THEN** text output shows one `undeclared` line carrying the count, `--json` carries every finding, and the exit code is unaffected

#### Scenario: Held with confidence is the one error
- **WHEN** one claim carries `evidential: held` with `confidence: 0.9`
- **THEN** `validate` exits 4 reporting `confidence_on_held`, and `recall` still returns the claim

#### Scenario: Basis on a position
- **WHEN** a claim file declares `evidential: held` and carries a `basis`
- **THEN** `validate` reports `basis_not_inferred` as an error and exits 4, and `recall` still returns the claim

#### Scenario: Basis on a synthesis
- **WHEN** a synthesis file carries a `basis`
- **THEN** `validate` reports `basis_not_inferred`, and the synthesis is still read and citable

#### Scenario: Inferred claims that predate basis
- **WHEN** a workspace holds forty `inferred` claims without a basis
- **THEN** text output shows one `inferred_without_basis` line carrying the count of forty, and the exit code is unaffected

### Requirement: Advisory warnings
`validate` SHALL report a warning (not an error) when: a synthesis cites, directly or transitively, a retracted input (`stale_synthesis`); a non-retracted claim cites a retracted object, directly or transitively through `basis` and `inputs` (`stale_claim`); a claim's `basis` lists the same id more than once (`duplicate_basis`) — writers refuse it, but the format does not forbid it, so a hand-written file is warned rather than failed; a particular has no claims and is neither asserted by nor reported from — a particular whose merge class any object's `source.author` or `source.document.author` resolves to is not an orphan (`orphan_particular`); an object file's serialisation differs from the canonical form (`non_canonical`); a synthesis's provenance was read from a legacy `produced-by` block (`legacy_produced_by`); an id parses under the lenient grammar but is not a canonical UUIDv7 (`legacy_id`); a non-retracted merge names a URI with no local particular (`unknown_merge_uri`); two non-retracted merges join the same pair (`duplicate_merge`); a non-retracted promotion covers a retracted object (`promotion_of_retracted`); two non-retracted promotions name the same object at the same scope (`duplicate_promotion`); a non-retracted synthesis has an **effective** scope wider than the effective scope of any input it cites, or a non-retracted inferred claim has an effective scope wider than the effective scope of any basis member (`scope_wider_than_inputs`).

`scope_wider_than_inputs` SHALL compare effective scope on both sides. It SHALL name the narrower inputs or basis members with their effective scopes, and where an effective scope differs from the asserted one it SHALL name the promotion responsible, so a reader is sent to the file that caused the condition. It SHALL remain a warning — reasoning across scopes is legitimate and no tool can judge whether prose discloses its sources — and SHALL NOT be reported for a retracted synthesis or claim. Because the condition is a property of workspace state rather than of the synthesis or claim file, a promotion SHALL be able to create or clear it without either file changing.

#### Scenario: Synthesis wider than its inputs
- **WHEN** a non-retracted `organisation` synthesis cites a `personal` claim and an `organisation` claim, and no promotions exist
- **THEN** `validate` reports a `scope_wider_than_inputs` warning naming the personal claim and not the organisation one, and exits 0 if there are no errors

#### Scenario: Synthesis no wider than its inputs
- **WHEN** a `personal` synthesis cites an `organisation` claim
- **THEN** `validate` reports no `scope_wider_than_inputs` warning

#### Scenario: A promotion clears the condition
- **WHEN** the narrower input of a warned synthesis is promoted to the synthesis's effective scope
- **THEN** `validate` no longer reports `scope_wider_than_inputs` for it, though neither the synthesis nor the claim changed

#### Scenario: A promotion creates the condition
- **WHEN** a synthesis whose inputs match its scope is promoted to `public` and its inputs are not
- **THEN** `validate` reports `scope_wider_than_inputs` for it, naming the promotion that widened it

#### Scenario: An inferred claim wider than its basis
- **WHEN** a non-retracted `organisation` inferred claim has a basis of one `personal` claim, and no promotions exist
- **THEN** `validate` reports a `scope_wider_than_inputs` warning naming the personal basis member, and exits 0 if there are no errors

#### Scenario: A stale rule
- **WHEN** an inferred claim's basis lists a claim that has since been retracted
- **THEN** `validate` reports a `stale_claim` warning for the inferred claim and exits 0 if there are no errors

#### Scenario: An attributed particular is not an orphan
- **WHEN** a particular has no claims about it but every claim's `author` resolves to it
- **THEN** `validate` reports no `orphan_particular` for it

#### Scenario: Non-canonical file
- **WHEN** a claim file was hand-written with keys in a non-spec order
- **THEN** `validate` reports a `non_canonical` warning and exits 0 if there are no errors

#### Scenario: Legacy synthesis
- **WHEN** a synthesis file written by v0.1.1 carries `produced-by` and no `source`
- **THEN** `validate` reports a `legacy_produced_by` warning and no `non_canonical` warning for the same cause, and exits 0 if there are no errors

#### Scenario: Legacy id
- **WHEN** a claim has id `clm_01j9xk2p3q4r5s6t`
- **THEN** `validate` reports a `legacy_id` warning
