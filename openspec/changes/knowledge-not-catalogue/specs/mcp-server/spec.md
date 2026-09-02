## ADDED Requirements

### Requirement: The tool surface teaches the register
The descriptions and parameter schemas of `claim_assert` and `particular_define` SHALL state the register that separates knowledge from catalogue: the subject is the thing in the world the fact is about, never the document or feed it was read in; content states the fact, and what was read goes in `source.document`. `particular_define`'s description SHALL give identity examples for global URIs (a person, a project) and SHALL NOT give reading matter as an example. The requirement pins presence of the register, not exact wording.

#### Scenario: Assert-time register present
- **WHEN** a client lists tools
- **THEN** `claim_assert`'s description or parameter schemas state that the subject is the thing in the world and that what was read belongs in `source.document`

#### Scenario: Define does not invite document particulars
- **WHEN** a client reads `particular_define`'s description
- **THEN** its URI examples name identities, not articles or feeds, and it states that a particular is a thing in the world, not a document being read
