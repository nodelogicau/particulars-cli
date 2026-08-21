## ADDED Requirements

### Requirement: Merge record format
A merge record SHALL be stored at `merges/mrg_<uuidv7>.yaml` with keys in the order `id`, `type: merge`, `uris` (exactly two URIs, sorted lexically), `reason` (optional), `source` (at least one of `author`/`harness`), `timestamp` (RFC 3339 UTC), and `retracted` (optional, as for claims). Merge records SHALL be encoded deterministically and SHALL never be rewritten except by appending a `retracted` block.

#### Scenario: Merge file written
- **WHEN** a merge between `https://example.com/particulars/project-x` and `urn:dkf:W:projectx` is created with reason "Same project"
- **THEN** `merges/mrg_….yaml` exists with `type: merge`, `uris` listing both URIs in lexical order, `reason: Same project`, a `source`, and a `timestamp`, and no other file except `index.yaml` changes

#### Scenario: Stable bytes
- **WHEN** a merge record is serialised, parsed, and serialised again
- **THEN** both serialisations are byte-identical

### Requirement: Merge verb
`particulars particular merge <a> <b> [--reason <text>] [--author] [--harness] [--model]` SHALL resolve each argument to a URI — via a local particular's id, uri, label, or alias, or, when nothing local matches and the argument is syntactically a URI, the argument itself — and SHALL write a merge record and update `index.yaml`. It SHALL exit 2 if both arguments resolve to the same URI or if a non-retracted merge already joins them, exit 2 if an argument is ambiguous, and exit 3 if an argument matches nothing locally and is not a URI. The result SHALL include the merge record and, for each side, the local particular id if one exists.

#### Scenario: Merge two local particulars
- **WHEN** `particular merge "Project X" ProjectX-legacy` is run and both resolve to distinct local particulars
- **THEN** a merge record with their two URIs is written and the result names both particular ids

#### Scenario: Merge with a foreign URI
- **WHEN** `particular merge "Project X" https://www.wikidata.org/entity/Q1` is run and the URI matches no local particular
- **THEN** a merge record is written with the foreign URI and the result marks that side as having no local particular

#### Scenario: Duplicate merge refused
- **WHEN** a non-retracted merge already joins the two URIs and the same merge is requested
- **THEN** the command exits with code 2 and writes nothing

#### Scenario: Self-merge refused
- **WHEN** both arguments resolve to the same URI
- **THEN** the command exits with code 2

### Requirement: Equivalence classes
Non-retracted merge records SHALL be treated as symmetric and transitive edges between URIs. The set of local particulars reachable from a particular through such edges (including through URIs that have no local particular) SHALL be its equivalence class. A retracted merge SHALL contribute no edge.

#### Scenario: Transitive class through a foreign URI
- **WHEN** merges join (A, U) and (U, B) where U has no local particular
- **THEN** A and B are in the same equivalence class

#### Scenario: Retracted merge removes the edge
- **WHEN** the only merge joining A and B is retracted
- **THEN** A and B are in different classes and the merge file remains on disk with its `retracted` block

### Requirement: Merges are retractable
`particulars retract mrg_… --reason <text>` SHALL append a `retracted` block to the merge file exactly as for claims, and SHALL reject `--superseded-by` for merge targets with exit 2. The index entry SHALL gain `retracted: true`.

#### Scenario: Retract a merge
- **WHEN** `retract mrg_M --reason "Different projects after all"` is run
- **THEN** `merges/mrg_M.yaml` gains a `retracted` block and its index entry has `retracted: true`

#### Scenario: Superseded-by rejected for merges
- **WHEN** `retract mrg_M --reason r --superseded-by clm_X` is run
- **THEN** the command exits with code 2 and writes nothing

### Requirement: Merge validation
`validate` SHALL report an error when a merge record has other than exactly two `uris`, when the two URIs are equal, when its `source` lacks both `author` and `harness`, when its `timestamp` is missing or invalid, or when its id, type, file name, or directory disagree. It SHALL report a warning `unknown_merge_uri` when a URI in a non-retracted merge matches no local particular, and a warning `duplicate_merge` when two non-retracted merges join the same pair.

#### Scenario: Three URIs
- **WHEN** a hand-edited merge record lists three URIs
- **THEN** `validate` reports an error with code `invalid_merge`

#### Scenario: Foreign URI warning
- **WHEN** a merge joins a local particular with `https://www.wikidata.org/entity/Q1` and no local particular has that URI
- **THEN** `validate` reports a warning with code `unknown_merge_uri` and exits 0 if there are no errors
