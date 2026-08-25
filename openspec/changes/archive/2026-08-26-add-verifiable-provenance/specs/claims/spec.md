## MODIFIED Requirements

### Requirement: Assert a claim
`particulars claim assert --subject <particular> (--content <text> | --content-file <path|->) [--author] [--harness] [--model] [--document <uri>] [--document-hash <sha256:…> | --hash-document] [--quote <text> | --quote-file <path|->] [--scope] [--topic <t>]... [--confidence <0..1>] [--timestamp <rfc3339>]` SHALL write a new claim file with `type: claim`, `subject` set to the resolved particular id, `source` populated from flags/environment/defaults, `context.scope` defaulting to `dkf.yaml` `defaults.scope`, `timestamp` defaulting to the current UTC time, and SHALL update `index.yaml`. The resolved `source` SHALL contain at least one of `author` or `harness`; if neither is available the command SHALL exit 2 naming both `DKF_AUTHOR` and `DKF_HARNESS`. The result SHALL include the full claim.

`--document-hash`, `--hash-document`, `--quote`, and `--quote-file` SHALL require `--document` and SHALL be usage errors without it. `--hash-document` SHALL compute the hash from the local file the document resolves to, and SHALL be a usage error when it does not resolve. `--document-hash` and `--hash-document` SHALL be mutually exclusive, as SHALL `--quote` and `--quote-file`.

#### Scenario: Claim with evidence
- **WHEN** `claim assert --subject "Project X" --content "Uses Postgres 16" --document docs/db.md` is run
- **THEN** a claim file is written whose `source.document` is the scalar `docs/db.md`

#### Scenario: Verifiable claim
- **WHEN** the same command adds `--hash-document --quote "Postgres 16 in every environment"`
- **THEN** `source.document` is a mapping carrying `uri`, a `sha256:` hash computed from the local file, and the quote

#### Scenario: Locator without a document
- **WHEN** `--quote` is given without `--document`
- **THEN** the command exits 2
