## MODIFIED Requirements

### Requirement: Instructions and prompt carry the skill
The `initialize` response SHALL include `instructions` consisting of a short header naming the bound workspace (root and id) and stating that writes land as files for human review, followed by the embedded skill's body. When the workspace carries a conventions document (per the `workspace` capability), the instructions SHALL additionally carry its content after the skill body, under a heading naming the file, truncated at 16 KiB with a note naming the file when longer. A configured conventions file that cannot be read SHALL be reported on stderr and omitted, never failing startup. The server SHALL also expose a prompt named `particulars-discipline` returning the same text.

#### Scenario: Instructions present
- **WHEN** a client initialises
- **THEN** `instructions` contains the workspace root and the phrase "Recall **before** you assert"

#### Scenario: Workspace conventions delivered
- **WHEN** the workspace root holds `CONVENTIONS.md` and a client initialises
- **THEN** `instructions` contains a heading naming `CONVENTIONS.md` followed by the file's content, after the skill body

#### Scenario: No conventions, no section
- **WHEN** the workspace has no conventions document
- **THEN** `instructions` contains no conventions heading

#### Scenario: Prompt available
- **WHEN** a client lists prompts and gets `particulars-discipline`
- **THEN** the prompt's message text equals the instructions body
