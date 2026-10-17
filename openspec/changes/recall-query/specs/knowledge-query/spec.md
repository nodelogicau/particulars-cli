## ADDED Requirements

### Requirement: Recall by query
`particulars recall --query <text> [--author <who>] [--topic <t>]... [--scope <s>] [--include-retracted] [--limit <n>]` SHALL implement DKF `recall-query`'s lexical baseline:
- **Particulars resolved.** It SHALL resolve the particulars whose label, an alias, or the `content` of a claim or synthesis about them matches the query, ignoring every filter.
- **Response per particular.** It SHALL then return, for each resolved merge class, exactly what `recall <lowest member id>` would return with the same `--author`, `--topic`, `--scope`, and `--include-retracted`. A class with nothing left after filtering SHALL be omitted.
- **Terms.** A term SHALL be a maximal run of characters without the Unicode `White_Space` property. Leading and trailing characters of Unicode general category P SHALL be removed from the whole query and from each term, as written and before case folding, and interior characters SHALL be kept. A term empty after removal SHALL be ignored.
- **Matching.** Matching SHALL be under full Unicode case folding. A text SHALL match when the trimmed whole query or any one trimmed term is a substring of it.
- **Empty query.** A query empty after removal SHALL produce an empty result and exit 0.

`--query` SHALL be a usage error together with a positional particular.

The JSON result SHALL carry:
- `query`;
- `particulars`, a list of `{particular, label, resolution, entries}`, plus `class` when the class has more than one member, where `particular` is the lowest id in the class;
- `count`, the number of particulars returned;
- `omitted`, the number of non-empty resolved classes not returned because of the limit;
- `author` when `--author` is given.

Entries SHALL be as in recall by particular. An entry whose `content` matches SHALL carry `matched: true`.

`resolution` SHALL be:
- `label` when a member's label, case-folded, occurs in the case-folded query with every query character outside that occurrence of category P;
- otherwise `alias`, on the same test against aliases;
- otherwise `lexical`.

Particulars SHALL be ordered by `resolution` in that order. Within each value they SHALL be ordered by:
1. whether the trimmed whole query matches a member's label, an alias, or an entry's `content`, matches first;
2. the number of distinct case-folded trimmed terms that match any of those, more first;
3. ascending `particular`.

`--limit` SHALL count particulars, keeping the first in this order with their entries whole. It SHALL default to 10 when omitted or `0`, and SHALL be a usage error when negative.

#### Scenario: A query matches a superseded claim
- **WHEN** `clm_1` about Project T contains "auth timeout", `syn_1` about Project T synthesises `clm_1` without the phrase, and `recall --query "auth timeout" --json` is run
- **THEN** Project T's entry lists `clm_1` with `matched: true` and `syn_1` with `current: true`

#### Scenario: A question mark does not hide a term
- **WHEN** a claim reads "The session timeout is 30 minutes." and `recall --query "timeout?"` is run
- **THEN** its subject is returned with `resolution: lexical` and the claim is `matched`

#### Scenario: An identifier at the end of a sentence
- **WHEN** a claim contains `PROJ-1234` and `recall --query "the ticket was PROJ-1234."` is run
- **THEN** the claim's subject is returned

#### Scenario: Interior punctuation is kept
- **WHEN** a claim's only matching text is "v1.2" and `recall --query "v1.2.3?"` is run
- **THEN** its subject is not returned

#### Scenario: Full case folding
- **WHEN** a claim contains "Straße" and `recall --query STRASSE` is run
- **THEN** the claim's subject is returned

#### Scenario: A name asked about is exact
- **WHEN** particulars are labelled `Project X` and `C#`, and the queries are `Project X?`, `"Project X"`, and `C#?`
- **THEN** each named particular has `resolution: label`

#### Scenario: A prefix is not exact
- **WHEN** a particular is labelled `C#`, a claim about it contains "C", and the query is `C`
- **THEN** its `resolution` is `lexical`

#### Scenario: Exact alias
- **WHEN** a particular has alias `DKF` and the query is `dkf`
- **THEN** its `resolution` is `alias`

#### Scenario: Punctuation alone
- **WHEN** the query is `?!`, even with a particular labelled `?`
- **THEN** `particulars` is empty and the command exits 0

#### Scenario: Retracted match
- **WHEN** the only match for the query is a retracted claim about Project A, which has other claims, and `--include-retracted` is not given
- **THEN** Project A is returned, but the retracted claim is not among its entries and does not count toward its terms

#### Scenario: Nothing in scope
- **WHEN** the query resolves Project P, all of whose objects are `personal`, and `--scope public` is given
- **THEN** Project P is not returned and is not counted in `omitted`

#### Scenario: A merge class appears once
- **WHEN** Project A and Project B are merged and the query matches a claim about each
- **THEN** one particular is returned, under the lower id, with `class` listing both and entries about both

#### Scenario: More terms matched ranks first
- **WHEN** the query is `auth timeout`, Project A's entries contain "auth" and "timeout" in separate claims, Project B's contain only "auth", and `--limit 1` is given
- **THEN** Project A is returned and `omitted` is 1

#### Scenario: The whole query ranks first
- **WHEN** the query is `auth timeout`, Project A has a claim containing "auth timeout", and Project B has "auth" and "timeout" in separate claims
- **THEN** Project A is ordered before Project B

#### Scenario: Exact before lexical under a limit
- **WHEN** one particular is labelled `Billing` and has a claim, twelve others have claims containing "billing", and `recall --query billing` is run
- **THEN** ten particulars are returned, the first with `resolution: label`, and `omitted` is 3

#### Scenario: Query and particular together
- **WHEN** `recall "Project X" --query timeout` is run
- **THEN** the command exits with the usage code

#### Scenario: Filters combine with a query
- **WHEN** `recall --query timeout --topic auth` is run
- **THEN** every returned entry carries topic `auth`
