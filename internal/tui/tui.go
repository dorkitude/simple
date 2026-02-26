package tui

import tea "github.com/charmbracelet/bubbletea"

// Run launches the interactive TUI.
func Run() error {
	useRealBackend()
	p := tea.NewProgram(NewAppModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// RunDemo launches the TUI in demo mode with a fake in-memory backend.
func RunDemo() error {
	useDemoBackend()
	p := tea.NewProgram(NewDemoAppModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
