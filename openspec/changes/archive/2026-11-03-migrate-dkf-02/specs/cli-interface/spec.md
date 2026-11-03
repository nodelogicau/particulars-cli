## MODIFIED Requirements

### Requirement: Version verb
The CLI SHALL provide `particulars version` reporting the binary version, the DKF formats it reads, and the DKF format it writes into a new workspace.

#### Scenario: Version output
- **WHEN** `particulars version --json` is run
- **THEN** stdout contains an object with `version`, `format: "dkf/0.2"` (the format written), and `reads: ["dkf/0.1", "dkf/0.2"]`
