package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dorkitude/simple/internal/config"
)

type HelpModel struct {
	width  int
	height int
}

func NewHelpModel() HelpModel {
	return HelpModel{}
}

func (m *HelpModel) Init() tea.Cmd { return nil }

func (m *HelpModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *HelpModel) Update(msg tea.Msg) tea.Cmd { return nil }

func (m HelpModel) View() string {
	cfgDir, _ := config.ResolveConfigDir()
	tokenPath, _ := config.TokenPath()
	configPath, _ := config.ConfigPath()

	left := panelStyle.Render(strings.Join([]string{
		panelTitleStyle.Render("Global Shortcuts"),
		"",
		"1 / h   Home",
		"2 / d   Domains",
		"3 / z   Zones",
		"4 / R   Records",
		"5 / ?   Help",
		"",
		"tab / shift+tab   Next/Prev tab",
		"/                 Domain search (global; jumps to Domains)",
		"j/k or arrows     Move selection",
		"enter             Open / inspect selected item",
		"esc               Back (or return home)",
		"r                 Refresh current list",
		"q                 Quit",
	}, "\n"))

	right := panelStyle.Render(strings.Join([]string{
		panelTitleStyle.Render("Storage Paths"),
		"",
		"Config dir:",
		codeStyle.Render(cfgDir),
		"",
		"Token file:",
		codeStyle.Render(tokenPath),
		"",
		"Config file:",
		codeStyle.Render(configPath),
		"",
		subtitleStyle.Render("Override config dir with DNSIMPLE_CONFIG_DIR"),
	}, "\n"))

	body := []string{
		titleStyle.Render("Help"),
		subtitleStyle.Render("TUI navigation and configuration reference"),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		"",
		panelStyle.Render(strings.Join([]string{
			panelTitleStyle.Render("Tab-Specific Actions"),
			"",
			"Zones tab: f (zone file), x (distribution status)",
			"Records tab: enter on a zone first, then x (record distribution status)",
			"",
			subtitleStyle.Render("Mutating operations (create/update/delete) remain available via CLI commands."),
		}, "\n")),
		"",
		footerStyle.Render("Use tab shortcuts to jump anywhere instantly. / opens global domain search."),
	}
	return frame(m.width, strings.Join(body, "\n"))
}
