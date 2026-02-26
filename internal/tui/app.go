package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/config"
)

type appRoute int

const (
	routeAuth appRoute = iota
	routeShell
)

type AppModel struct {
	width  int
	height int
	route  appRoute
	auth   AuthModel
	shell  ShellModel
}

func NewAppModel() *AppModel {
	m := &AppModel{}
	if config.HasToken() {
		m.route = routeShell
		m.shell = NewShellModel(nil)
	} else {
		m.route = routeAuth
		m.auth = NewAuthModel()
	}
	return m
}

func NewDemoAppModel() *AppModel {
	var whoami *dnsimple.WhoamiData
	if data, err := getBackend().Whoami(context.Background()); err == nil {
		whoami = data
	}
	return &AppModel{
		route: routeShell,
		shell: NewShellModel(whoami),
	}
}

func (m *AppModel) Init() tea.Cmd {
	switch m.route {
	case routeShell:
		return m.shell.Init()
	default:
		return m.auth.Init()
	}
}

func (m *AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.auth.SetSize(m.width, m.height)
		m.shell.SetSize(m.width, m.height)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if matches(msg, keys.Quit) && !(m.route == routeShell && m.shell.BlocksGlobalKeys()) {
			return m, tea.Quit
		}
	case authDoneMsg:
		m.route = routeShell
		m.shell = NewShellModel(msg.data)
		m.shell.SetSize(m.width, m.height)
		return m, m.shell.Init()
	}

	switch m.route {
	case routeShell:
		return m, m.shell.Update(msg)
	default:
		return m, m.auth.Update(msg)
	}
}

func (m *AppModel) View() string {
	switch m.route {
	case routeShell:
		return m.shell.View()
	default:
		return m.auth.View()
	}
}
