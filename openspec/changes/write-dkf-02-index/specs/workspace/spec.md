## ADDED Requirements

### Requirement: Sealed months gain nothing new
In a `dkf/0.2` workspace, a month SHALL be sealed once the workspace holds an id minted two or more months after it, the current month being that of the newest id found in the month directories. Creating an object whose new UUIDv7 id falls in a sealed month SHALL be refused, and writing a retraction whose `timestamp` falls in a sealed month SHALL be refused, each before any file is written, with code `sealed_month`, a message naming the sealed month and the workspace's current month and saying to check the machine's clock, and exit code 1. The CLI and MCP server SHALL mint ids and date retractions at the machine's clock, so that only a clock more than a month behind the workspace can reach either refusal.

#### Scenario: A clock months behind
- **WHEN** the newest id in a `dkf/0.2` workspace is from December 2026 and `claim assert` runs on a machine whose clock reads September 2026
- **THEN** the command exits 1 with `sealed_month`, naming September and December, and writes nothing

#### Scenario: A clock days behind across a boundary
- **WHEN** the newest id is from 2 December 2026 and a machine two days slow mints on what it reads as 30 November
- **THEN** the claim is written under `claims/2026-11/`, which is open, and nothing is refused

#### Scenario: Retracting with a clock months behind
- **WHEN** the newest id is from December 2026 and `retract` runs on a machine whose clock reads September 2026
- **THEN** the command exits 1 with `sealed_month` and the file is unchanged

## REMOVED Requirements

### Requirement: A dkf/0.2 workspace is read-only in this release
**Reason**: The CLI now writes the `dkf/0.2` index, so every write verb works in a `dkf/0.2` workspace.
**Migration**: None; the refusal was never released.
