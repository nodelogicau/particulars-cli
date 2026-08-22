# Changelog

## v0.5.1 — scope warning, `workspace pointer`, skill corrections

Everything here came out of using the tool on real knowledge rather than reading
the code: publishing a synthesis to Copilot, and being misled twice by the skill.

- `validate` warns `scope_wider_than_inputs` when a synthesis is shareable more
  widely than an assertion it reasons from. Scope is declared per assertion and
  is **not** inherited, so an `organisation` synthesis of `personal` claims is
  published while its evidence is withheld — its content can carry their
  substance across the boundary anyway. The warning names the narrower inputs;
  it stays a warning because cross-scope reasoning is legitimate and the call
  belongs to the human reviewing the PR. Raised upstream as
  [particulars#15](https://github.com/nodelogicau/particulars/issues/15).

- `particulars workspace pointer [dir]` writes a `.dkf` pointer for a workspace
  that already exists — previously only `init --pointer` could, so a workspace
  predating the repository layout around it had to have the file hand-written.
  Relative target inside the tree, absolute outside with a warning not to commit
  it; refuses to shadow a `dkf.yaml` or to silently replace a pointer naming a
  different workspace.

- Skill: state workspace precedence in the right order. The setup note listed
  `dkf.yaml`, `$DKF_WORKSPACE`, `--workspace` — which is precedence *backwards*,
  leading with the one that loses. An agent reading it would expect a `dkf.yaml`
  underfoot to override an already-set `$DKF_WORKSPACE`; it does not.

- Skill: correct the zsh guidance for synthesis inputs. The previous note said
  to quote ids, but quoting does not help — zsh applies the `:t` history
  modifier inside double quotes, so `"$id:thesis"` silently becomes
  `…hesis` and the call fails. The skill now tells agents to brace: `"${id}:thesis"`.

No format or on-disk change: v0.5.0 workspaces are read and written identically.

## v0.5.0 — publish knowledge to Microsoft 365 Copilot

- `particulars export --format graph` emits Microsoft Graph `externalItem`
  payloads as NDJSON — one item per particular, carrying the current belief, what
  it could not reconcile, and the claims that support it. `--schema` emits the
  connection and schema registration payloads; `--manifest` writes the exported
  ids so a sync job can delete what no longer qualifies.
- **`personal` knowledge is never exported**, and `--scope personal` is refused;
  retracted claims, syntheses, and merges never appear, and a retracted synthesis
  is never treated as the belief.
- The command emits only: it makes no network request, and no Graph SDK or
  authentication library enters this codebase. `docs/examples/graph-sync.yml`
  shows the push, authenticating with GitHub OIDC federated to Entra so no client
  secret is stored.
- `docs/graph.md` covers what is indexed, the one-time connection setup, deletion,
  and the data-movement trade-off against a federated connector.

## v0.4.1 — fix the Desktop bundle on Intel Macs

- `particulars-0.4.0.mcpb` shipped an arm64-only macOS binary and could not start
  on Intel Macs. The bundle script globbed the wrong directory for GoReleaser's
  universal binary and, on the Linux release runner (no `lipo`), silently fell
  back to a single architecture. It now matches any `*_darwin_all` directory,
  refuses to write a single-architecture bundle, and is built on a macOS CI job
  that asserts the binary is universal. No change to the CLI or the server.

## v0.4.0 — MCP server and Claude Desktop bundle

- `particulars serve --mcp`: a stdio Model Context Protocol server bound to one
  workspace (flag, `DKF_WORKSPACE`, or `dkf.yaml`/`.dkf` discovery from the cwd).
  Tools use the DKF specification's names — `particular_define`, `particular_resolve`,
  `particular_merge`, `claim_assert`, `claim_retract`, `synthesis_create`
  (with `particular_id`), `knowledge_recall`, `conflict_detect` (particular or
  claim-set form), `lineage_trace` — plus two labelled extensions, `topics_list`
  and `workspace_status`. Results equal the CLI's `--json`; errors are tool results
  carrying the CLI's codes; `particular_resolve` returns `null` on a miss.
- The agent discipline is delivered as MCP `instructions` and as the prompt
  `particulars-discipline`; `source.harness` defaults to the client's name from
  the handshake.
- `particulars-<version>.mcpb`, a Claude Desktop extension bundle (macOS universal
  binary + Windows x64) with a workspace-folder picker, attached to each release.
- Shared `internal/apperr` and `internal/prov` packages so CLI and MCP map errors
  and provenance identically; `query.AnalyseSet` for claim-set conflict analysis.

## v0.3.1 — install the skill for any harness

- `skill install --harness <preset>` (repeatable): `claude` (default),
  `copilot` (`.github/skills/`), `agents` (`.agents/skills/`, vendor-neutral),
  `cursor` (`.cursor/rules/particulars.mdc` with Cursor frontmatter), and
  `agents-md` (a marker-bounded section in `AGENTS.md`; `--file` to retarget).
  `--user` for the presets that have a personal location.
- `skill show --harness <preset>` prints exactly what `install` would write.
- `--check` is per target and, for `agents-md`, checks only the owned section.
- Warns when a second GitHub Copilot-readable skills directory already holds the
  skill (Copilot loads `.claude/skills`, `.github/skills`, and `.agents/skills`).

## v0.3.0 — one-command install, workspace pointer

- `install.sh` (Linux/macOS, POSIX sh): resolves the latest release or
  `PARTICULARS_VERSION`, verifies the published SHA-256, installs to
  `PARTICULARS_INSTALL_DIR` / `/usr/local/bin` / `~/.local/bin` without prompting.
  Exercised in CI weekly and on change.
- Homebrew (macOS): `brew install nodelogicau/tap/particulars` — a cask published
  by the release pipeline to `nodelogicau/homebrew-tap`.
- `.dkf` workspace pointer: a file whose first line is the workspace path lets
  verbs run from a repository root (or anywhere above the workspace) find it.
  `init <dir> --pointer` writes it. `dkf.yaml` always wins over a pointer in the
  same directory; pointers do not chain.
- New `workspace` verb: prints the resolved root and how it was found
  (`flag`, `env`, `dkf.yaml`, `pointer`); exit 5 when nothing resolves.
- The documented Claude Code `SessionStart` hook is now live: installer + skill.

## v0.2.1 — the skill ships in the binary

- New `skill` verb: `skill show` prints the embedded agent skill; `skill install`
  writes it to `./.claude/skills/particulars/SKILL.md` (default), `~/.claude/…`
  (`--user`), or `--dir <path>`. The text is stamped with the binary version and
  an ownership marker; files the tool did not write are refused without `--force`;
  `--check` verifies without writing (exit 4 on drift), ignoring the stamped version.
- The repository's own `.claude/skills/particulars/SKILL.md` is now generated
  (`make skill`) and verified in CI.
- Docs: one-command agent setup in the README; a sample Claude Code
  `SessionStart` hook in `docs/examples/claude-settings.json`.

## v0.2.0 — align with the resolved DKF v0.1 draft

Implements the decisions recorded in
[nodelogicau/particulars@27743db](https://github.com/nodelogicau/particulars/commit/27743db)
(2026-08-21), which resolved all ten feedback issues from this implementation.

**Breaking (JSON and on-disk shape of syntheses)**

- Syntheses carry `source` (author?, harness, model?, document?) in place of
  `produced-by`. `synthesis create` writes `source`; `--json` output uses `source`.
  Files written by v0.1.x are still read (their `produced-by` becomes `source`) and
  `validate` reports them as `legacy_produced_by` warnings. No rewrite.

**Relaxed**

- A `source` needs at least one of `author` or `harness` — on claims, syntheses,
  retractions, and merges. Agent-only provenance is valid. (v0.1.x required `author`.)

**New**

- Merge records: `particular merge <a> <b>` writes `merges/mrg_<uuidv7>.yaml`.
  Non-retracted merges define equivalence classes that `recall`, `conflicts`, and
  `lineage` operate over; `recall --json` lists `class`, `conflicts` lists `members`.
- Top-level `retract <id>` for claims, syntheses, and merges. `claim retract` remains
  as an alias.
- `synthesis create --author`, `--document`.
- `recall` entries carry `source` and `unsynthesised`; `lineage` shows
  `superseded_by` on retracted nodes.
- `validate` findings: `conflicting_provenance`, `invalid_merge`, `invalid_base_uri`
  (errors); `legacy_produced_by`, `legacy_id`, `unknown_merge_uri`, `duplicate_merge`
  (warnings). `stale_synthesis` is now transitive.

**Changed**

- `current` synthesis is chosen by `timestamp` then id (was id only).
- `stale` includes syntheses citing a retracted input transitively (was direct only).
- `workspace.base-uri` must end in `/`: `init` appends it and says so; an existing
  `dkf.yaml` without it fails to open. `MintURI` no longer special-cases `#`/`:`.
- `init` creates `merges/`. Identifier grammar accepts `mrg_`.

## v0.1.1

- `topics` verb; agent-facing `skills/particulars/SKILL.md`; issue links in
  `SPEC-FEEDBACK.md`.

## v0.1.0

- Initial release: `init`, `particular define|resolve`, `claim assert|retract`,
  `synthesis create`, `recall`, `lineage`, `conflicts`, `index`, `validate`.
