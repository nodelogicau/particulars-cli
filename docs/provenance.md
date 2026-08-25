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
    uri: docs/architecture.md
    hash: sha256:9f2a…
    quote: |
      In staging, the billing service listens on 443.
```

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
normalised.**

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

## Not implemented

The spec suggests cross-checking `kind` against drift, treating a
`supersession` over an unchanged hash as suspect. This implementation does not,
and [particulars-cli#3](https://github.com/nodelogicau/particulars-cli/issues/3)
records why: drift is a signal about the *source* joint while supersession is
the *world* joint, so the check only holds for documents that describe current
state. A claim sourced from an ADR, an incident report, or a commit-pinned URL
is *supposed* to be superseded while its document stays byte-identical.
