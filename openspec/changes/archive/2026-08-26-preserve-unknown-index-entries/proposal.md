## Why

`index.yaml` is the one place the format's compatibility promise breaks. Object files degrade gracefully for an older reader — a consumer that ignores `/publishes/` still reads every claim — but an older binary that *rebuilds the index* reproduces only the entry types it knows, silently stripping the rest, and its drift check reports a newer writer's entries as corruption. Both halves have been observed: the knowledge repo's CI failed with `index_stale: extra 5` when a v0.6.0 binary met promotion entries, and the same workspace was nearly written with a stale `dist/particulars` that would have dropped every `publishes/` row **and then reported the workspace clean** — data loss in the cache, surfacing only when a newer tool next computed effective scope.

Raised as [particulars#16](https://github.com/nodelogicau/particulars/issues/16); the spec resolved it at [`db748da`](https://github.com/nodelogicau/particulars/commit/db748da): a rebuild MUST preserve entries whose `type` it does not recognise, a drift check MUST NOT report them as differences, and consumers MUST ignore entries they do not understand. This CLI — the implementation that filed the issue — does none of the three, and the current `index` capability spec explicitly requires the wrong behaviour ("regenerate … ignoring any existing index content").

## What Changes

- **Reading the index preserves unknown entry types** as raw nodes, kept apart from the typed entries so every existing consumer of `idx.Entries` ignores them automatically — the MUST-ignore falls out of the structure.
- **Rebuild and every incremental update carry them through**, unchanged, merged into the id order the file already uses. This implementation stops being the older tool that destroys the cache the day a sixth record type exists.
- **`index --check` treats them as background, not drift**: the comparison sees exactly what a rebuild would write, which now includes them, so evidence of a newer conforming writer no longer fails CI.
- `validate`'s `index_stale` inherits the same behaviour, since it is the same check.

## Capabilities

### Modified Capabilities
- `index`: `Rebuild` loses "ignoring any existing index content" in favour of preservation; `Check` stops counting unknown types as drift; a new requirement states preservation and the ignore rule.

## Impact

- **Modified**: `internal/store/index.go` (decode split, encode merge, check exclusion). Nothing outside the store changes: consumers already iterate `idx.Entries`, which continues to hold only known types.
- **Compatibility**: an index with no unknown entries round-trips byte-identically; this change is invisible until a newer writer exists.
- **Ships with**: `add-claim-evidential`, as v0.9.0.
