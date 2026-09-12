## ADDED Requirements

### Requirement: The skill directs a reader to the register before a subject is chosen
The embedded skill body SHALL direct its reader to read the workspace register once, before choosing a subject or a topic for a new assertion, and SHALL state that an MCP client receives it with the server's instructions and does not need to ask. The existing per-subject loop — resolve, recall, conflicts, unresolved — SHALL be unchanged, because it answers what is believed about a subject already chosen and cannot reveal what else the workspace uses.

#### Scenario: The register precedes the loop
- **WHEN** a reader follows the skill to record a fact in a workspace it has not written to this session
- **THEN** it reads the register before choosing the subject, and then follows the per-subject loop unchanged

#### Scenario: An MCP client does not call the verb
- **WHEN** a client has received the register in the server's instructions
- **THEN** the skill tells it no call is needed
