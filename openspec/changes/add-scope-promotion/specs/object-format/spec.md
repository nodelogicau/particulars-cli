## MODIFIED Requirements

### Requirement: Identifier format
Object and record identifiers SHALL have the form `<prefix>_<uuid>` where `prefix` is `par` for particulars, `clm` for claims, `syn` for syntheses, `mrg` for merge records, `pub` for promotion records, and `uuid` is a lowercase, hyphenated, canonical UUID version 7 (RFC 9562). Minting SHALL use a monotonic counter so that identifiers minted by one process are strictly increasing in lexical order. On read, the CLI SHALL accept any identifier matching `^(par|clm|syn|mrg|pub)_[A-Za-z0-9-]+$` so that workspaces written by other implementations remain readable. The canonical pattern SHALL admit every minted prefix, so that an identifier this implementation mints is never reported as `legacy_id`.

#### Scenario: Minted identifier shape
- **WHEN** a claim is created
- **THEN** its id matches `^clm_[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`

#### Scenario: Merge identifier shape
- **WHEN** a merge record is created
- **THEN** its id starts with `mrg_` followed by a canonical lowercase UUIDv7

#### Scenario: Promotion identifier shape
- **WHEN** a promotion record is created
- **THEN** its id starts with `pub_` followed by a canonical lowercase UUIDv7, and `validate` reports no `legacy_id` warning for it
