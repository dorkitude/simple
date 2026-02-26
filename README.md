# simple

A terminal-first DNSimple client with two interfaces:

- A Bubble Tea TUI (`simple` with no subcommand)
- A Cobra CLI (`simple auth`, `simple domains`, `simple zones`, `simple records`, etc.)

The project was originally developed as `dnsimplectl`; the binary name is intentionally flexible (`simple`, `dnsimplectl`, or `dnsimple`).

## Highlights

- TUI auth flow with token validation before saving credentials
- Tabbed TUI shell with keyboard-first navigation
- Global `/` fuzzy domain search modal (jumps to Domains and opens search)
- Full-screen domain dashboard with sections for overview, records, zone, diagnostics, and actions
- Confirm-typed mutation dialogs (`type confirm`) for destructive/state-changing TUI actions
- Demo mode (`simple demo`) using the real TUI against an in-memory fake backend
- CLI coverage for auth, domains, zones, and records (including record create/update/delete)
- JSON output (`--json`) for scripting and automation
- DNSimple sandbox support (`--sandbox`)

## Requirements

- Go `1.24+` (module currently targets `go 1.24.2`)
- DNSimple API token (not needed for `simple demo`)

## Installation

### Build locally

```bash
git clone https://github.com/dorkitude/simple.git
cd simple
go build -o simple .
```

### Install with Go

```bash
go install github.com/dorkitude/simple@latest
```

### Interactive installer

The repo includes `install.sh`, which lets you choose the installed command name (`simple`, `dnsimplectl`, or `dnsimple`) and installs into `GOBIN` / `GOPATH/bin`.

```bash
./install.sh
```

## Quick Start

### TUI (default)

```bash
simple
```

If you are not authenticated, the app starts the TUI auth flow:

1. Welcome screen (shows config/token paths)
2. Optional config directory override (`c`)
3. Masked token entry
4. Token validation via DNSimple `whoami`
5. Save credentials and continue into the TUI shell

### CLI auth flow

```bash
simple auth setup
simple auth login
simple auth status
```

### Demo mode (no auth, fake data)

```bash
simple demo
```

`simple demo` launches the same TUI application using a seeded in-memory backend. It is safe for screenshots, demos, and testing UI interactions without touching your real DNSimple account.

## CLI Guide

### Global flags

These apply to most commands:

- `--json` output JSON instead of styled text
- `--account <id>` override cached DNSimple account ID
- `--sandbox` use DNSimple sandbox API
- `--no-color` disable colored output

### Commands

#### Authentication

```bash
simple auth login
simple auth logout
simple auth status
simple auth setup
```

#### Whoami

```bash
simple whoami
simple whoami --json
```

#### Domains

```bash
simple domains list
simple domains list --filter example
simple domains get example.com
simple domains create example.com
simple domains delete example.com
```

#### Zones

```bash
simple zones list
simple zones get example.com
simple zones file example.com
simple zones distribution example.com
simple zones activate example.com
simple zones deactivate example.com
```

#### Records

```bash
simple records list example.com
simple records list example.com --type A
simple records get example.com 12345

simple records create example.com --type A --name www --content 1.2.3.4
simple records update example.com 12345 --content 5.6.7.8
simple records delete example.com 12345

simple records distribution example.com 12345
```

### JSON output for automation

```bash
simple domains list --json | jq
simple records list example.com --json | jq '.[] | {id, type, name, content}'
```

### Sandbox usage

```bash
simple --sandbox whoami
simple --sandbox domains list
```

## TUI Guide

Running `simple` with no subcommand launches the TUI. `simple --help` still shows CLI help.

### Tabs

- `Home`
- `Domains`
- `Zones`
- `Records`
- `Help`

### Global shortcuts

- `1` / `h` -> Home
- `2` / `d` -> Domains
- `3` / `z` -> Zones
- `4` / `R` -> Records
- `5` / `?` -> Help
- `Tab` / `Shift+Tab` -> next / previous tab
- `/` -> global fuzzy domain search (opens Domains search modal)
- `Ctrl-C` -> always quit immediately (even inside modals)
- `q` -> quit (blocked while a modal is open)

Notes:
- Lowercase `r` is reserved for refresh in resource screens.
- Input/confirm modals hijack keystrokes until dismissed.

### Common navigation

- `j` / `k` or arrow keys -> move selection
- `Enter` -> open / inspect selected item
- `Esc` -> back / close modal / return to previous screen
- `r` -> refresh current list

### Home tab

- Shows account identity (whoami/account plan info)
- Category launcher for Domains / Zones / Records
- `Enter` jumps to the selected tab

### Domains tab

- Lists domains in your account (large lists are windowed to fit terminal height)
- `Enter` opens a full-screen domain dashboard for the selected domain
- `/` opens the fuzzy domain search modal

#### Domain dashboard

The domain dashboard takes over the Domains experience and is designed as the main operational UI for a single domain.

Sections (mnemonic shortcuts use underlined characters in the UI):

- `o` -> Overview
- `c` -> Records
- `z` -> Zone
- `g` -> Diagnostics
- `a` -> Actions

Dashboard shortcuts:

- `R` -> refresh dashboard
- `f` -> fetch zone file (Diagnostics)
- `x` -> check distribution (zone or selected record, context-dependent)
- `D` -> delete selected record (Records section; confirm dialog required)
- `Esc` -> return to Domains list

Records and detail panes wrap long content fields (such as TXT record content) to avoid breaking the TUI layout.

#### Mutation safety (TUI)

All currently exposed TUI mutations use a popup confirmation dialog and require typing:

```text
confirm
```

The dialog explicitly names the mutation target (domain, record, zone/action target) before executing.

Current TUI mutations include:

- Activate zone DNS service
- Deactivate zone DNS service
- Delete selected record
- Delete domain

### Zones tab

- Lists zones
- `Enter` loads zone details
- `f` fetches zone file
- `x` checks zone distribution status

### Records tab

- First shows zones
- `Enter` on a zone loads records for that zone
- `Enter` on a record shows record details
- `x` checks selected record distribution status
- `Esc` returns from record list to zone list

### Help tab

- Built-in shortcut reference
- Config/token/config file path display
- Config directory override guidance (`DNSIMPLE_CONFIG_DIR`)

## Configuration and Credential Storage

By default, credentials are stored under:

```text
~/.config/dnsimplectl/
```

Files:

- `token` (API token)
- `config.json` (cached account ID + settings)

### Config directory override

Set `DNSIMPLE_CONFIG_DIR` to override the storage directory:

```bash
export DNSIMPLE_CONFIG_DIR="$HOME/.local/share/dnsimplectl"
```

Resolution priority:

1. TUI runtime override (set during auth flow when you choose a custom path)
2. `DNSIMPLE_CONFIG_DIR`
3. Default `~/.config/dnsimplectl`

## Demo Mode

`simple demo` is intended for demos, screenshots, and UX iteration. It uses:

- Static seeded fake account/domain/zone/record data
- The same TUI screens and workflows as the real app
- Session-local fake mutations (no network writes)

## Current Scope and Gaps

What is strong today:

- TUI navigation and inspection workflows
- Domain-centric TUI dashboard operations
- Core DNS zone/record/domain CLI workflows
- Safe demo mode for iteration

What is still CLI-only or missing in the TUI:

- Record create/update flows
- Batch record changes
- Some advanced DNSimple features (DNSSEC, templates, webhooks, registrar APIs, etc.)

See `ROADMAP.md` for the implementation roadmap and API coverage priorities.

## Development

### Build

```bash
go build -o simple .
```

### Run

```bash
./simple         # TUI
./simple --help  # CLI help
./simple demo    # demo backend TUI
```

### Format

```bash
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
```

### Contributing

See `CONTRIBUTING.md`.

## License

See `LICENSE`.
