## ADDED Requirements

### Requirement: A dkf/0.2 index is not yet read or written
Until the CLI implements the `dkf/0.2` index (root and sealed segments), `particulars index` and `particulars index --check` in a `dkf/0.2` workspace SHALL refuse without reading or writing any index document, with the message that the release reads `dkf/0.2` but does not write it yet, naming particulars-cli#11, and exit with code 1. Their behaviour in a `dkf/0.1` workspace SHALL be unchanged.

#### Scenario: Checking a sharded index
- **WHEN** `index --check` is run in a `dkf/0.2` workspace
- **THEN** the command exits 1 with the read-only message and no file changes

#### Scenario: Rebuilding a sharded index
- **WHEN** `index` is run in a `dkf/0.2` workspace
- **THEN** the command exits 1 with the read-only message and no file changes
