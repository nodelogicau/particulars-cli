## ADDED Requirements

### Requirement: An author value is a particular reference
Wherever the CLI reads a `source.author` or `source.document.author`, the value SHALL be interpreted as one of: a particular id (`par_` prefix), a particular URI (a scheme-prefixed value), or a bare name. Readers SHALL resolve an id by exact match; a URI by a particular's `uri`, including a URI joined to it by non-retracted merge records; and a bare name by label or alias, case-insensitively. A value resolving to no particular, or a name matching more than one, SHALL be treated as an opaque value that still satisfies the source minimum — unresolved, never invalid.

#### Scenario: Retroactive attribution
- **WHEN** a workspace holds claims carrying `author: ben` and a particular with alias `ben` is then defined
- **THEN** those claims are asserted by that particular without any file changing

#### Scenario: URI through a merge
- **WHEN** a claim carries `author: urn:dkf:W:jane`, that URN is merged with a particular's ORCID `uri`, and objects reported from the ORCID are queried
- **THEN** the claim is included

### Requirement: Writers resolve the author and refuse only what cannot be meant
When writing any object or record carrying a `source`, the CLI SHALL resolve `author` (and `document.author`) as follows: an id naming a particular is written as that particular's `uri`; an id naming no particular SHALL be refused with exit code 3; a URI is written unchanged whether or not a particular carries it; a bare name resolving to exactly one particular is written as that particular's `uri`; a bare name resolving to none is written unchanged; a bare name given explicitly (a per-verb flag or per-call MCP field) resolving to more than one SHALL be refused with exit code 2 listing the candidate ids; a bare name taken from a default (`dkf.yaml`, `DKF_AUTHOR`, or `serve --author`) that resolves to more than one SHALL be written unchanged, and the result SHALL carry a `warnings` entry naming the candidates (printed to stderr in text mode; `--json` stderr stays reserved for the error envelope). The CLI SHALL NOT define a particular for a person as a side effect of `init` or of any write.

#### Scenario: Default author resolves to a URI
- **WHEN** `dkf.yaml` has `defaults.source.author: ben`, exactly one particular has alias `ben`, and `claim assert` runs without `--author`
- **THEN** the written claim carries that particular's `uri` as `source.author`

#### Scenario: Explicit ambiguous name refused
- **WHEN** two particulars carry alias `ben` and `claim assert --author ben` runs
- **THEN** the command exits 2 listing both candidate ids and writes nothing

#### Scenario: Default ambiguous name falls through
- **WHEN** two particulars carry alias `ben`, `defaults.source.author` is `ben`, and `claim assert` runs without `--author`
- **THEN** the claim is written with `author: ben` unchanged and the result's `warnings` names both candidates

#### Scenario: Unknown id refused
- **WHEN** `claim assert --author par_01aa00000000000000000000000000` runs and no such particular exists
- **THEN** the command exits 3 and writes nothing

#### Scenario: Unknown URI written unchanged
- **WHEN** `claim assert --author https://orcid.org/0000-0002-1825-0097` runs and no particular carries that `uri`
- **THEN** the claim is written with that URI as `source.author`

### Requirement: Asserted-by and reported-from are distinct relations over merge classes
An object SHALL be asserted by the particular its `source.author` resolves to and reported from the particular its `source.document.author` resolves to, both computed over the merge equivalence class of the resolved particular, and the two relations SHALL never be collapsed when reported.

#### Scenario: Recorded testimony
- **WHEN** a claim's `source.author` resolves to Ben and its `source.document.author` resolves to Jane
- **THEN** the claim is asserted by Ben and reported from Jane, and neither relation is reported as the other
