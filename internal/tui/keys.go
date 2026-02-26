package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type keyMap struct {
	Quit    key.Binding
	Back    key.Binding
	Enter   key.Binding
	Up      key.Binding
	Down    key.Binding
	TabNext key.Binding
	TabPrev key.Binding
}

var keys = keyMap{
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Back:    key.NewBinding(key.WithKeys("esc", "backspace"), key.WithHelp("esc", "back")),
	Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continue")),
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	TabNext: key.NewBinding(key.WithKeys("tab", "l"), key.WithHelp("tab", "next tab")),
	TabPrev: key.NewBinding(key.WithKeys("shift+tab", "H"), key.WithHelp("shift+tab", "prev tab")),
}

func matches(msg tea.KeyMsg, binding key.Binding) bool {
	return key.Matches(msg, binding)
}
