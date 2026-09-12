# Workspace Register

## Purpose

The generated list of what a workspace already uses — its particulars and its topic vocabulary, with how heavily each is used — delivered where a subject or a tag is chosen, so that a name is reused rather than re-minted. Derived from the object files, never authored, because a hand-typed registry goes stale and a topic can never be renamed.

## Requirements

### Requirement: The register reports the names a workspace already uses
`particulars register [--json]` SHALL report, from the object files alone, what the workspace already uses: every particular with its `label`, its `aliases`, whether its `uri` is global or workspace-minted, the number of live claims and syntheses whose `subject` it is, and the number of those not yet reconciled into a current synthesis; then every topic in use with the number of assertions carrying it. The register SHALL be derived, never authored: it SHALL NOT read the conventions document and SHALL NOT be affected by one. It SHALL require no network and SHALL be computable from the index where the index is current.

#### Scenario: Reporting both registries
- **WHEN** `particulars register` runs in a workspace with 39 particulars and 195 topics
- **THEN** it lists every particular with its counts and every topic with its count

#### Scenario: A particular with no belief
- **WHEN** a particular carries 8 claims and no synthesis
- **THEN** its row reports 8 claims, no current synthesis, and 8 unsynthesised

#### Scenario: Global against minted identity
- **WHEN** one particular carries `uri: https://www.wikidata.org/entity/Q42` and another a `urn:dkf:` URI minted from its label
- **THEN** the register distinguishes them, so a reader can see which things have been given a global identity

#### Scenario: Derived, not authored
- **WHEN** the workspace carries a conventions document naming particulars that no longer exist
- **THEN** the register reports what the files hold, and the conventions document does not change it

### Requirement: The register is deterministically ordered and bounded
The register SHALL order particulars by claim count descending, then label ascending, and topics by assertion count descending, then topic ascending, so that two runs over the same files agree. Rendered into text, it SHALL be bounded by a size cap; where the content exceeds it, entries SHALL be dropped from the tail of each ordering — the least-used first — and the register SHALL state how many were omitted. Truncation SHALL NOT drop an entry while retaining one used less often.

#### Scenario: Two runs agree
- **WHEN** the register is generated twice from unchanged files
- **THEN** the two renderings are byte-identical

#### Scenario: Truncating a large workspace
- **WHEN** the rendered register would exceed the cap
- **THEN** the most-used entries are kept, the least-used are dropped, and a line names how many topics and particulars were omitted

#### Scenario: Ties are stable
- **WHEN** two topics carry the same count
- **THEN** they are ordered by topic ascending, in both renderings and in `--json`

### Requirement: Singletons are marked
A topic carried by exactly one assertion SHALL be marked as such in the register. The marking SHALL be a fact, not a warning: a new topic is a singleton until it is reused, and the specification SHALL NOT treat one as an error, since a tag that has just been coined is indistinguishable from one that failed to catch on.

#### Scenario: A newly coined topic
- **WHEN** a topic has been used once
- **THEN** the register marks it a singleton and nothing reports it as a problem

#### Scenario: The proportion is visible
- **WHEN** 110 of 195 topics are carried by one assertion each
- **THEN** the register makes that visible without the reader counting rows
