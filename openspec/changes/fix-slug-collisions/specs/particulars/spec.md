## MODIFIED Requirements

### Requirement: URI minting
When `--uri` is omitted, the CLI SHALL mint `uri = <base> + <slug>` where `base` is `workspace.base-uri` from `dkf.yaml` if set, otherwise `urn:dkf:<workspace.id>:`, and `slug` is the label lowercased, NFKD-folded with combining marks stripped, with every run of characters outside `[a-z0-9]` replaced by a single hyphen and leading/trailing hyphens removed. A slug SHALL NOT by itself decide that two labels name the same particular; *A minted URI resolves only to the particular it was minted for* decides that. The MCP `particular_define` without `uri` SHALL mint the same way.

If the slug is empty, the CLI SHALL NOT mint a URI. It SHALL refuse the call with exit code 2 (code `usage`) without creating or changing any file. The refusal SHALL say that the label yields no slug and that an explicit `--uri` is required. This check SHALL come before any lookup of the minted URI.

#### Scenario: Minted under base URI
- **WHEN** `dkf.yaml` has `base-uri: https://example.com/particulars/` and `particular define --label "Billing Service"` is run
- **THEN** the particular's `uri` is `https://example.com/particulars/billing-service`

#### Scenario: Minted under URN fallback
- **WHEN** `dkf.yaml` has no `base-uri`, `workspace.id` is `W`, and `particular define --label "Auth & Sessions"` is run
- **THEN** the particular's `uri` is `urn:dkf:W:auth-sessions`

#### Scenario: Empty slug rejected
- **WHEN** `particular define --label "!!!"` is run without `--uri`
- **THEN** the command exits with code 2

#### Scenario: A symbol label
- **WHEN** `particular define --label "#"` is run without `--uri`
- **THEN** the command exits with code 2, the message says an explicit `--uri` is required, and no file is written

#### Scenario: A label in a script the slug drops
- **WHEN** `particular define --label "東京"` is run without `--uri`, or `particular_define{label: "東京"}` is called over MCP
- **THEN** the call is refused with code `usage` and no file is written

#### Scenario: An explicit URI for such a label
- **WHEN** `particular define --label "東京" --uri https://www.wikidata.org/entity/Q1490` is run
- **THEN** the particular is created with that URI and `created: true`

#### Scenario: A label with some characters the slug keeps
- **WHEN** `dkf.yaml` has no `base-uri`, `workspace.id` is `W`, and `particular define --label "C 東京"` is run
- **THEN** the minted URI is `urn:dkf:W:c`, and the call is subject to *A minted URI resolves only to the particular it was minted for*

### Requirement: Define is idempotent on URI
When `--uri` is given (or `uri` over MCP) and a particular with that URI already exists, `define` SHALL NOT create a new one. It SHALL update the existing particular: `label` is replaced by the supplied label, and `aliases` becomes the union of the existing aliases, the supplied aliases, and the previous label if it differs from the new one. The result SHALL have `created: false` and the existing `id`. A define without `--uri` SHALL instead follow *A minted URI resolves only to the particular it was minted for*.

#### Scenario: Same URI, second session
- **WHEN** `particular define --label "Project X" --uri https://example.com/x` is run twice in the same workspace
- **THEN** only one particular file exists and the second result has `created: false` with the same id as the first

#### Scenario: Label change preserved as alias
- **WHEN** a particular has `label: ProjectX` and `define --uri <same-uri> --label "Project X"` is run
- **THEN** the particular's `label` is `Project X` and `aliases` contains `ProjectX`

#### Scenario: Renaming through an explicit URI
- **WHEN** a particular with label `C` and minted URI `urn:dkf:W:c` exists, and `particular define --uri urn:dkf:W:c --label "C (language)"` is run
- **THEN** the result has `created: false`, the particular's `label` is `C (language)`, and `aliases` contains `C`

## ADDED Requirements

### Requirement: A minted URI resolves only to the particular it was minted for
When `particular define` runs without `--uri`, or `particular_define` is called without `uri`, the existing particulars for the minted URI SHALL be those in its merge class: the particular carrying that URI, plus every particular joined to it by non-retracted merge records. If there are any, the call SHALL resolve to one of them only if the supplied label equals, under Unicode case folding, the `label` or one of the `aliases` of a particular in that class. The supplied aliases SHALL NOT count towards the match. When more than one particular in the class matches, the call SHALL resolve to the one carrying the minted URI if it matches, otherwise to the matching particular with the lowest id.

If no particular in the class matches, the call SHALL be refused with code `conflict` and exit code 1, without creating or changing any file. The refusal SHALL name an existing particular's `id`, `uri` and `label`: the one carrying the minted URI if there is one, otherwise the class member with the lowest id. It SHALL say that a different thing needs an explicit `--uri`, and that the same thing can be referred to by its URI or id. The MCP refusal SHALL be an error result with the same code and message.

When the call resolves to an existing particular, `define` SHALL NOT change its `label` or `uri`. It SHALL add each supplied alias that the particular doesn't already carry under case folding, and isn't equal to its label under case folding. It SHALL NOT remove any alias. The result SHALL have `created: false` and that particular's `id`. If no alias is added, no file SHALL be written. When the minted URI names no particular, `define` SHALL create one as *Define a particular* specifies.

#### Scenario: Same label, second session
- **WHEN** `particular define --label "Project X"` is run twice in the same workspace
- **THEN** only one particular file exists and the second result has `created: false` with the same id as the first

#### Scenario: Case-only difference keeps the label
- **WHEN** a particular with label `Project X` was minted, and `particular define --label "project x"` is run without `--uri`
- **THEN** the result has `created: false` with the same id, and the particular's `label` remains `Project X`

#### Scenario: Same slug, different thing
- **WHEN** a particular with label `C` and minted URI `urn:dkf:W:c` exists, and `particular define --label "C#"` is run without `--uri`
- **THEN** the command exits with code 1, the JSON error has code `conflict`, the message names that particular's id, `urn:dkf:W:c` and `C` and says to pass `--uri` for a different thing, and no file is written

#### Scenario: A label differing only in diacritics
- **WHEN** a particular with label `Café` exists, and `particular define --label "Cafe"` is run without `--uri`
- **THEN** the call is refused with code `conflict`, naming the particular labelled `Café`

#### Scenario: Matching an existing alias
- **WHEN** a particular with label `C`, alias `C-lang` and URI `urn:dkf:W:c-lang` exists (minted as `C-lang`, then renamed through its URI), and `particular define --label "c-lang"` is run without `--uri`
- **THEN** the result has `created: false` with that particular's id, and its `label` remains `C`

#### Scenario: Supplied aliases do not steer the match
- **WHEN** a particular with label `C` exists at `urn:dkf:W:c`, and `particular define --label "C#" --alias C` is run without `--uri`
- **THEN** the call is refused with code `conflict` and no file is written

#### Scenario: Aliases are added, never removed
- **WHEN** a particular with label `Project X` and alias `ProjectX` exists, and `particular define --label "project x" --alias project_x` is run without `--uri`
- **THEN** the particular's `label` remains `Project X` and its `aliases` are `ProjectX` and `project_x`

#### Scenario: A repeat define writes nothing
- **WHEN** a particular with label `Project X` exists, and `particular define --label "Project X"` is run again without `--uri`
- **THEN** the result has `created: false` and the particular file is byte-identical to before

#### Scenario: Explicit URI beside a minted one
- **WHEN** a particular with label `C` and minted URI `urn:dkf:W:c` exists, and `particular define --label "C#" --uri https://www.wikidata.org/entity/Q2370` is run
- **THEN** a second particular is created with `created: true`, and the particular labelled `C` is unchanged

#### Scenario: Minted URI reached through a merge
- **WHEN** no particular carries `urn:dkf:W:ada-lovelace`, a merge record joins it to `https://orcid.org/0000-0001` whose particular is labelled `Ada Lovelace`, and `particular define --label "Ada Lovelace"` is run without `--uri`
- **THEN** the result has `created: false` with the ORCID particular's id, and no particular file carrying `urn:dkf:W:ada-lovelace` is written

#### Scenario: A merged class that doesn't match
- **WHEN** no particular carries `urn:dkf:W:ada`, a merge record joins it to a particular labelled `Ada Lovelace` with no aliases, and `particular define --label "ADA"` is run without `--uri`
- **THEN** the call is refused with code `conflict`, naming the particular labelled `Ada Lovelace`

#### Scenario: Same rule over MCP
- **WHEN** a particular with label `C` exists at `urn:dkf:W:c`, and `particular_define{label: "C#"}` is called
- **THEN** the result is an error with code `conflict` naming that particular, and no file is written
