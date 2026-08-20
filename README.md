# particulars

A command-line tool for dialectical knowledge management.

`particulars` reads and writes [Dialectical Knowledge Format (DKF)](https://github.com/nodelogicau/particulars)
workspaces: directories of plain YAML files that record **particulars** (the things
knowledge is about), **claims** (assertions with provenance), and **syntheses**
(resolutions of thesis and antithesis that carry the reasoning that produced them).

It is built to be driven by an LLM agent and reviewed by humans through git:

- every verb is non-interactive and supports `--json`
- exit codes are documented and stable
- files are written deterministically and only ever *created* — the one permitted
  mutation is an append-only retraction block — so pull-request diffs stay honest
- the binary is static, dependency-free, and does no reasoning of its own. The
  agent reasons; `particulars` stores, indexes, and reports structure.

This is a reference implementation of the DKF v0.1 *draft*. Field names and the
ID scheme may change before v0.1 is declared; see [SPEC-FEEDBACK.md](SPEC-FEEDBACK.md)
for what this implementation proposes upstream.

## Install

Download a binary for your platform from the
[releases page](https://github.com/nodelogicau/particulars-cli/releases), or build
from source (Go 1.26+):

```sh
git clone https://github.com/nodelogicau/particulars-cli
cd particulars-cli
make build          # → dist/particulars
```

## Quick start

```sh
# 1. Create a workspace (anywhere; commit it to git)
particulars init ./knowledge --author ben --harness claude \
    --base-uri https://example.com/particulars/
cd knowledge

# 2. Define the thing you are learning about
particulars particular define --label "Project X" --alias ProjectX
#   → par_0191…  uri: https://example.com/particulars/project-x

# 3. Record what you believe, with evidence
particulars claim assert --subject "Project X" \
    --content "Project X uses a microservices architecture." \
    --document docs/architecture.md --topic architecture --confidence 0.9

particulars claim assert --subject "Project X" \
    --content "Project X was consolidated into a monolith in November 2024." \
    --document adr/0042-monolith.md --topic architecture --confidence 0.8

# 4. See what has not been reconciled
particulars conflicts
#   par_0191…  Project X  priority=2
#     current:       (none)
#     unsynthesised: clm_0191…a, clm_0191…b

# 5. Reason (that's your job), then record the synthesis
particulars synthesis create --subject "Project X" \
    --input clm_0191…a:thesis --input clm_0191…b:antithesis \
    --unresolved "Compliance basis for the separate auth service is unsourced." \
    --content "Microservices 2022–2024, consolidated to a monolith in Nov 2024; \
auth remains separately deployable for compliance."

# 6. Recall current knowledge
particulars recall "Project X"
particulars recall --topic architecture --json
```

Everything lands as files:

```
knowledge/
  dkf.yaml                 workspace marker + defaults
  particulars/par_….yaml
  claims/clm_….yaml
  syntheses/syn_….yaml
  index.yaml               derived manifest (regenerate with `particulars index`)
```

## How review works

The agent works on a branch and opens a pull request. The diff *is* the proposal:
new files under `claims/` and `syntheses/` are new knowledge; a modified claim file
can only ever be a retraction. Merging is acceptance. If something merged turns out
to be wrong, retract it — never delete it:

```sh
particulars claim retract clm_0191…a --reason "Port is 8443, not 443" \
    --superseded-by clm_0191…c
```

Retraction appends a `retracted:` block to the claim file, recording who, when, and
why. `recall` hides retracted claims by default; `conflicts` flags syntheses that
cite them as **stale** so they can be re-synthesised.

See [docs/review-workflow.md](docs/review-workflow.md) for the full flow and a
GitHub Action that runs `validate` and `index --check` on every PR.

## Verbs

| Verb | Purpose |
|---|---|
| `init [dir]` | Create a workspace: `dkf.yaml`, type directories, empty index |
| `particular define --label L [--uri U] [--alias A]…` | Create or update a particular; idempotent on URI |
| `particular resolve <id\|uri\|label\|alias>` | Find particulars (label/alias case-insensitive) |
| `claim assert --subject P (--content T \| --content-file F\|-) […]` | Record a claim |
| `claim retract <id> --reason R [--superseded-by id]` | Append a retraction block to a claim or synthesis |
| `synthesis create --subject P --input id:role[:weight]… --unresolved U […]` | Record a synthesis the agent has reasoned |
| `recall [P] [--topic T]… [--scope S] [--include-retracted] [--limit N]` | Claims and syntheses in lineage order; current synthesis marked |
| `lineage <id> [--depth N]` | Provenance tree of a claim or synthesis |
| `conflicts [P] [--fail-on-conflicts]` | Unsynthesised claims and stale syntheses per particular |
| `index [--check]` | Rebuild `index.yaml`, or verify it (exit 4 on drift) |
| `validate` | Structural and referential checks; exit 4 on errors |
| `version` | Binary version and supported format (`dkf/0.1`) |

Run `particulars <verb> --help` for every flag.

### Subjects and URIs

Wherever a verb takes a particular, pass an id, URI, label, or alias. Ambiguity is
an error (exit 2) listing the candidates.

Without `--uri`, `particular define` mints one from the label:
`<base-uri><slug>` when `dkf.yaml` sets `workspace.base-uri`, otherwise
`urn:dkf:<workspace-id>:<slug>`. The same label therefore always resolves to the
same particular across sessions. For things that already have a global identifier —
Wikidata, ORCID, a GitHub URL — pass it with `--uri`.

### Conflicts are structural

`conflicts` does not judge whether two statements contradict. For each particular it
computes:

- **current** — the most recent non-retracted synthesis
- **unsynthesised** — non-retracted claims and syntheses not reconciled into it
- **stale** — syntheses that cite a retracted input

and reports the particular when there is something to reconcile. Deciding whether an
unsynthesised claim extends or contradicts the current belief is the agent's call.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | runtime error (IO, already exists, already retracted) |
| 2 | usage error (bad flags, invalid values, ambiguous subject) |
| 3 | not found (unknown id, unresolvable subject, no `resolve` match) |
| 4 | check failed (`validate`, `index --check`, `conflicts --fail-on-conflicts`) |
| 5 | no workspace found |

With `--json`, success writes one JSON object to stdout; failure writes
`{"error": {"code": "...", "message": "..."}}` to stderr and nothing to stdout.
Check-style verbs (`validate`, `index --check`, `conflicts --fail-on-conflicts`)
write their report to stdout *and* the error envelope to stderr on exit 4.

## Configuration

**Workspace selection:** `--workspace <dir>`, else `$DKF_WORKSPACE`, else the nearest
ancestor directory containing `dkf.yaml`.

**Provenance defaults** for `source.author`, `harness`, `model`: explicit flag, then
`$DKF_AUTHOR` / `$DKF_HARNESS` / `$DKF_MODEL`, then `dkf.yaml`:

```yaml
# dkf.yaml
format: dkf/0.1
workspace:
  id: 0191…                                  # UUIDv7 minted at init
  base-uri: https://example.com/particulars/ # optional
defaults:
  scope: personal                            # personal | organisation | public
  source:
    author: ben
    harness: claude
```

## Identifiers and files

Object ids are `<prefix>_<uuidv7>` — `par_`, `clm_`, `syn_` — lowercase and
time-ordered, so `ls claims/` is a chronological log and new files cluster at the
end of a PR diff. Ids written by other implementations are accepted on read.

Files are written with keys in spec order, 2-space indentation, literal block
scalars for multi-line text, and RFC 3339 UTC timestamps. The same object always
serialises to the same bytes.

`index.yaml` is a derived cache. Every mutating verb keeps it current; if a merge
conflicts on it, run `particulars index` rather than resolving by hand.

## Scope of this release

Implemented: the eight non-federation tools from the DKF spec (as CLI verbs), plus
`init`, `index`, `validate`. Not yet: federation (`feed_index`, `particular_merge`,
`knowledge_publish`), signing, and an MCP transport — the core packages are
structured so `serve --mcp` can be added without touching the format layer.

## Development

```sh
make test     # go test ./...
make lint     # go vet + golangci-lint
make cross    # darwin/linux/windows × amd64/arm64 → dist/
```

Packages: `internal/dkf` (format: types, ids, codec, field validation — no IO),
`internal/store` (workspace, files, retraction append, index), `internal/query`
(resolve, recall, lineage, conflicts, validate), `internal/cli` (cobra front-end).

## License

MIT — see [LICENSE](LICENSE). The DKF specification itself is CC0.
