# Changelog

## Unreleased

- **DKF v0.1 is declared** —
  [nodelogicau/particulars@v0.1](https://github.com/nodelogicau/particulars/tree/v0.1),
  2026-08-26. This implementation is conformant in full; README and
  SPEC-FEEDBACK now say "v0.1" rather than "draft", and SPEC-FEEDBACK stands as
  the record of the nineteen items that got it there.
- `validate` errors `forbidden_alias` on object files using YAML anchors or
  aliases — the commitment from
  [#5](https://github.com/nodelogicau/particulars-cli/issues/5). Only a node
  walk can see them: a struct decode silently expands aliases, and the file
  stays readable, so validate is the one place the prohibition can live.

## v0.9.1 — report by condition

- Corpus-fact aggregation classifies **by condition, never by severity**.
  Severity was a proxy, and it aggregated `defect_unverifiable` — a rare,
  per-object finding a reviewer wants the path for — along with the wallpaper.
  Now `defect_unverifiable`, drift under a retracted object, and
  `quoted_source` list per object at info severity, while `undeclared`,
  `confidence_on_undeclared`, and `unverified_document` aggregate. Surfaced by
  the spec's `condition-reporting` capability, which classifies an unverifiable
  defect as a finding about an object; the mismatch is the answer to the second
  question on [#4](https://github.com/nodelogicau/particulars-cli/issues/4).

## v0.9.0 — the evidential axis

- **BREAKING (writers): every new claim declares what backs it.** `claim assert`
  and the MCP `claim_assert` require `--evidential observed|inferred|held`, with
  no default — if absence meant `observed`, the laziest path would produce the
  most authoritative-looking output. Readers stay lenient: every existing
  workspace remains valid, its claims reported as `undeclared` in one aggregate
  line — not a fourth value, and not a synonym for observed; the warrant cannot
  now be established, and the distinction ages out rather than being migrated.
  Implements DKF `claim-evidential` (fdab9f9), the last breaking change before
  v0.1, shaped by this implementation's review on #3 and #4.
- **`confidence` finally means something**: the inverse probability that the
  claim is mistaken, defined for `observed` and `inferred` and refused with
  `held` at write time — a position is not on that scale, and a fluent unsourced
  judgement carrying `confidence: 0.9` is the most plausible bad claim an
  agent-written format will ever hold. A file carrying both is the validate
  error `confidence_on_held`, with the claim still readable everywhere.
  Confidence on an undeclared claim is a note, never an error. The skill's old
  calibration rule — `0.9+` seen directly, `0.6–0.8` inferred — was this
  distinction smuggled into a probability, and is rewritten.
- **`method` closes to three values**: `reconciliation`, `qualification`,
  `positions`. A synthesis declares no evidential — it is backed by argument
  from its inputs. Two conflicting `held` claims still want a synthesis, a
  `positions` one. Unknown methods in existing files read leniently and warn.
- **The brief carries the register**: `[position]` on held claims,
  `[undeclared]` on pre-evidential ones, so Copilot can tell a position from an
  observed fact. Recall entries carry `evidential` when declared.
- `index.yaml` **preserves entry types it does not recognise** through rebuilds
  and incremental updates, and the drift check no longer fails on evidence of a
  newer conforming writer — implements DKF `db748da`, from this implementation's
  [particulars#16](https://github.com/nodelogicau/particulars/issues/16), where
  a stale binary one directory away nearly stripped every `publishes/` row and
  would then have reported the workspace clean.
- `validate` text output reports **corpus facts in aggregate**: the legacy
  compatibility markers (`legacy_produced_by`, `legacy_id`,
  `legacy_document_uri`) and every informational note now render as one line
  per condition carrying a count, with `--notes` expanding to per-object lines
  and `--json` unchanged. A corpus fact is permanent — the files carrying it
  can never be rewritten — so its discovery value is spent on first sight while
  a per-object listing recurs on every run forever: six identical legacy lines
  were wallpaper, and the 88-object `undeclared` report that `claim-evidential`
  will bring would have been the entire output. Findings about an object —
  drift, scope, dangling references — still list per object, because the
  object is the unit of action. From the discussion on
  [particulars-cli#4](https://github.com/nodelogicau/particulars-cli/issues/4).

## v0.8.1 — align with the drift corrections

- Align with the drift corrections applied to DKF in
  [`9388161`](https://github.com/nodelogicau/particulars/commit/9388161), after
  [particulars-cli#3](https://github.com/nodelogicau/particulars-cli/issues/3).
  **`document.uri` is now `document.ref`**, which holds a URI, a workspace path,
  *or* an identifier for something unfetchable — the case that decided the
  rename, since an unfetchable source can still carry a quote and `chat session
  2026-08-22` in a field called `uri` is a lie. Readers accept `uri` and warn
  `legacy_document_uri`: such a file can never be rewritten, so the warning is
  the only way anyone learns it is there.
- **Retracted objects' documents are now verified**, because the new
  `defect_unverifiable` finding is about the retraction rather than a live
  claim: a `defect` declared against a document that has since drifted cannot be
  checked, since the text the claim is said to have misread is no longer the text
  a reviewer can read. Drift under a retracted object is reported as an
  observation rather than a warning.
- **A supersession is never cross-checked against drift.** The spec's earlier
  SHOULD is gone and this implementation never shipped it.
- Hashes accept any `<algorithm>:<digest>`; an algorithm this implementation does
  not compute is reported *unverified* rather than invalid, so two conforming
  implementations are never unable to check each other. A digest whose algorithm
  we do implement is still checked for shape, so a truncated `sha256:abc` is
  caught as the typo it is.

## v0.8.0 — verifiable provenance

- **Verifiable provenance.** `source.document` may now be a mapping of `uri`,
  `hash`, and `quote` as well as a bare reference — which stays valid and is not
  inferior provenance. `claim assert` gains `--document-hash`, `--hash-document`
  (compute it from the local file), `--quote`, and `--quote-file`. The quote is
  the point: a reviewer can check a claim against its source **while reading the
  pull request**, with no tooling and no network.
- **Drift is two signals.** `validate` reports `context_drift` when the document
  changed around a quote that is still present, and `quote_drift` when the cited
  text is gone. Hashing the quote alone would miss a document whose "In staging"
  became "In production" without touching the quoted words. Neither is ever an
  error: a claim whose source drifted stays valid, readable, and citable.
- **A hash is taken over LF-normalised bytes and nothing else.** Line endings
  differ between checkouts of one commit, so raw bytes would report drift on
  every claim from a Windows checkout; trailing whitespace is deliberately left
  alone, because normalising it would blind the check to a class of real edit.
  The spec leaves this open — the reasoning is on
  [particulars-cli#3](https://github.com/nodelogicau/particulars-cli/issues/3).
- **Verification never reaches the network.** A document is checked only when it
  resolves to a file in the workspace; everything else is `unverified_document`
  at the new `info` severity, which records that provenance was not
  machine-checked rather than that anything is wrong. `validate` now prints
  notes separately from warnings.
- **`retract --kind defect|supersession|provenance-failure`** records which of
  the three joints in `claim → source → world` broke. Declared, never inferred
  from `--superseded-by`. The spec's suggested cross-check of kind against drift
  is **not** implemented, and `validate` carries a comment saying why.
- A quote discloses its source completely, where a synthesis summarises — so
  promoting a quoted claim now says so, and `validate` notes quoted claims shared
  beyond `personal`.
- Skill: stop teaching `--document src/billing/cron.go:14`. A line offset rots on
  the next edit above it; the skill now teaches `--quote`, what to quote, and
  that a quote travels with the claim when it is promoted.

- Examples: pin `PARTICULARS_VERSION` to v0.7.0 in both workflows (`dkf-check.yml`
  was still on v0.2.0) and say why the pin is load-bearing. An older binary does
  not read record types it predates: `dkf-check` rebuilds `index.yaml` without
  their entries and fails on `index_stale`, and `graph-sync` cannot see promotion
  records, so it would publish only what the asserted scopes allow — less than was
  authorised, quietly. Raised upstream as
  [particulars#16](https://github.com/nodelogicau/particulars/issues/16).

## v0.7.0 — promotion records and effective scope

- **Promotion records and effective scope.** `particulars publish <id>… --scope <s>`
  writes `publishes/pub_….yaml` naming claims and syntheses to share more widely.
  Claims are immutable, so scope is never rewritten: an object's *effective* scope
  is its asserted scope widened by the non-retracted promotions covering it. A
  workspace written entirely at `personal` becomes exportable by promotion, keeping
  every id and the whole lineage — which is what
  [particulars#14](https://github.com/nodelogicau/particulars/issues/14) asked for,
  after this implementation's own Graph export produced zero items from 38 claims.
- **Promotion may only widen**, measured against asserted scope so that a record's
  validity never depends on the order records were written. Reduce exposure by
  retracting the promotion, not by promoting downwards. **Promotion does not
  cascade** to a synthesis's inputs, and there is deliberately no flag that
  promotes a chain in one step — that is the silent lineage-widening the rule
  exists to prevent.
- `recall --scope`, `topics --scope`, `export --format graph`, and
  `export --format dot|mermaid --scope` all filter on effective scope. Index
  entries for promotions carry `claims` and `scope`, so effective scope is
  computable from `index.yaml` without opening an object file.
- `scope_wider_than_inputs` now compares **effective** scope on both sides, and is
  evaluated at the three points the spec names: on `synthesis create` (reported,
  never blocking — a wider synthesis is permitted), on promotion, and during
  `validate`. Comparing asserted scope went wrong in both directions once
  promotions existed: it warned about inputs promoted to match, and stayed silent
  when a synthesis was promoted past inputs that were not — the case it exists to
  catch, and one in which **neither file changes**. The message now names the
  promotion responsible for a scope.
- `knowledge_publish` joins the MCP tool list, completing the spec's tool surface
  at twelve. `retract` accepts `pub_` ids and refuses `--superseded-by` for them,
  as it does for merges.
- New warnings `promotion_of_retracted` and `duplicate_promotion`. A workspace with
  no `publishes/` directory behaves exactly as before.
- Skill: tell agents to show the shape, not just the diff. When writing a branch
  up for a human, draw each particular touched with `export --format mermaid
  --subject <thing> --depth 2` and paste it into a fenced block — the reviewer
  then sees which claim was cited as the antithesis and whether the synthesis
  became the current belief, which a list of filenames cannot convey. Carries the
  two cautions with it: a diagram discloses whatever it contains, and CI may
  already be posting one.

## v0.6.0 — draw the workspace

- `particulars export --format dot|mermaid` draws the workspace. With
  `--subject`, one particular's dialectic: claims and syntheses as nodes, inputs
  as edges labelled with their role, the current belief emphasised, and
  `unsynthesised`, `stale`, retracted, and cross-particular inputs each marked
  distinctly. Without one, a map of every particular, weighted by what is known
  about it and joined by merge records. `--depth` bounds the lineage;
  `--include-retracted` draws what was withdrawn, with `superseded-by` as its own
  edge.
- Mermaid renders with no tooling — in a pull request comment, an issue, or a
  markdown file — so `docs/examples/dkf-check.yml` gains a commented-out step
  that draws the particulars a pull request touches. DOT is for Graphviz when a
  workspace outgrows what a comment can show. Neither requires Graphviz to be
  installed to *emit*.
- Unlike `--format graph`, the drawings include `personal` knowledge by default:
  writing a local file is not a transfer to a third party, and a diagram missing
  half the graph misrepresents the reasoning. `--scope` narrows it, and
  `docs/visualise.md` states plainly that publishing a diagram is the same
  judgement as publishing the objects it draws.

## v0.5.2 — unsubstituted template variables

- Treat an unsubstituted template variable as absent everywhere provenance is
  resolved. Claude Desktop expands `${user_config.author}` only when the user
  fills the field in; left blank, the literal string reached the server and was
  recorded as `source.author` on every claim written from that conversation. A
  `${…}` value in a flag, a `DKF_*` variable, or `dkf.yaml` now falls through to
  the next layer, so a blank field yields the workspace default (or no author at
  all, which the format allows alongside a harness). An unsubstituted
  `--workspace` is reported as such rather than as a missing `dkf.yaml`.

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
