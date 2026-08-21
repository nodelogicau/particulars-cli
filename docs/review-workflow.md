# Reviewing agent knowledge through pull requests

`particulars` has no review state of its own. It leans on git, which already gives
every DKF workspace version history, authorship, diffs, and a review UI.

```
   agent (branch)                      human (PR)                      main
   ──────────────                      ──────────                      ────
   particulars claim assert …   ─┐
   particulars claim assert …    ├──▶  + claims/clm_….yaml  ×2
   particulars synthesis create ┘       + syntheses/syn_….yaml
                                        ± index.yaml
                                              │
                                        read the YAML, follow the
                                        evidence in source.document
                                              │
                                        merge ─────────────────────▶ accepted
                                        close ─────────────────────▶ never happened
                                        merge, later regret ───────▶ claim retract
```

## The agent's side

0. Once per repo: `particulars skill install`, so the agent has the verbs and
   the discipline below in its context; if the workspace lives in a
   subdirectory, `particulars init ./knowledge --pointer` (or a hand-written
   `.dkf`) so every verb finds it from the repo root. Per remote session, the
   `SessionStart` hook in `docs/examples/claude-settings.json` does both.
1. Work on a branch. Everything the agent learns goes through the CLI:
   `particular define`, `claim assert`, `synthesis create`.
2. Before asserting, the agent checks what it already knows —
   `particular resolve`, `recall`, `conflicts` — so it links to existing
   particulars and reconciles instead of duplicating.
3. `recall` on the branch sees the agent's own unmerged claims, so memory is
   consistent within a session.
4. Open a PR. Nothing else: the CLI never commits, branches, or pushes.

## The reviewer's side

A DKF PR is easy to read because of three invariants the tool enforces:

- **New knowledge is new files.** Anything under `claims/` or `syntheses/` that is
  *added* is a proposal. Ids are time-ordered, so the newest files sort last.
- **A modified claim is always a retraction.** The only legal change to an existing
  claim or synthesis file is an appended `retracted:` block. If a diff shows
  anything else changing inside a claim, something is wrong — `validate` will flag
  it as non-canonical, and the reviewer should reject.
- **`index.yaml` is derived.** Its diff is noise; if it conflicts, rebuild it with
  `particulars index` instead of merging by hand.
- **A merge record is a claim about identity.** A new file under `merges/` says two
  URIs are the same thing; nothing else moves. Check both URIs really denote one
  particular — a wrong merge silently pools two particulars' knowledge until it is
  retracted.

What to look at in each new claim:

| Field | Ask |
|---|---|
| `content` | Is this a single, falsifiable statement? |
| `subject` | Is it about the right particular? (`particular resolve <id>`) |
| `source.document` | Can I open this and see the evidence? |
| `confidence` | Does it match the strength of the evidence? |
| `context.scope` | Is `organisation`/`public` appropriate, or should this stay `personal`? |

And in each synthesis:

| Field | Ask |
|---|---|
| `inputs` | Are the thesis/antithesis roles right? Is anything missing? (`particulars lineage <id>`) |
| `unresolved` | Is it honest about what was *not* reconciled? (`None identified` means "considered, nothing open") |
| `source.harness` | Which harness produced it? (`source` replaced `produced-by` in v0.2.0) |
| `content` | Does it carry the reasoning, not just the conclusion? |

Workspaces written before v0.2.0 hold syntheses with `produced-by`. They remain
valid; `validate` reports each as a `legacy_produced_by` warning until a newer
synthesis supersedes it. Do not rewrite them.

## After merge

- **Something was wrong.** Retract it; never delete or edit:
  ```sh
  particulars retract clm_… --reason "…" [--superseded-by clm_…]
  ```
  The retraction records who and when. `recall` stops returning it, and any
  synthesis that cited it is reported by `conflicts` as **stale** until a new
  synthesis supersedes it.
- **A PR was closed unmerged.** It leaves no trace — the agent may assert the same
  thing again later. If you want the rejection remembered, merge and retract with a
  reason instead.

## CI check

Add this to the repository that holds the workspace. It fails the PR when the
workspace is structurally invalid or the index is stale, and it leaves a comment
listing what still needs synthesis.

```yaml
# .github/workflows/dkf-check.yml
name: DKF check
on:
  pull_request:
    paths: ["knowledge/**"]          # adjust to your workspace path

permissions:
  contents: read
  pull-requests: write

jobs:
  check:
    runs-on: ubuntu-latest
    env:
      DKF_WORKSPACE: ${{ github.workspace }}/knowledge
      PARTICULARS_VERSION: v0.2.0
    steps:
      - uses: actions/checkout@v4

      - name: Install particulars
        run: |
          curl -sSL -o particulars.tgz \
            "https://github.com/nodelogicau/particulars-cli/releases/download/${PARTICULARS_VERSION}/particulars_${PARTICULARS_VERSION#v}_linux_amd64.tar.gz"
          # extract to a temp dir: a workspace at the repo root has a particulars/ directory
          tar -xzf particulars.tgz -C "${RUNNER_TEMP}" particulars
          sudo install "${RUNNER_TEMP}/particulars" /usr/local/bin/particulars
          particulars version

      - name: Validate workspace
        run: particulars validate

      - name: Index is current
        run: particulars index --check

      - name: Report open conflicts
        if: always()
        run: |
          {
            echo '### Knowledge awaiting synthesis'
            echo
            echo '```'
            particulars conflicts || true
            echo '```'
          } > conflicts.md
          cat conflicts.md >> "$GITHUB_STEP_SUMMARY"

      - name: Comment on PR
        if: always() && github.event_name == 'pull_request'
        uses: actions/github-script@v7
        with:
          script: |
            const fs = require('fs');
            const body = fs.readFileSync('conflicts.md', 'utf8');
            await github.rest.issues.createComment({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              body,
            });
```

A copy of this workflow lives at [`examples/dkf-check.yml`](examples/dkf-check.yml).

To make unsynthesised knowledge itself block a merge, add
`--fail-on-conflicts` to the `conflicts` step — useful for workspaces where every
PR is expected to leave the particular it touches reconciled.
