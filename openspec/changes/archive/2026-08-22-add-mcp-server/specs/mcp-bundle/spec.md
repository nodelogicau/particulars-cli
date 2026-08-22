## ADDED Requirements

### Requirement: Desktop extension bundle
Each release SHALL attach `particulars-<version>.mcpb`, a zip containing `manifest.json` (`manifest_version` `0.3`, `name` `particulars`, `version` equal to the release), an icon, and the server binaries for darwin (arm64, amd64) and win32 (x64, arm64) under `server/`. The manifest's `server.type` SHALL be `binary`, `mcp_config.command` SHALL point at the bundled binary via `${__dirname}` with `platform_overrides` selecting the right one, and `args` SHALL be `serve --mcp --workspace ${user_config.workspace}` plus `--author ${user_config.author}` when set. `compatibility.platforms` SHALL list `darwin` and `win32`.

#### Scenario: Bundle contents
- **WHEN** `make bundle` runs after a cross-compile
- **THEN** the produced `.mcpb` contains `manifest.json` with the build's version and every binary the manifest references, and the manifest parses as JSON

#### Scenario: Release asset
- **WHEN** a tag is released
- **THEN** the release includes `particulars-<version>.mcpb` alongside the archives and `checksums.txt` lists it

### Requirement: User configuration
The manifest SHALL declare `user_config.workspace` (type `directory`, required, described as the folder containing `dkf.yaml`) and `user_config.author` (type `string`, optional). An empty `author` SHALL be treated by the server as unset.

#### Scenario: Installed in Claude Desktop
- **WHEN** the bundle is installed and the user picks a workspace folder containing `dkf.yaml`
- **THEN** Claude Desktop starts the server bound to that folder and its tools appear

#### Scenario: Folder without dkf.yaml
- **WHEN** the user picks a folder that is not a workspace
- **THEN** the server exits with code 5 and a message on stderr naming the folder and `particulars init`

### Requirement: Documented client configuration
`docs/mcp.md` SHALL show how to connect from Claude Desktop (bundle and manual `claude_desktop_config.json`), Claude Code (`.mcp.json` with the server spawned in the project so discovery finds the workspace), and Cursor, and SHALL list every tool with its parameters, marking the two extension tools.

#### Scenario: Claude Code config
- **WHEN** a user follows the `.mcp.json` example in a repository with a `.dkf` pointer
- **THEN** the server starts without `--workspace` and `workspace_status` reports the pointed-to workspace
