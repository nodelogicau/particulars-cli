## MODIFIED Requirements

### Requirement: Instructions and prompt carry the skill
The `initialize` response SHALL include `instructions` consisting of a short header naming the bound workspace (root and id) and stating that writes land as files for human review, followed by the embedded skill's body. When the workspace carries a conventions document (per the `workspace` capability), the instructions SHALL additionally carry its content after the skill body, under a heading naming the file. When the server limits what it delivers it SHALL deliver at least the first 16 KiB of the document's UTF-8 encoding, SHALL cut only on a character boundary — advancing past a character that straddles the limit, never cutting short of it — and SHALL append a note saying the text was truncated and naming the file. A configured conventions file that cannot be read SHALL be reported on stderr and omitted, never failing startup. The server SHALL also expose a prompt named `particulars-discipline` returning the same text.

#### Scenario: Instructions present
- **WHEN** a client initialises
- **THEN** `instructions` contains the workspace root and the phrase "Recall **before** you assert"

#### Scenario: Workspace conventions delivered
- **WHEN** the workspace root holds `dkf.md` and a client initialises
- **THEN** `instructions` contains a heading naming `dkf.md` followed by the file's content, after the skill body

#### Scenario: No conventions, no section
- **WHEN** the workspace has no conventions document
- **THEN** `instructions` contains no conventions heading

#### Scenario: Truncation lands on a character boundary and honours the floor
- **WHEN** the document is longer than 16 KiB and a multi-byte character begins before and ends after the 16 KiB mark
- **THEN** the delivered text is valid UTF-8, contains that whole character, is at least 16 KiB long, and is followed by a note naming the file

#### Scenario: Prompt available
- **WHEN** a client lists prompts and gets `particulars-discipline`
- **THEN** the prompt's message text equals the instructions body

## ADDED Requirements

### Requirement: Conventions document is a resource
When the workspace carries a readable conventions document at startup, the server SHALL expose it as an MCP resource with a `file://` URI naming the document's absolute path, `name` equal to its workspace-relative path, `mimeType` `text/markdown`, whose content is read from the file at request time. When no document applies, no resource SHALL be listed.

#### Scenario: Resource listed and readable
- **WHEN** the root holds `dkf.md` and a client lists resources and reads the one named `dkf.md`
- **THEN** the listing contains exactly one resource, and its content equals the file's current bytes

#### Scenario: No document, no resource
- **WHEN** the workspace has no conventions document
- **THEN** the resource listing is empty
