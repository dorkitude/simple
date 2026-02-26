package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dnsimple/dnsimple-go/dnsimple"
)

type shellTab int

const (
	tabHome shellTab = iota
	tabDomains
	tabZones
	tabRecords
	tabHelp
)

type shellTabDef struct {
	id       shellTab
	label    string
	shortcut string
}

var shellTabs = []shellTabDef{
	{id: tabHome, label: "Home", shortcut: "1/h"},
	{id: tabDomains, label: "Domains", shortcut: "2/d"},
	{id: tabZones, label: "Zones", shortcut: "3/z"},
	{id: tabRecords, label: "Records", shortcut: "4/R"},
	{id: tabHelp, label: "Help", shortcut: "5/?"},
}

type tabContent interface {
	Init() tea.Cmd
	Update(tea.Msg) tea.Cmd
	View() string
	SetSize(width, height int)
}

type ShellModel struct {
	width       int
	height      int
	active      shellTab
	initialized map[shellTab]bool
	home        HomeModel
	domains     BrowserModel
	zones       BrowserModel
	records     BrowserModel
	help        HelpModel
}

func NewShellModel(whoami *dnsimple.WhoamiData) ShellModel {
	return ShellModel{
		active:      tabHome,
		initialized: map[shellTab]bool{},
		home:        NewHomeModel(whoami),
		domains:     NewBrowserModel(categoryDomains),
		zones:       NewBrowserModel(categoryZones),
		records:     NewBrowserModel(categoryRecords),
		help:        NewHelpModel(),
	}
}

func (m *ShellModel) Init() tea.Cmd {
	return m.initTab(tabHome)
}

func (m *ShellModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.home.SetSize(width, height)
	m.domains.SetSize(width, height)
	m.zones.SetSize(width, height)
	m.records.SetSize(width, height)
	m.help.SetSize(width, height)
}

func (m *ShellModel) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.BlocksGlobalKeys() {
		if cmd := m.handleGlobalKeys(keyMsg); cmd != nil {
			return cmd
		}
	}

	switch msg := msg.(type) {
	case navigateCategoryMsg:
		switch msg.category {
		case categoryDomains:
			return m.activate(tabDomains)
		case categoryZones:
			return m.activate(tabZones)
		case categoryRecords:
			return m.activate(tabRecords)
		}
	case browserBackMsg:
		return m.activate(tabHome)
	}

	return m.activeModel().Update(msg)
}

func (m *ShellModel) View() string {
	parts := []string{
		titleStyle.Render("Simple - a TUI for DNSimple.com"),
		"",
		tabBarStyle.Render(m.tabBar()),
		"",
		m.activeModel().View(),
	}
	return strings.Join(parts, "\n")
}

func (m *ShellModel) handleGlobalKeys(msg tea.KeyMsg) tea.Cmd {
	switch {
	case matches(msg, keys.TabNext):
		return m.activate(m.nextTab())
	case matches(msg, keys.TabPrev):
		return m.activate(m.prevTab())
	}

	switch msg.String() {
	case "/":
		return m.openGlobalDomainSearch()
	case "1", "h":
		return m.activate(tabHome)
	case "2", "d":
		return m.activate(tabDomains)
	case "3", "z":
		return m.activate(tabZones)
	case "4", "R":
		return m.activate(tabRecords)
	case "5", "?":
		return m.activate(tabHelp)
	}

	return nil
}

func (m *ShellModel) openGlobalDomainSearch() tea.Cmd {
	m.active = tabDomains
	return tea.Batch(m.initTab(tabDomains), m.domains.OpenSearch())
}

func (m *ShellModel) nextTab() shellTab {
	return shellTab((int(m.active) + 1) % len(shellTabs))
}

func (m *ShellModel) prevTab() shellTab {
	n := len(shellTabs)
	return shellTab((int(m.active) - 1 + n) % n)
}

func (m *ShellModel) activate(tab shellTab) tea.Cmd {
	m.active = tab
	return m.initTab(tab)
}

func (m *ShellModel) initTab(tab shellTab) tea.Cmd {
	if m.initialized == nil {
		m.initialized = map[shellTab]bool{}
	}
	if m.initialized[tab] {
		return nil
	}
	m.initialized[tab] = true
	return m.modelFor(tab).Init()
}

func (m *ShellModel) activeModel() tabContent {
	return m.modelFor(m.active)
}

func (m *ShellModel) modelFor(tab shellTab) tabContent {
	switch tab {
	case tabDomains:
		return &m.domains
	case tabZones:
		return &m.zones
	case tabRecords:
		return &m.records
	case tabHelp:
		return &m.help
	default:
		return &m.home
	}
}

func (m *ShellModel) tabBar() string {
	rendered := make([]string, 0, len(shellTabs))
	for _, t := range shellTabs {
		label := t.label + " " + tabHintStyle.Render("["+t.shortcut+"]")
		if t.id == m.active {
			rendered = append(rendered, activeTabStyle.Render(label))
			continue
		}
		rendered = append(rendered, tabStyle.Render(label))
	}
	return strings.Join(rendered, " ")
}

func (m *ShellModel) BlocksGlobalKeys() bool {
	if blocker, ok := m.activeModel().(interface{ BlocksGlobalKeys() bool }); ok {
		return blocker.BlocksGlobalKeys()
	}
	return false
}
