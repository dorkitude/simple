package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/dorkitude/simple/internal/ui"
)

var (
	docStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.Primary)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(ui.Subtle)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.Primary).
			Padding(1, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ui.Primary)

	labelStyle = lipgloss.NewStyle().
			Foreground(ui.Subtle)

	valueStyle = lipgloss.NewStyle().
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(ui.Secondary).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(ui.Error).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(ui.Warning).
			Bold(true)

	linkStyle = lipgloss.NewStyle().
			Foreground(ui.Accent).
			Underline(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(ui.Primary).
				Bold(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	footerStyle = lipgloss.NewStyle().
			Foreground(ui.Subtle)

	codeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E5E7EB")).
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	tabBarStyle = lipgloss.NewStyle().
			Padding(0, 2)

	tabStyle = lipgloss.NewStyle().
			Foreground(ui.Subtle).
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ui.Primary).
			Bold(true).
			Padding(0, 1)

	tabHintStyle = lipgloss.NewStyle().
			Foreground(ui.Subtle)

	modalPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(ui.Warning).
			Background(lipgloss.Color("#111827")).
			Padding(1, 2).
			MaxWidth(96)
)

func frame(width int, content string) string {
	if width > 0 {
		return docStyle.Width(width).Render(content)
	}
	return docStyle.Render(content)
}

func overlayDialog(base, dialog string) string {
	if dialog == "" {
		return base
	}
	return dialog
}
