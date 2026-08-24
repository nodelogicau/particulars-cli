## MODIFIED Requirements

### Requirement: Advisory warnings
`validate` SHALL report a warning (not an error) when: a synthesis cites, directly or transitively, a retracted input (`stale_synthesis`); a particular has no claims (`orphan_particular`); an object file's serialisation differs from the canonical form (`non_canonical`); a synthesis's provenance was read from a legacy `produced-by` block (`legacy_produced_by`); an id parses under the lenient grammar but is not a canonical UUIDv7 (`legacy_id`); a non-retracted merge names a URI with no local particular (`unknown_merge_uri`); two non-retracted merges join the same pair (`duplicate_merge`); a non-retracted promotion covers a retracted object (`promotion_of_retracted`); two non-retracted promotions name the same object at the same scope (`duplicate_promotion`); a non-retracted synthesis has an **effective** scope wider than the effective scope of any input it cites (`scope_wider_than_inputs`).

`scope_wider_than_inputs` SHALL compare effective scope on both sides. It SHALL name the narrower inputs with their effective scopes, and where an effective scope differs from the asserted one it SHALL name the promotion responsible, so a reader is sent to the file that caused the condition. It SHALL remain a warning — reasoning across scopes is legitimate and no tool can judge whether prose discloses its sources — and SHALL NOT be reported for a retracted synthesis. Because the condition is a property of workspace state rather than of the synthesis file, a promotion SHALL be able to create or clear it without either file changing.

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

#### Scenario: Non-canonical file
- **WHEN** a claim file was hand-written with keys in a non-spec order
- **THEN** `validate` reports a `non_canonical` warning and exits 0 if there are no errors

#### Scenario: Legacy synthesis
- **WHEN** a synthesis file written by v0.1.1 carries `produced-by` and no `source`
- **THEN** `validate` reports a `legacy_produced_by` warning and no `non_canonical` warning for the same cause, and exits 0 if there are no errors

#### Scenario: Legacy id
- **WHEN** a claim has id `clm_01j9xk2p3q4r5s6t`
- **THEN** `validate` reports a `legacy_id` warning
