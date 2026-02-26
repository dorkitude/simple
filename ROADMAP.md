# ROADMAP

This roadmap reflects the current codebase and a DNSimple API coverage audit performed on February 26, 2026.

## Current State

The app is already strong for day-to-day DNS browsing and core DNS changes:

- TUI: auth, tabbed navigation, global domain search, domain dashboard, diagnostics, guarded mutations
- CLI: auth, whoami, domains, zones, records (including record create/update/delete)
- Demo mode: full TUI against a seeded in-memory backend (`simple demo`)

The next phases focus on closing high-value gaps in DNS workflows, then expanding toward broader DNSimple platform coverage.

## Principles

- Prefer high-leverage features first (things that reduce repetitive DNS work)
- Keep CLI and TUI capabilities aligned where practical
- Use confirmation dialogs and explicit target labeling for all TUI mutations
- Preserve demo mode parity by implementing new features through the shared backend interface

## Phase 1: TUI DNS Editing Parity (High Value, Near-Term)

Goal: make the TUI a complete daily driver for common DNS record operations.

### TUI Record CRUD

- Add record create modal (type, name, content, TTL, priority, regions)
- Add record edit modal (pre-filled values)
- Add record duplicate / clone action
- Add record disable/enable patterns where represented via provider semantics (if applicable)
- Keep all mutations guarded by typed confirmation for destructive actions (`confirm`)

### TUI Batch DNS Workflows

- Add staged record changes UI (queue multiple creates/updates/deletes)
- Execute via DNSimple zone batch endpoint in one commit-like action
- Preview diff before execution
- Support rollback-friendly JSON export/import for queued changes

### Quality Improvements

- Inline validation for record types/TTL/priority
- Better error surfacing for API validation failures
- Search result highlighting in domain search modal
- Pagination/virtualization if very large lists become slow

## Phase 2: DNS Operations Power Features (CLI + TUI)

Goal: expand beyond basic zones/records into common operational features.

### DNSSEC

- CLI support for enable/disable DNSSEC
- CLI support for DS record management (list/create/get/delete)
- TUI DNSSEC panel in domain dashboard (status + actions)

### Zone NS Management

- Support updating zone NS records (`PUT /zones/{zone}/ns_records`)
- TUI zone management form with confirmation and validation

### Templates

- CLI template CRUD
- CLI template record CRUD
- Apply template to domain
- TUI template browser/apply workflow

### Webhooks

- CLI webhook list/create/get/delete
- TUI webhook inspection and basic CRUD dialogs

## Phase 3: Multi-Account and Team UX

Goal: improve user-token workflows and account context handling.

### Accounts UX

- CLI `accounts list` command
- CLI account selection helper / cache switch command
- TUI account switcher modal (with current account indicator)
- Persist explicit selected account separately from auto-detected fallback

### Context Clarity

- Show active account and environment (production/sandbox/demo) in shell header
- Add clear demo mode badge and sandbox badge in TUI
- Add account-aware filtering/search hints

## Phase 4: Domain Lifecycle / Registrar Features

Goal: broaden from DNS-only to domain operations where users expect DNSimple parity.

### Registrar Essentials (CLI first, TUI follow)

- Availability checks
- Pricing / premium pricing
- Registration
- Transfer-in and transfer authorization/cancel/status
- Renewal status / renew / auto-renew enable/disable
- WHOIS privacy status / enable / disable / renew
- Delegation management (including vanity delegation transitions)

### Domain Services and Email Forwards

- Domain services (list/apply/unapply)
- Email forwards CRUD
- Domain pushes (initiate/list/accept/reject)

## Phase 5: Advanced DNS / Enterprise-Oriented Features

Goal: support broader operational use cases for advanced DNSimple users.

### Secondary DNS

- Primary server CRUD
- Link/unlink primary servers
- Secondary zone creation workflows

### Vanity Name Servers

- Enable/disable vanity nameservers

### Certificates

- Certificate list/get/download
- Private key retrieval (with strong warnings and safe handling)
- Let's Encrypt purchase/issue/renew flows

## Phase 6: Platform / API Coverage Completion

Goal: close the remaining API coverage gaps and expose more of the platform in a maintainable way.

### Remaining Areas (CLI-first)

- Contacts CRUD
- TLD listing/details/extended attributes
- Registrant changes workflows
- Billing charges and analytics endpoints
- OAuth token exchange flow (`/oauth/access_token`) where appropriate
- Domain research status endpoint

### Tooling and DX

- Generate and track an API coverage matrix from the OpenAPI spec
- Add feature flags / capability map for TUI tabs based on backend support (real vs demo)
- Add snapshot tests for TUI views and modal flows

## API Coverage Snapshot (Audit Summary)

As of the February 26, 2026 audit:

- DNSimple OpenAPI operations: `114`
- App coverage (CLI + TUI combined): about `18` operations (~16%)

Currently strongest coverage areas:

- Identity (`whoami`)
- Domains (basic CRUD, minus research status)
- Zones (list/get/file/distribution/activate/deactivate)
- Records (CLI CRUD + distribution, TUI inspect/delete/distribution)

## Suggested Execution Order (Pragmatic)

1. TUI record create/update dialogs
2. Zone batch record changes (CLI first, then TUI staging UI)
3. Accounts list/switch UX
4. DNSSEC + DS records
5. Templates + template records + apply
6. Webhooks
7. Registrar essentials (availability/pricing/renewal/privacy/delegation)

## Notes for Contributors

- New user-visible TUI features should be demo-mode compatible from day one.
- Prefer adding backend interface methods to keep the TUI independent from the transport/client implementation.
- Any new TUI mutation should use a modal dialog and require typed confirmation for destructive actions.
