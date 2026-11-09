## Why

Upstream `recall-query-whitespace` (nodelogicau/particulars#37) settled how whitespace at the edges of a query is handled, and the CLI still follows the earlier rule. Today `Project X ?`, ` Project X`, or a query ending in a newline loses its exact `label` match, and a trailing space changes the whole-query ranking. The same change adds a first ordering key so that, for the query `C#`, the label `C#` ranks before `C` (#12).

## What Changes

- Characters with the Unicode `White_Space` property are trimmed from the edges of the whole query and of each term, alongside category P. Interior whitespace is kept.
- An exact match (`label` or `alias`) allows only category P or `White_Space` outside the name's occurrence in the query.
- Within `label` and `alias`, particulars are ordered first by the length of the longest exact label or alias, counted in code points after case folding, longer first. The existing keys follow it.
- `recall --help` says that spaces are trimmed and that longer exact names rank first.

## Capabilities

### Modified Capabilities

- `knowledge-query`: *Recall by query*. This changes the term and exact-match rules, adds the length key, and adds scenarios.

## Impact

- `internal/query/recall_query.go`: `isEdge` adds `unicode.IsSpace`, which is exactly `White_Space`. Stage one records the longest exact name per class, and the sort uses it.
- `internal/query/recall_query_test.go`: the cases that pinned the old behaviour are flipped, and the new upstream scenarios are added.
- `internal/cli/cmd_query.go`: help text.
- No change to the response shape, the MCP surface, or the skill. `mcp-server` defers to `knowledge-query` for the rules.
