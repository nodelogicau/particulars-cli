## MODIFIED Requirements

### Requirement: Promotion does not cascade
Promoting a synthesis SHALL NOT promote the assertions it cites, and promoting an inferred claim SHALL NOT promote the members of its `basis`. Implementations SHALL NOT offer an option that promotes a synthesis together with its inputs, or an inferred claim together with its basis, in one act, because that is the silent lineage-widening the rule exists to prevent; promoting a chain SHALL require naming each object.

#### Scenario: Inputs stay where they were
- **WHEN** a synthesis citing two `personal` claims is promoted to `organisation`
- **THEN** the synthesis's effective scope is `organisation` and both claims remain `personal`

#### Scenario: A basis stays where it was
- **WHEN** an inferred claim whose basis lists two `personal` claims is promoted to `public`
- **THEN** only the inferred claim becomes `public`, the basis members remain `personal`, and the promotion reports `scope_wider_than_inputs` naming them

## ADDED Requirements

### Requirement: A basis is treated as inputs for disclosure
For `scope_wider_than_inputs`, the members of an inferred claim's `basis` SHALL be treated as that claim's inputs. The condition SHALL be evaluated for an inferred claim when it is asserted, during `validate`, and whenever `publish` or `knowledge_publish` writes a promotion that covers the claim or covers any of its basis members. The findings that hold after the promotion SHALL be carried in the result exactly as for a synthesis, so a finding the promotion cleared is absent from it. The condition SHALL remain a warning and SHALL NOT prevent the write.

#### Scenario: A shared rule from personal instances
- **WHEN** `claim assert --evidential inferred --scope organisation --basis clm_p` is run and `clm_p` is `personal`
- **THEN** the claim is written and the result carries `scope_wider_than_inputs` naming `clm_p`

#### Scenario: Promoting a basis member clears the condition
- **WHEN** an `organisation` inferred claim is warned for its `personal` basis member, and `publish <member> --scope organisation` is run
- **THEN** the promotion's result carries no `scope_wider_than_inputs` finding for the inferred claim, though the promotion did not name it, and `validate` no longer reports it
