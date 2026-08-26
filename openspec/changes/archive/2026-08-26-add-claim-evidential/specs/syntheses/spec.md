## MODIFIED Requirements

### Requirement: Create a synthesis
`particulars synthesis create --subject <particular> (--content <text> | --content-file <path|->) --input <id>:<role>[:<weight>]... --unresolved <text> [--method reconciliation|qualification|positions] [--author] [--harness] [--model] [--document <uri>] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new synthesis file with `type: synthesis`, `inputs` as a list of `{id, role, weight}`, `unresolved`, `source: {author?, harness, model?, document?}`, `method` (one of the three values; default `reconciliation`; any other value SHALL be a usage error), and context/timestamp/confidence as for claims, and SHALL update `index.yaml`. The file SHALL NOT contain a `produced-by` key and SHALL NOT contain an `evidential` key: a synthesis is backed by argument from its inputs, so the value is implied and cannot vary.

#### Scenario: Minimal synthesis
- **WHEN** `synthesis create` is run with a subject, content, two inputs, and an unresolved statement
- **THEN** a new `syn_` file exists with two inputs whose weights default to `primary`, `source.harness: claude`, `source.model: claude-sonnet-4-6`, `method: reconciliation`, no `produced-by` key, and an index entry listing both input ids

#### Scenario: Method vocabulary enforced
- **WHEN** `--method consensus` is given
- **THEN** the command exits 2 naming the three values
