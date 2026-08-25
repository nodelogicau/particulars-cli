## ADDED Requirements

### Requirement: Promoting a quoted claim reports the disclosure
Promotion SHALL report, without refusing, when any object it covers carries a document `quote`. A quote reproduces its source verbatim, so widening a quoted claim's scope publishes that source text in full — unlike a synthesis, which summarises what it cites. The report SHALL name the objects concerned.

#### Scenario: Quoted claim promoted
- **WHEN** a claim carrying a quote is promoted to `public`
- **THEN** the result names it and states that verbatim source text is now published, and the promotion succeeds

#### Scenario: Unquoted promotion is silent
- **WHEN** no promoted object carries a quote
- **THEN** no disclosure is reported
