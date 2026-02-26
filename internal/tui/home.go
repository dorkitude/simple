package tui

import (
	"context"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dnsimple/dnsimple-go/dnsimple"
)

type homeWhoamiMsg struct {
	data *dnsimple.WhoamiData
	err  error
}

type HomeModel struct {
	width    int
	height   int
	items    []category
	selected int
	whoami   *dnsimple.WhoamiData
	loading  bool
	errMsg   string
	spinner  spinner.Model
}

func NewHomeModel(whoami *dnsimple.WhoamiData) HomeModel {
	spin := spinner.New()
	spin.Spinner = spinner.Line
	spin.Style = subtitleStyle

	return HomeModel{
		items:   []category{categoryDomains, categoryZones, categoryRecords},
		whoami:  whoami,
		loading: whoami == nil,
		spinner: spin,
	}
}

func (m *HomeModel) Init() tea.Cmd {
	if m.loading {
		return tea.Batch(m.spinner.Tick, fetchWhoamiCmd())
	}
	return nil
}

func (m *HomeModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *HomeModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case matches(msg, keys.Up):
			if m.selected > 0 {
				m.selected--
			}
		case matches(msg, keys.Down):
			if m.selected < len(m.items)-1 {
				m.selected++
			}
		case matches(msg, keys.Enter):
			if len(m.items) > 0 {
				return func() tea.Msg { return navigateCategoryMsg{category: m.items[m.selected]} }
			}
		}
	case homeWhoamiMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.whoami = msg.data
		m.errMsg = ""
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m HomeModel) View() string {
	body := []string{
		titleStyle.Render("DNSimple Home"),
		subtitleStyle.Render("Browse your DNSimple resources or jump directly with tab shortcuts"),
		"",
		panelStyle.Render(m.accountPanel()),
		"",
		panelStyle.Render(m.menuPanel()),
		"",
		footerStyle.Render("j/k or arrows: navigate   enter: open tab   /: global domain search   1-5/hdzR?: tabs   q: quit"),
	}
	return frame(m.width, strings.Join(body, "\n"))
}

func (m HomeModel) accountPanel() string {
	lines := []string{panelTitleStyle.Render("Account")}
	if m.loading {
		lines = append(lines, "", m.spinner.View()+" Loading account info...")
		return strings.Join(lines, "\n")
	}
	if m.errMsg != "" {
		lines = append(lines, "", errorStyle.Render("Failed to load account info"), m.errMsg)
		return strings.Join(lines, "\n")
	}

	if m.whoami == nil {
		lines = append(lines, "", subtitleStyle.Render("No account info available."))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "")
	if m.whoami.Account != nil {
		a := m.whoami.Account
		lines = append(lines,
			labelStyle.Render("Type: ")+"Account token",
			labelStyle.Render("Email: ")+a.Email,
			labelStyle.Render("ID: ")+strconv.FormatInt(a.ID, 10),
			labelStyle.Render("Plan: ")+a.PlanIdentifier,
		)
	}
	if m.whoami.User != nil {
		u := m.whoami.User
		lines = append(lines,
			labelStyle.Render("User: ")+u.Email,
			labelStyle.Render("User ID: ")+strconv.FormatInt(u.ID, 10),
		)
	}
	return strings.Join(lines, "\n")
}

func (m HomeModel) menuPanel() string {
	lines := []string{panelTitleStyle.Render("Categories"), ""}
	for i, item := range m.items {
		prefix := "  "
		style := itemStyle
		if i == m.selected {
			prefix = "› "
			style = selectedItemStyle
		}
		lines = append(lines, style.Render(prefix+item.Label()))
	}
	lines = append(lines, "", subtitleStyle.Render("Press Enter to switch to the selected tab (or use 2/3/4)."))
	return strings.Join(lines, "\n")
}

func fetchWhoamiCmd() tea.Cmd {
	return func() tea.Msg {
		data, err := getBackend().Whoami(context.Background())
		return homeWhoamiMsg{data: data, err: err}
	}
}
