## MODIFIED Requirements

### Requirement: Scope filters but does not silently withhold
The visual formats SHALL include `personal` knowledge by default, because emitting a local file is not a transfer to a third party and a diagram that omitted part of the graph would misrepresent it. `--scope <s>` SHALL restrict the export to objects whose **effective** scope is at least `s`, and unlike `--format graph` the value `personal` SHALL be accepted. Documentation SHALL state that a diagram discloses whatever it contains, so publishing one — in a pull request comment, for instance — carries the same judgement as publishing the objects themselves.

#### Scenario: Personal knowledge is drawn
- **WHEN** a particular's claims are all `personal` and `export --format mermaid --subject <it>` is run
- **THEN** those claims appear as nodes

#### Scenario: Scope narrows the diagram
- **WHEN** `--scope organisation` is given
- **THEN** only assertions whose effective scope is `organisation` or wider appear

#### Scenario: A promoted claim survives a narrowed diagram
- **WHEN** a `personal` claim has been promoted to `organisation` and `--scope organisation` is given
- **THEN** that claim appears
