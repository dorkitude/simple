# TUI Mode + Auth Flow for `simple`

## Context

Running `simple` with no args currently shows help text. Chef wants it to launch a TUI instead — and if the user isn't authenticated yet, walk them through a friendly auth flow that shows where credentials are stored and lets them change the path.

## What Changes

### 1. Config path customization (`internal/config/config.go`)
- Add `DNSIMPLE_CONFIG_DIR` env var support
- Add `SetConfigDir(path)` runtime override (for TUI)
- Add `ResolveConfigDir()` to preview the path without creating it
- Priority: `SetConfigDir()` > env var > default `~/.config/dnsimplectl/`

### 2. Root command launches TUI (`cmd/root.go`)
- Add `RunE` on `rootCmd` that calls `tui.Run()`
- Cobra handles `--help` before `RunE` fires, so help still works
- All subcommands (`simple domains list`, etc.) still work normally

### 3. New TUI package (`internal/tui/` — 6 files)

| File | Purpose |
|------|---------|
| `tui.go` | `Run()` entry point, creates `tea.Program` with alt screen |
| `app.go` | `AppModel` — top-level router between auth and home screens |
| `auth.go` | `AuthModel` — multi-step auth flow (welcome → config dir → token → validate → success) |
| `home.go` | `HomeModel` — account info + category menu (Domains, Zones, Records) |
| `styles.go` | TUI-specific lipgloss styles built on `internal/ui` palette |
| `keys.go` | Shared key bindings (quit, back, enter, arrows) |

### Auth flow steps:
1. **Welcome** — shows config path, Enter to continue, `c` to change path
2. **Config Dir** — text input for custom path (only if user pressed `c`)
3. **Token Input** — masked text input, link to where to get a token
4. **Validating** — spinner while calling Whoami API
5. **Success** — account info box, Enter to proceed to home
6. **Error** — error message, Enter to retry

### Home screen (minimal proof-of-concept):
- Account info box (email, ID, plan)
- Category list: Domains, Zones, Records (navigable with j/k/arrows)
- `q` to quit

## Dependencies to add
```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
```

## Implementation order
1. `go get` bubbletea + bubbles
2. Modify `internal/config/config.go` (env var + SetConfigDir + ResolveConfigDir)
3. Create `internal/tui/keys.go`
4. Create `internal/tui/styles.go`
5. Create `internal/tui/auth.go`
6. Create `internal/tui/home.go`
7. Create `internal/tui/app.go`
8. Create `internal/tui/tui.go`
9. Modify `cmd/root.go` (add RunE → tui.Run())
10. `go mod tidy && go build`

## Verification
```bash
go build -o simple .
./simple                    # → launches TUI (auth flow if no token, home if authenticated)
./simple --help             # → still shows CLI help text
./simple help               # → still shows CLI help text
./simple domains list       # → still works as CLI
./simple auth status        # → still works as CLI
```
