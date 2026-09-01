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

This is the reference implementation of **DKF v0.1**, declared on 2026-08-26 at
[nodelogicau/particulars@v0.1](https://github.com/nodelogicau/particulars/tree/v0.1).
[SPEC-FEEDBACK.md](SPEC-FEEDBACK.md) records what this implementation proposed
and how each point was decided. v0.2.0 aligns the CLI with those decisions; see
[CHANGELOG.md](CHANGELOG.md).

## Install

```sh
# macOS — Homebrew cask
brew install nodelogicau/tap/particulars

# Linux or macOS, including CI and agent sandboxes — verifies the release checksum
curl -sSL https://raw.githubusercontent.com/nodelogicau/particulars-cli/main/install.sh | sh
```

The script never prompts. It installs to `/usr/local/bin` when it can (directly, or
via passwordless `sudo`), otherwise to `~/.local/bin`, and says where. Knobs:
`PARTICULARS_VERSION=v0.2.1` pins a release, `PARTICULARS_INSTALL_DIR=…` picks the
directory. Windows: download the `.zip` from the
[releases page](https://github.com/nodelogicau/particulars-cli/releases).

From source (Go 1.26+): `git clone … && make build` → `dist/particulars`.

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
  merges/mrg_….yaml        merge records (two URIs declared the same particular)
  index.yaml               derived manifest (regenerate with `particulars index`)
```

## How review works

The agent works on a branch and opens a pull request. The diff *is* the proposal:
new files under `claims/` and `syntheses/` are new knowledge; a modified claim file
can only ever be a retraction. Merging is acceptance. If something merged turns out
to be wrong, retract it — never delete it:

```sh
particulars retract clm_0191…a --reason "Port is 8443, not 443" \
    --superseded-by clm_0191…c
```

Retraction appends a `retracted:` block to the claim file, recording who, when, and
why. `recall` hides retracted claims by default; `conflicts` flags syntheses that
cite them as **stale** so they can be re-synthesised.

See [docs/review-workflow.md](docs/review-workflow.md) for the full flow and a
GitHub Action that runs `validate` and `index --check` on every PR.

## Teaching an agent the verbs

The binary carries an agent-facing skill — the recall-before-assert loop, a verb
cheat-sheet, and rules for good claims, syntheses, and retractions — stamped with
its own version so skill and verbs cannot drift. This is the recommended setup for
any harness with a shell; for one without, see
[Use from Claude Desktop](#use-from-claude-desktop-or-any-mcp-client). Install it
for your harness:

| `--harness` | Writes | Read by |
|---|---|---|
| `claude` (default) | `.claude/skills/particulars/SKILL.md` | Claude Code, GitHub Copilot |
| `copilot` | `.github/skills/particulars/SKILL.md` | GitHub Copilot |
| `agents` | `.agents/skills/particulars/SKILL.md` | GitHub Copilot; the vendor-neutral Agent Skills location |
| `cursor` | `.cursor/rules/particulars.mdc` | Cursor |
| `agents-md` | a marker-bounded section in `AGENTS.md` (`--file` to retarget, e.g. `GEMINI.md`) | Codex, Jules, Gemini CLI, Windsurf, Zed, Cursor, Copilot, … |

```sh
particulars skill install                      # Claude Code (and Copilot)
particulars skill install --harness cursor     # Cursor rule
particulars skill install --harness agents-md  # section in AGENTS.md; the rest of the file is untouched
particulars skill install --user               # personal location (claude/copilot/agents presets)
particulars skill show --harness agents-md     # print any variant to pipe elsewhere
```

**GitHub Copilot reads all three skills directories** (`.claude/skills`,
`.github/skills`, `.agents/skills`) — install to exactly one, or it loads the
skill twice; `install` warns if it sees a second. For a repository serving
several harnesses, `agents` is the neutral choice. `--dir <path>` writes a plain
`SKILL.md` anywhere (not useful for Cursor, which ignores `.md` in its rules
directory).

`install` never overwrites a skill file it did not write (pass `--force` once if
you copied it by hand before v0.2.1), and in `AGENTS.md` it owns only the region
between its markers. `skill install --check` verifies without writing (exit 4 on
drift) — handy in CI. The canonical text lives at
[`skills/particulars/SKILL.md`](skills/particulars/SKILL.md).

For a zero-setup remote session, commit a `.dkf` pointer and add the Claude Code
`SessionStart` hook from
[`docs/examples/claude-settings.json`](docs/examples/claude-settings.json): it
installs the binary if missing and installs the skill, on every session start.

## Use from Claude Desktop or any MCP client

`particulars serve --mcp` serves one workspace over stdio with the DKF
specification's tool names (`claim_assert`, `knowledge_recall`, …) and results
identical to `--json`. The agent discipline travels in the server's `instructions`.
Claude Desktop users install the `.mcpb` bundle from the releases page and pick a
workspace folder — that is what the server is for: harnesses with no shell.

**In a harness that already has a shell — Claude Code, Cursor's agent — prefer the
skill and the CLI above.** The server's tools and instructions cost about 4,900
tokens of context in every session, while the skill stays collapsed to ~86 tokens
until something triggers it; the shell also composes (`recall --json | jq …`) and
resolves the workspace on every invocation rather than binding it at startup.
Reasoning, numbers, and the cases that do favour the server:
[`docs/mcp.md`](docs/mcp.md#which-should-i-use--the-mcp-server-or-the-skill-and-cli).

Details, tool table, and client configs: [`docs/mcp.md`](docs/mcp.md).

**Microsoft 365 Copilot** is a different case again: it runs in Microsoft's cloud
and cannot reach a local server at all. There, merged knowledge is *exported* into
Microsoft Graph, where it is indexed and cited — `particulars export --format graph`,
pushed by a workflow when a knowledge pull request is merged. See
[`docs/graph.md`](docs/graph.md).

## Verbs

| Verb | Purpose |
|---|---|
| `init [dir] [--pointer]` | Create a workspace; `--pointer` also writes `./.dkf` pointing at it |
| `workspace` | Show the resolved workspace root and how it was found (`flag`/`env`/`dkf.yaml`/`pointer`) |
| `workspace pointer [dir] [--at D] [--force]` | Write a `.dkf` pointer to an existing workspace |
| `particular define --label L [--uri U] [--alias A]…` | Create or update a particular; idempotent on URI |
| `particular resolve <id\|uri\|label\|alias>` | Find particulars (label/alias case-insensitive) |
| `claim assert --subject P (--content T \| --content-file F\|-) --evidential observed\|inferred\|held [--author A] [--document D] [--document-author W] [--hash-document] [--quote Q] […]` | Record a claim with its warrant, optionally with checkable evidence; `--document-author` names who produced what was read (see [docs/provenance.md](docs/provenance.md)) |
| `retract <id> --reason R [--superseded-by id]` | Append a retraction block to a claim, synthesis, or merge (`claim retract` is an alias) |
| `particular merge <a> <b> [--reason R]` | Declare two URIs the same particular; writes `merges/mrg_….yaml` |
| `synthesis create --subject P --input id:role[:weight]… --unresolved U […]` | Record a synthesis the agent has reasoned |
| `recall [P] [--author W] [--topic T]… [--scope S] [--include-retracted] [--limit N]` | Claims and syntheses in lineage order; current synthesis marked. `--author` returns what a particular asserted or is reported for, labelled `relations` |
| `topics [P] [--scope S] [--include-retracted]` | Topics in use, with assertion and particular counts |
| `lineage <id> [--depth N]` | Provenance tree of a claim or synthesis |
| `conflicts [P] [--fail-on-conflicts]` | Unsynthesised claims and stale syntheses per particular |
| `index [--check]` | Rebuild `index.yaml`, or verify it (exit 4 on drift) |
| `validate` | Structural and referential checks; exit 4 on errors |
| `serve --mcp [--workspace D]` | Serve the workspace to an MCP client over stdio (see [docs/mcp.md](docs/mcp.md)) |
| `publish <id>… --scope S [--reason R]` | Share claims and syntheses more widely; writes a promotion record (widen-only, never cascades) |
| `export --format graph [--schema]` | Emit Microsoft Graph items so merged knowledge is searchable in M365 Copilot (see [docs/graph.md](docs/graph.md)) |
| `export --format dot\|mermaid [--subject P]` | Draw the workspace: one particular's dialectic, or a map of them all (see [docs/visualise.md](docs/visualise.md)) |
| `skill show` / `skill install [--harness P]… [--user\|--dir D] [--file F] [--force] [--check]` | Print or install the embedded agent skill for Claude Code, Copilot, Cursor, or AGENTS.md |
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

### Provenance

Every `source` (on claims, syntheses, retractions, and merges) needs at least one of
`author` or `harness`; an agent acting alone is a valid source. Syntheses additionally
require `source.harness`. Files written by v0.1.x carried `produced-by` on syntheses;
they are still read (as `source`) and `validate` marks them `legacy_produced_by`.

### Merges

`particular merge` declares that two URIs denote the same thing and writes a small
record — nothing is moved or rewritten. Non-retracted merges are symmetric and
transitive, so merged particulars form an equivalence class that `recall`,
`conflicts`, and `lineage` operate over; each claim keeps its own `subject`. Either
side may be a URI with no local particular, which is how a workspace bridges to a
public identifier. Undo a merge with `retract <mrg_id>`.

### Conflicts are structural

`conflicts` does not judge whether two statements contradict. For each particular (or
merge class) it computes:

- **current** — the most recent non-retracted synthesis, by `timestamp` then id
- **unsynthesised** — non-retracted claims and syntheses not reconciled into it
- **stale** — syntheses that cite a retracted input, directly or transitively

and reports the particular when there is something to reconcile. Deciding whether an
unsynthesised claim extends or contradicts the current belief is the agent's call.
`recall` marks the same facts on each entry (`current`, `unsynthesised`).

A synthesis that settles everything still needs `--unresolved`; the conventional value
is the exact string `None identified`.

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
ancestor directory containing `dkf.yaml` — or a `.dkf` pointer file whose first line
is the path to the workspace. An explicit directory may likewise be either the
workspace root or one holding a `.dkf` pointer to it. `particulars workspace` shows
what was resolved and how.

A workspace kept in a subdirectory of a repository (the `knowledge/` convention) is
invisible to upward discovery from the repo root; a pointer fixes that:

```sh
particulars init ./knowledge --pointer     # at creation: writes ./.dkf containing "knowledge"
particulars workspace pointer ./knowledge  # afterwards: same file, workspace already exists
particulars workspace                      # → …/knowledge  via: pointer
```

The path is written relative when the workspace is inside the pointer's directory,
so the file survives being cloned elsewhere, and absolute when it is not — in which
case it is machine-specific and should stay out of git.

A workspace can teach agents its own conventions — topic vocabulary, local
naming — via `CONVENTIONS.md` at the root (or `workspace.conventions: <file>` in
`dkf.yaml`): `serve --mcp` delivers it with the `initialize` instructions, so it
reaches MCP-only clients that never read the repository, and
`particulars workspace` shows which file applies.

`.dkf` is an implementation extension; spec-conformant readers still find the
workspace by walking up from inside it. `workspace.base-uri`, if set, must end in
`/` (`init` adds it if missing).

**Authors are particular references.** `source.author` (and a document's `author`)
may be a particular id, URI, or bare name; readers resolve names by label or
alias at read time, so defining a particular with alias `ben` retroactively
attributes every claim already carrying `author: ben`, no file changing. Writers
prefer the URI: a value resolving to exactly one particular is written as that
particular's `uri` — the identifier that survives leaving the workspace, frozen
at the moment it was unambiguous. An unknown `par_` id is refused; an explicitly
given ambiguous name is refused with the candidates; an ambiguous or unknown
default is written unchanged (and `validate` aggregates it). People are defined
deliberately — `init` never mints the author's particular; run
`particular define --label "Ben Fortuna" --uri <chosen-uri> --alias ben` once,
with the URI its owner is willing to be cited under.

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

Object ids are `<prefix>_<uuidv7>` — `par_`, `clm_`, `syn_`, `mrg_` — lowercase and
time-ordered, so `ls claims/` is a chronological log and new files cluster at the
end of a PR diff. Ids written by other implementations are accepted on read.

Files are written with keys in spec order, 2-space indentation, literal block
scalars for multi-line text, and RFC 3339 UTC timestamps. The same object always
serialises to the same bytes.

`index.yaml` is a derived cache. Every mutating verb keeps it current; if a merge
conflicts on it, run `particulars index` rather than resolving by hand.

## Scope of this release

Implemented: the eight non-federation tools from the DKF spec plus `particular_merge`
— as CLI verbs and as MCP tools (`serve --mcp`) — and `init`, `index`, `validate`,
`topics`, `skill`, `workspace`, `publish`. Not yet: `feed_index`, signing, and a
hosted/HTTP server.

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
