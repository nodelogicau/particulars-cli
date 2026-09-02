## ADDED Requirements

### Requirement: URL-in-content note
`validate` SHALL report, at informational severity, `url_in_content` for each claim whose `content` contains a scheme-prefixed URL (`http://` or `https://`) — the catalogue smell: content that names its reading matter rather than stating a fact, when the reference belongs in `source.document`. The condition SHALL be reported as a corpus fact — one aggregate line carrying the count in text output, every finding individually in `--json`, listable with `--notes` — SHALL never be an error, SHALL not change the exit code, and SHALL NOT be reported for syntheses.

#### Scenario: Catalogue-shaped claims aggregate
- **WHEN** forty claims carry an article URL in their content
- **THEN** text output shows one `url_in_content` line with the count, `--json` carries all forty, and the exit code is unaffected

#### Scenario: A legitimate endpoint claim is a note, not a problem
- **WHEN** one claim's content is "The billing API base URL is https://api.example.com/v2"
- **THEN** `url_in_content` is reported for it at informational severity and validation exits 0

#### Scenario: Documents in their place produce no note
- **WHEN** a claim states a fact in content and cites the article in `source.document`
- **THEN** no `url_in_content` is reported
