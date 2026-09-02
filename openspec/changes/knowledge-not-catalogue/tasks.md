## 1. Skill

- [x] 1.1 Register rule in *Rules for good claims*: deletion test, nobody-recalls-a-feed, the legitimate document-subject case
- [x] 1.2 ✗/✓ contrast example (feed-as-subject vs fact-with-document)
- [x] 1.3 Regenerate the installed copy from a HEAD build; `skill install --check` green

## 2. Tool surface

- [x] 2.1 `claim_assert` description + `particular_id`/`content` schema clauses
- [x] 2.2 `particular_define` description: identity examples, thing-in-the-world clause

## 3. Validate

- [x] 3.1 `url_in_content` (info, claims only, `https?://` match) on the corpus-fact whitelist
- [x] 3.2 Tests: aggregate line, json carries all, no note for documents in their place, no note on syntheses

## 4. Docs

- [x] 4.1 docs/mcp.md conventions section: ingestion register belongs in the workspace conventions file
- [x] 4.2 docs/provenance.md: the deletion test beside the quote guidance

## 5. Verification

- [x] 5.1 `go test ./...`, `golangci-lint`, `openspec validate knowledge-not-catalogue`
- [ ] 5.2 Measure: URL-bearing-content rate per source.model in the observing workspace, before/after (recorded on the change's issue or PR)
