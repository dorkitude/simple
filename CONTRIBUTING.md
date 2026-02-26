# CONTRIBUTING

Thanks for contributing to `simple`.

## Scope

`simple` is a DNSimple client with:

- A Cobra CLI for scripting and direct operations
- A Bubble Tea TUI for interactive workflows
- A demo backend used by `simple demo` for safe UI iteration

## Prerequisites

- Go `1.24+`
- A DNSimple API token for real-account testing (optional if using `simple demo`)
- A terminal that supports modern ANSI features (for the TUI experience)

## Local Setup

```bash
git clone https://github.com/dorkitude/simple.git
cd simple
go build -o simple .
```

## Run Modes

### TUI (real backend)

```bash
./simple
```

### TUI demo mode (fake backend)

```bash
./simple demo
```

### CLI

```bash
./simple --help
./simple whoami
./simple domains list
```

## Development Workflow

### Format

```bash
gofmt -w $(find . -name '*.go' -not -path './vendor/*')
```

### Build

```bash
go build -o simple .
```

### Smoke checks (recommended before opening a PR)

```bash
./simple --help
./simple demo --help
./simple auth status
```

If you are changing TUI behavior, also run:

```bash
./simple demo
```

and manually verify the affected keyboard flows.

## Project Structure (High-Level)

- `cmd/` - Cobra commands (CLI entrypoints)
- `internal/client/` - DNSimple client construction and account resolution
- `internal/config/` - token/config persistence and config directory resolution
- `internal/output/` - JSON/plain output helpers
- `internal/ui/` - shared CLI styling
- `internal/tui/` - Bubble Tea TUI app, tabs, domain dashboard, demo backend support
- `main.go` - binary entrypoint

## TUI Contribution Guidelines

### Keyboard UX

- Prefer explicit shortcuts and show them in the footer/help text
- Avoid shortcut collisions (especially with global tab navigation and refresh)
- `Ctrl-C` must remain a hard global quit
- Modals should hijack keystrokes until dismissed

### Mutations

- Destructive or state-changing TUI actions must use a modal dialog
- The dialog should clearly name the mutation target (domain/zone/record)
- Keep typed confirmation for destructive operations (`confirm`)

### Layout Safety

- Assume users may have very large domain/record sets
- Avoid unbounded vertical rendering; use windowed lists/viewport-like behavior
- Wrap long record content and other untrusted strings to preserve layout

### Demo Mode Parity

If you add a TUI feature that calls backend data:

1. Add it to the shared backend interface (`internal/tui/data.go`)
2. Implement it in the real backend
3. Implement it in the demo backend with sensible fake behavior

The goal is to keep `simple demo` useful without special-case UI code.

## CLI Contribution Guidelines

- Support `--json` where a command returns structured data
- Keep errors actionable and consistent with existing command behavior
- Prefer additive subcommands/flags over breaking changes

## Commit Style

Use clear, scoped commits. Conventional commit prefixes are preferred:

- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation changes
- `refactor:` internal cleanup with no behavior change
- `chore:` tooling/repo maintenance

## Pull Requests

Please include:

- What changed
- Why it changed
- How you tested it (commands run + any manual TUI flows)
- Screenshots or terminal captures for significant TUI changes (demo mode is ideal for this)

## Release Notes / Roadmap

- Use `README.md` for current user-facing behavior and installation instructions
- Use `ROADMAP.md` for planned features and API coverage priorities
