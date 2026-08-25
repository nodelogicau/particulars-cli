# Verifiable provenance

DKF's first principle is that provenance is non-negotiable, and `source.document`
was the weakest field in the format: an optional, unvalidated string. A reviewer
could see that a claim cited `docs/architecture.md` and had no way to know
whether the document still said what the claim said it said — or ever had.

Two optional fields close that:

```yaml
source:
  author: ben
  harness: claude
  document:
    ref: docs/architecture.md
    hash: sha256:9f2a…
    quote: |
      In staging, the billing service listens on 443.
```

`ref` holds one of three things — a URI, a path resolved against the workspace
root, or an identifier for something that cannot be fetched at all. That third
case is why it is not called `uri`: an unfetchable source can still carry a
quote, and quoting what someone said, with nothing to fetch and no hash, is
provenance a reviewer can weigh.

A bare reference stays valid and is **not** inferior provenance:

```yaml
    document: chat session 2026-08-22
```

Some evidence cannot be fetched, hashed, or quoted, and pretending otherwise
would only push people to invent a URI.

## Writing one

```sh
particulars claim assert --subject "Billing Service" \
  --content "Invoices are generated nightly at 02:00 UTC." \
  --document src/billing/cron.go --hash-document \
  --quote 'c.AddFunc("0 2 * * *", generateInvoices)'
```

`--hash-document` computes the hash from the local file the document resolves
to. `--document-hash` records one you computed elsewhere. `--quote-file` reads
the quote from a file or stdin. All of them require `--document` — a hash with
nothing to point at records evidence for a source nobody named.

## Why a quote and not a line number

`src/billing/cron.go:14` looks precise and breaks the moment anyone inserts a
line above it. It would report drift for edits that touched nothing relevant,
and a check that cries wolf is a check reviewers learn to skip.

A quote is content-addressed. It survives insertion and reformatting, and it
does something no offset can: **a reviewer can audit the claim against its
source while reading the pull request** — no tooling, no network, no checkout.

## Drift is two signals

| quote | document hash | meaning |
|---|---|---|
| present | matches | nothing moved |
| present | differs | **`context_drift`** — the text around the quote changed |
| absent | differs | **`quote_drift`** — the cited text is gone or altered |

The middle row is why hashing the quote alone is not enough. A document reading
*"In staging, the billing service listens on 443"* yields the claim *"the
billing service listens on 443"*; changing **staging** to **production**
falsifies the claim without touching a character of the quote.

**Drift is never a validation failure.** A claim whose source has drifted stays
valid, readable, and citable — it is a condition for a reader to resolve.

## What the hash is taken over

The document's bytes, **with CRLF normalised to LF, and nothing else
normalised.** Writers write `sha256`; a reader accepts any
`<algorithm>:<digest>` and reports one it does not implement as *unverified*
rather than invalid — refusing it would leave two conforming implementations
unable to check each other's hashes.

Line endings differ between checkouts of the same commit, so hashing raw bytes
would report drift on every claim for anyone on a Windows checkout — the
cries-wolf failure again, arriving through a different door. Everything else is
left alone deliberately: trailing whitespace changes a hash without changing
meaning, but normalising it would blind the check to a class of real edit.

The cost: `shasum` on a CRLF file will not match the recorded hash. It matches
for LF files, which is what git usually hands you.

## Verification never reaches the network

`validate` checks a document only when it resolves to a file **inside the
workspace** — a relative path, or a `file:` URI. Everything else is reported as
`unverified_document`, at note severity:

```
info    claims/clm_….yaml  unverified_document: document is not a file in this
                           workspace; verification would need a fetch, which
                           validate does not do
```

That says nothing is known, not that anything is wrong. A workspace citing
mostly URLs will carry many, which is why it is a note and never a warning.

## What killed a claim

`retract --kind` records which of three joints broke:

```
   claim ─────────▶ source ─────────▶ world
     │                 │                │
  defect      provenance-failure   supersession
```

- **`defect`** — the claim misread what its source said.
- **`supersession`** — the source was right then; the world moved on.
- **`provenance-failure`** — the source itself was wrong. Every other claim
  citing that document is now worth re-reading.

Drift is reported **alongside** a declared kind, never checked against it. A
`supersession` over an unchanged document is not suspect: an ADR, an incident
report, or a commit-pinned URL is *supposed* to stay byte-identical while the
world moves on. The one sound direction is the opposite — a `defect` against a
document that has since drifted is reported `defect_unverifiable`, because the
text the claim is said to have misread is no longer the text you can read.

It is declared, never inferred from `--superseded-by`: a typo-grade defect
usually *has* a replacement, and a claim retracted because its subject was
decommissioned usually has none.

## A quote discloses its source completely

A synthesis summarises its inputs. A quote reproduces its source verbatim, so
the claim's effective scope governs that text too. `validate` notes quoted
claims shared beyond `personal`, and promoting one says so:

```
warning: clm_… carries a verbatim quote from docs/architecture.md, so promoting
         it to public publishes that source text in full
```

Reported, never refused — whether the quoted material may travel is a judgement,
and the tool's job is to make sure you know you are making it.

## A note on `uri`

v0.8.0 wrote `ref` as `uri` for one day. Readers accept it and report
`legacy_document_uri`. The warning is not tidiness: a file carrying `uri` can
never be rewritten — appending a retraction is the only permitted modification
to an existing object — so readers must go on accepting it, and the warning is
the only way anyone learns it is there.
