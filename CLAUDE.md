# CLAUDE.md

This file provides guidance to Claude Code when working with simple.

## Project Overview

simple is a CLI + TUI for the DNSimple API v2, built in Go with lipgloss for styled terminal output. Modeled after gooctl (the GSuite CLI in the same repo).

## Build & Install

```bash
./install.sh                 # Interactive install — prompts for binary name (default: simple)
go build -o simple .         # Build manually
go run .                     # Run directly
```

The default binary name is `simple`. The install script lets users choose between `simple`, `dnsimplectl`, and `dnsimple`. The CLI dynamically adapts its help text to match whatever binary name is used.

## Project Structure

- `main.go` - Entry point
- `cmd/` - Cobra CLI commands (root, auth, whoami, domains, zones, records, etc.)
- `internal/client/` - DNSimple SDK wrapper (App struct with client + account ID)
- `internal/config/` - Token + config storage in ~/.config/dnsimplectl/
- `internal/ui/` - Lipgloss DNS-themed color palette and styled helpers
- `internal/output/` - JSON vs styled output dispatcher

## Key Dependencies

- [cobra](https://github.com/spf13/cobra) - CLI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [dnsimple-go](https://github.com/dnsimple/dnsimple-go) - Official DNSimple SDK

## Config

API token stored in `~/.config/dnsimplectl/token` (0600 perms).
Account config in `~/.config/dnsimplectl/config.json`.

## Auth

DNSimple uses API tokens (not OAuth). `auth login` prompts for token, validates via Whoami endpoint, stores locally. Much simpler than gooctl's OAuth flow.
