package tui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type category int

const (
	categoryDomains category = iota
	categoryZones
	categoryRecords
)

func (c category) Label() string {
	switch c {
	case categoryDomains:
		return "Domains"
	case categoryZones:
		return "Zones"
	case categoryRecords:
		return "Records"
	default:
		return "Unknown"
	}
}

type navigateCategoryMsg struct {
	category category
}

type browserBackMsg struct{}

type browserScreen int

const (
	browserDomainsList browserScreen = iota
	browserDomainDashboard
	browserZonesList
	browserRecordsZones
	browserRecordsList
)

type browserItem struct {
	Key      string
	Title    string
	Subtitle string
	ID       int64
}

type browserListLoadedMsg struct {
	screen    browserScreen
	header    string
	items     []browserItem
	statusMsg string
	err       error
}

type browserDetailLoadedMsg struct {
	title string
	body  string
	err   error
}

type domainSearchMatch struct {
	itemIndex int
	score     int
}

type domainSearchModal struct {
	visible  bool
	input    textinput.Model
	selected int
	matches  []domainSearchMatch
}

type BrowserModel struct {
	width       int
	height      int
	category    category
	screen      browserScreen
	selected    int
	items       []browserItem
	spinner     spinner.Model
	loading     bool
	errMsg      string
	listHeader  string
	statusMsg   string
	detailTitle string
	detailBody  string
	recordsZone string
	domainDash  DomainDashboardModel
	search      domainSearchModal
}

func NewBrowserModel(cat category) BrowserModel {
	spin := spinner.New()
	spin.Spinner = spinner.Line
	spin.Style = subtitleStyle

	screen := browserDomainsList
	switch cat {
	case categoryZones:
		screen = browserZonesList
	case categoryRecords:
		screen = browserRecordsZones
	}

	return BrowserModel{
		category:   cat,
		screen:     screen,
		spinner:    spin,
		listHeader: cat.Label(),
		search: domainSearchModal{
			input: newDomainSearchInput(),
		},
	}
}

func (m *BrowserModel) Init() tea.Cmd {
	m.loading = true
	m.errMsg = ""
	return tea.Batch(m.spinner.Tick, m.loadListCmd())
}

func (m *BrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.domainDash.SetSize(width, height)
}

func (m *BrowserModel) Update(msg tea.Msg) tea.Cmd {
	if m.search.visible {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return m.updateSearchModal(keyMsg)
		}
	}

	if m.category == categoryDomains && m.screen == browserDomainDashboard {
		switch msg := msg.(type) {
		case domainDashboardExitMsg:
			m.screen = browserDomainsList
			return nil
		case domainDashboardDeletedMsg:
			m.screen = browserDomainsList
			m.statusMsg = fmt.Sprintf("Domain '%s' deleted.", msg.domain)
			m.detailTitle = ""
			m.detailBody = ""
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadListCmd())
		}
		return m.domainDash.Update(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return nil
		}
		switch {
		case matches(msg, keys.Up):
			if m.selected > 0 {
				m.selected--
			}
			return nil
		case matches(msg, keys.Down):
			if m.selected < len(m.items)-1 {
				m.selected++
			}
			return nil
		case matches(msg, keys.Enter):
			return m.handleEnter()
		case matches(msg, keys.Back):
			if m.category == categoryRecords && m.screen == browserRecordsList {
				m.screen = browserRecordsZones
				m.recordsZone = ""
				m.selected = 0
				m.detailTitle = ""
				m.detailBody = ""
				m.errMsg = ""
				m.loading = true
				return tea.Batch(m.spinner.Tick, m.loadListCmd())
			}
			return func() tea.Msg { return browserBackMsg{} }
		case msg.String() == "r":
			m.detailTitle = ""
			m.detailBody = ""
			m.errMsg = ""
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadListCmd())
		case msg.String() == "/":
			if m.category == categoryDomains && m.screen == browserDomainsList && len(m.items) > 0 {
				m.openSearchModal()
				return textinput.Blink
			}
		case msg.String() == "f":
			if m.category == categoryZones && m.screen == browserZonesList && len(m.items) > 0 {
				m.errMsg = ""
				m.loading = true
				return tea.Batch(m.spinner.Tick, m.loadZoneFileCmd())
			}
		case msg.String() == "x":
			if len(m.items) == 0 {
				return nil
			}
			if m.category == categoryZones && m.screen == browserZonesList {
				m.errMsg = ""
				m.loading = true
				return tea.Batch(m.spinner.Tick, m.loadZoneDistributionCmd())
			}
			if m.category == categoryRecords && m.screen == browserRecordsList {
				m.errMsg = ""
				m.loading = true
				return tea.Batch(m.spinner.Tick, m.loadRecordDistributionCmd())
			}
		}
	case browserListLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.items = nil
			m.listHeader = m.defaultHeader()
			return nil
		}
		m.errMsg = ""
		m.items = msg.items
		m.listHeader = msg.header
		m.statusMsg = msg.statusMsg
		if m.search.visible {
			m.updateSearchMatches()
		}
		if len(m.items) == 0 {
			m.selected = 0
		} else if m.selected >= len(m.items) {
			m.selected = len(m.items) - 1
		}
		return nil
	case browserDetailLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.detailTitle = msg.title
		m.detailBody = msg.body
		return nil
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *BrowserModel) handleEnter() tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}

	if m.category == categoryRecords && m.screen == browserRecordsZones {
		m.recordsZone = m.items[m.selected].Key
		m.screen = browserRecordsList
		m.selected = 0
		m.detailTitle = ""
		m.detailBody = ""
		m.errMsg = ""
		m.loading = true
		return tea.Batch(m.spinner.Tick, m.loadListCmd())
	}

	if m.category == categoryDomains && m.screen == browserDomainsList {
		item := m.items[m.selected]
		m.domainDash = NewDomainDashboardModel(item.Key)
		m.domainDash.SetSize(m.width, m.height)
		m.screen = browserDomainDashboard
		return m.domainDash.Init()
	}

	m.errMsg = ""
	m.loading = true
	return tea.Batch(m.spinner.Tick, m.loadDetailCmd())
}

func (m BrowserModel) View() string {
	if m.category == categoryDomains && m.screen == browserDomainDashboard {
		return m.domainDash.View()
	}

	title := "Browse " + m.category.Label()
	subtitle := "Read-only TUI browser"
	if m.category == categoryRecords && m.screen == browserRecordsZones {
		subtitle = "Choose a zone to inspect records"
	} else if m.category == categoryRecords && m.screen == browserRecordsList {
		subtitle = "Records in " + m.recordsZone
	} else if m.category == categoryZones {
		subtitle = "Use f for zone file and x for distribution status"
	}

	body := []string{
		titleStyle.Render(title),
		subtitleStyle.Render(subtitle),
		"",
		panelStyle.Render(m.listPanel()),
		"",
		panelStyle.Render(m.detailPanel()),
		"",
		footerStyle.Render(m.footerHelp()),
	}
	content := frame(m.width, strings.Join(body, "\n"))
	if m.search.visible {
		content = overlayDialog(content, m.searchModalView())
	}
	return content
}

func (m BrowserModel) listPanel() string {
	lines := []string{panelTitleStyle.Render(m.listHeader)}

	if m.loading {
		lines = append(lines, "", m.spinner.View()+" Loading...")
		return strings.Join(lines, "\n")
	}

	if m.errMsg != "" && len(m.items) == 0 {
		lines = append(lines, "", errorStyle.Render("Failed to load"), m.errMsg)
		return strings.Join(lines, "\n")
	}

	if len(m.items) == 0 {
		lines = append(lines, "", subtitleStyle.Render("No items found."))
		return strings.Join(lines, "\n")
	}

	lines = append(lines, "")
	capacity := m.listLineBudget()
	start, end := windowRange(len(m.items), m.selected, capacity)
	for i := start; i < end; i++ {
		item := m.items[i]
		prefix := "  "
		style := itemStyle
		if i == m.selected {
			prefix = "› "
			style = selectedItemStyle
		}
		row := prefix + item.Title
		if item.Subtitle != "" {
			row += "  |  " + item.Subtitle
		}
		lines = append(lines, style.Render(truncateText(row, m.listRowWidth())))
	}
	lines = append(lines, subtitleStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.items))))

	if m.statusMsg != "" {
		lines = append(lines, "", subtitleStyle.Render(m.statusMsg))
	}
	return clipMultilineText(strings.Join(lines, "\n"), m.panelLineBudget())
}

func (m BrowserModel) detailPanel() string {
	lines := []string{panelTitleStyle.Render("Details")}

	if m.loading && m.detailBody == "" {
		lines = append(lines, "", subtitleStyle.Render("Press Enter on an item to load details."))
		return strings.Join(lines, "\n")
	}

	if m.errMsg != "" && m.detailBody == "" && len(m.items) > 0 {
		lines = append(lines, "", errorStyle.Render(m.errMsg))
		return strings.Join(lines, "\n")
	}

	if m.detailBody == "" {
		lines = append(lines, "", subtitleStyle.Render("Select an item and press Enter."))
		return strings.Join(lines, "\n")
	}

	if m.detailTitle != "" {
		lines = append(lines, "", valueStyle.Render(m.detailTitle))
		lines = append(lines, "")
	}
	lines = append(lines, strings.Split(clipMultilineText(m.detailBody, m.detailLineBudget()), "\n")...)
	return clipMultilineText(strings.Join(lines, "\n"), m.panelLineBudget())
}

func (m BrowserModel) footerHelp() string {
	switch {
	case m.category == categoryRecords && m.screen == browserRecordsZones:
		return "Enter: open zone   /: global domain search   r: refresh   esc: home   q: quit"
	case m.category == categoryRecords && m.screen == browserRecordsList:
		return "Enter: record details   x: distribution   /: global domain search   r: refresh   esc: zones   q: quit"
	case m.category == categoryZones && m.screen == browserZonesList:
		return "Enter: details   f: zone file   x: distribution   /: global domain search   r: refresh   esc: home   q: quit"
	default:
		if m.category == categoryDomains {
			return "Enter: open domain dashboard   /: global search   r: refresh   esc: home   q: quit"
		}
		return "Enter: details   /: global domain search   r: refresh   esc: home   q: quit"
	}
}

func (m BrowserModel) defaultHeader() string {
	if m.category == categoryRecords && m.screen == browserRecordsList && m.recordsZone != "" {
		return "Records / " + m.recordsZone
	}
	if m.category == categoryRecords && m.screen == browserRecordsZones {
		return "Records / Zones"
	}
	return m.category.Label()
}

func (m BrowserModel) loadListCmd() tea.Cmd {
	cat := m.category
	screen := m.screen
	zone := m.recordsZone

	return func() tea.Msg {
		ctx := context.Background()
		backend := getBackend()

		switch {
		case cat == categoryDomains:
			domains, err := backend.ListDomains(ctx)
			if err != nil {
				return browserListLoadedMsg{screen: screen, err: err}
			}
			items := make([]browserItem, 0, len(domains))
			for _, d := range domains {
				meta := "state: " + d.State
				if d.ExpiresAt != "" {
					meta += " | expires: " + truncateText(d.ExpiresAt, 10)
				}
				if d.AutoRenew {
					meta += " | auto-renew"
				}
				items = append(items, browserItem{
					Key:      d.Name,
					Title:    d.Name,
					Subtitle: meta,
					ID:       d.ID,
				})
			}
			return browserListLoadedMsg{
				screen:    screen,
				header:    fmt.Sprintf("Domains (%d)", len(items)),
				items:     items,
				statusMsg: "Use Enter to inspect a domain.",
			}

		case cat == categoryZones && screen == browserZonesList:
			zones, err := backend.ListZones(ctx)
			if err != nil {
				return browserListLoadedMsg{screen: screen, err: err}
			}
			items := make([]browserItem, 0, len(zones))
			for _, z := range zones {
				flags := []string{}
				if z.Active {
					flags = append(flags, "active")
				} else {
					flags = append(flags, "inactive")
				}
				if z.Reverse {
					flags = append(flags, "reverse")
				}
				if z.Secondary {
					flags = append(flags, "secondary")
				}
				items = append(items, browserItem{
					Key:      z.Name,
					Title:    z.Name,
					Subtitle: strings.Join(flags, " | "),
					ID:       z.ID,
				})
			}
			return browserListLoadedMsg{
				screen:    screen,
				header:    fmt.Sprintf("Zones (%d)", len(items)),
				items:     items,
				statusMsg: "Use Enter to inspect a zone.",
			}

		case cat == categoryRecords && screen == browserRecordsZones:
			zones, err := backend.ListZones(ctx)
			if err != nil {
				return browserListLoadedMsg{screen: screen, err: err}
			}
			items := make([]browserItem, 0, len(zones))
			for _, z := range zones {
				flags := []string{}
				if z.Active {
					flags = append(flags, "active")
				}
				if z.Reverse {
					flags = append(flags, "reverse")
				}
				if z.Secondary {
					flags = append(flags, "secondary")
				}
				if len(flags) == 0 {
					flags = append(flags, "standard")
				}
				items = append(items, browserItem{
					Key:      z.Name,
					Title:    z.Name,
					Subtitle: strings.Join(flags, " | "),
					ID:       z.ID,
				})
			}
			return browserListLoadedMsg{
				screen:    screen,
				header:    fmt.Sprintf("Records / Zones (%d)", len(items)),
				items:     items,
				statusMsg: "Choose a zone, then press Enter to list its records.",
			}

		case cat == categoryRecords && screen == browserRecordsList:
			records, err := backend.ListRecords(ctx, zone)
			if err != nil {
				return browserListLoadedMsg{screen: screen, err: err}
			}
			items := make([]browserItem, 0, len(records))
			for _, r := range records {
				name := r.Name
				if name == "" {
					name = "@"
				}
				title := fmt.Sprintf("%-6s %s", r.Type, name)
				meta := fmt.Sprintf("ttl %d | %s", r.TTL, truncateText(r.Content, 72))
				if r.Priority != 0 {
					meta += fmt.Sprintf(" | pri %d", r.Priority)
				}
				if r.SystemRecord {
					meta += " | system"
				}
				items = append(items, browserItem{
					Key:      strconv.FormatInt(r.ID, 10),
					Title:    title,
					Subtitle: meta,
					ID:       r.ID,
				})
			}
			return browserListLoadedMsg{
				screen:    screen,
				header:    fmt.Sprintf("Records / %s (%d)", zone, len(items)),
				items:     items,
				statusMsg: "Use Enter to inspect a record. Esc returns to zones.",
			}
		}

		return browserListLoadedMsg{screen: screen, err: fmt.Errorf("unsupported browser state")}
	}
}

func (m BrowserModel) loadDetailCmd() tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}
	item := m.items[m.selected]
	cat := m.category
	screen := m.screen
	zone := m.recordsZone

	return func() tea.Msg {
		ctx := context.Background()
		backend := getBackend()

		switch {
		case cat == categoryDomains:
			d, err := backend.GetDomain(ctx, item.Key)
			if err != nil {
				return browserDetailLoadedMsg{err: err}
			}
			lines := []string{
				"ID: " + strconv.FormatInt(d.ID, 10),
				"Name: " + d.Name,
			}
			if d.UnicodeName != "" && d.UnicodeName != d.Name {
				lines = append(lines, "Unicode: "+d.UnicodeName)
			}
			lines = append(lines,
				"State: "+d.State,
				"Auto-Renew: "+strconv.FormatBool(d.AutoRenew),
				"Private WHOIS: "+strconv.FormatBool(d.PrivateWhois),
			)
			if d.ExpiresAt != "" {
				lines = append(lines, "Expires: "+d.ExpiresAt)
			}
			lines = append(lines, "Created: "+d.CreatedAt, "Updated: "+d.UpdatedAt)
			return browserDetailLoadedMsg{title: d.Name, body: strings.Join(lines, "\n")}

		case cat == categoryZones && screen == browserZonesList:
			z, err := backend.GetZone(ctx, item.Key)
			if err != nil {
				return browserDetailLoadedMsg{err: err}
			}
			lines := []string{
				"ID: " + strconv.FormatInt(z.ID, 10),
				"Name: " + z.Name,
				"Active: " + strconv.FormatBool(z.Active),
				"Reverse: " + strconv.FormatBool(z.Reverse),
				"Secondary: " + strconv.FormatBool(z.Secondary),
				"Created: " + z.CreatedAt,
				"Updated: " + z.UpdatedAt,
			}
			return browserDetailLoadedMsg{title: z.Name, body: strings.Join(lines, "\n")}

		case cat == categoryRecords && screen == browserRecordsList:
			r, err := backend.GetRecord(ctx, zone, item.ID)
			if err != nil {
				return browserDetailLoadedMsg{err: err}
			}
			name := r.Name
			if name == "" {
				name = "@"
			}
			lines := []string{
				"ID: " + strconv.FormatInt(r.ID, 10),
				"Type: " + r.Type,
				"Name: " + name,
				"Zone: " + zone,
			}
			lines = append(lines, wrapLabelValue("Content: ", r.Content, 80)...)
			lines = append(lines, "TTL: "+strconv.Itoa(r.TTL))
			if r.Priority != 0 {
				lines = append(lines, "Priority: "+strconv.Itoa(r.Priority))
			}
			if len(r.Regions) > 0 {
				lines = append(lines, "Regions: "+strings.Join(r.Regions, ", "))
			}
			lines = append(lines,
				"System: "+strconv.FormatBool(r.SystemRecord),
				"Created: "+r.CreatedAt,
				"Updated: "+r.UpdatedAt,
			)
			return browserDetailLoadedMsg{
				title: fmt.Sprintf("%s %s.%s", r.Type, name, zone),
				body:  strings.Join(lines, "\n"),
			}
		}

		return browserDetailLoadedMsg{err: fmt.Errorf("unsupported detail view")}
	}
}

func (m BrowserModel) loadZoneFileCmd() tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}
	item := m.items[m.selected]
	return func() tea.Msg {
		zoneFile, err := getBackend().GetZoneFile(context.Background(), item.Key)
		if err != nil {
			return browserDetailLoadedMsg{err: err}
		}
		return browserDetailLoadedMsg{
			title: "Zone file: " + item.Key,
			body:  zoneFile,
		}
	}
}

func (m BrowserModel) loadZoneDistributionCmd() tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}
	item := m.items[m.selected]
	return func() tea.Msg {
		distributed, err := getBackend().CheckZoneDistribution(context.Background(), item.Key)
		if err != nil {
			return browserDetailLoadedMsg{err: err}
		}
		status := "Not distributed yet"
		if distributed {
			status = "Fully distributed"
		}
		return browserDetailLoadedMsg{
			title: "Zone distribution: " + item.Key,
			body:  "Distributed: " + strconv.FormatBool(distributed) + "\nStatus: " + status,
		}
	}
}

func (m BrowserModel) loadRecordDistributionCmd() tea.Cmd {
	if len(m.items) == 0 || m.recordsZone == "" {
		return nil
	}
	item := m.items[m.selected]
	return func() tea.Msg {
		distributed, err := getBackend().CheckRecordDistribution(context.Background(), m.recordsZone, item.ID)
		if err != nil {
			return browserDetailLoadedMsg{err: err}
		}
		status := "Not distributed yet"
		if distributed {
			status = "Fully distributed"
		}
		return browserDetailLoadedMsg{
			title: fmt.Sprintf("Record distribution: %d (%s)", item.ID, m.recordsZone),
			body:  "Distributed: " + strconv.FormatBool(distributed) + "\nStatus: " + status,
		}
	}
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func (m BrowserModel) listLineBudget() int {
	if m.height <= 0 {
		return 10
	}
	return maxInt(5, (m.height-18)/2)
}

func (m BrowserModel) detailLineBudget() int {
	if m.height <= 0 {
		return 10
	}
	return maxInt(6, (m.height-18)/2)
}

func (m BrowserModel) panelLineBudget() int {
	if m.height <= 0 {
		return 18
	}
	return maxInt(8, m.height-14)
}

func (m BrowserModel) listRowWidth() int {
	if m.width <= 0 {
		return 100
	}
	return maxInt(30, m.width-14)
}

func (m *BrowserModel) BlocksGlobalKeys() bool {
	if m.search.visible {
		return true
	}
	return m.category == categoryDomains &&
		m.screen == browserDomainDashboard &&
		m.domainDash.ModalVisible()
}

func (m *BrowserModel) OpenSearch() tea.Cmd {
	if m.category != categoryDomains {
		return nil
	}
	m.screen = browserDomainsList
	m.detailTitle = ""
	m.detailBody = ""
	m.openSearchModal()
	return textinput.Blink
}

func newDomainSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "Search domains (fuzzy)"
	ti.CharLimit = 200
	ti.Width = 48
	return ti
}

func (m *BrowserModel) openSearchModal() {
	m.search.visible = true
	m.search.selected = 0
	m.search.input.SetValue("")
	m.search.input.Focus()
	m.updateSearchMatches()
}

func (m *BrowserModel) closeSearchModal() {
	m.search.visible = false
	m.search.selected = 0
	m.search.matches = nil
	m.search.input.Blur()
}

func (m *BrowserModel) updateSearchModal(msg tea.KeyMsg) tea.Cmd {
	switch {
	case msg.String() == "/":
		return nil
	case msg.String() == "esc":
		m.closeSearchModal()
		return nil
	case msg.String() == "up":
		if m.search.selected > 0 {
			m.search.selected--
		}
		return nil
	case msg.String() == "down":
		if m.search.selected < len(m.search.matches)-1 {
			m.search.selected++
		}
		return nil
	case matches(msg, keys.Enter):
		if len(m.search.matches) == 0 {
			return nil
		}
		match := m.search.matches[m.search.selected]
		m.selected = match.itemIndex
		m.closeSearchModal()
		if m.category == categoryDomains && m.screen == browserDomainsList && m.selected >= 0 && m.selected < len(m.items) {
			item := m.items[m.selected]
			m.domainDash = NewDomainDashboardModel(item.Key)
			m.domainDash.SetSize(m.width, m.height)
			m.screen = browserDomainDashboard
			return m.domainDash.Init()
		}
		return nil
	}

	var cmd tea.Cmd
	m.search.input, cmd = m.search.input.Update(msg)
	m.updateSearchMatches()
	return cmd
}

func (m *BrowserModel) updateSearchMatches() {
	if m.category != categoryDomains || m.screen != browserDomainsList {
		m.search.matches = nil
		m.search.selected = 0
		return
	}

	query := strings.TrimSpace(m.search.input.Value())
	matches := make([]domainSearchMatch, 0, len(m.items))
	for i, item := range m.items {
		score, ok := fuzzyScore(query, item.Title)
		if !ok {
			continue
		}
		matches = append(matches, domainSearchMatch{
			itemIndex: i,
			score:     score,
		})
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].score == matches[j].score {
			a := strings.ToLower(m.items[matches[i].itemIndex].Title)
			b := strings.ToLower(m.items[matches[j].itemIndex].Title)
			return a < b
		}
		return matches[i].score > matches[j].score
	})

	m.search.matches = matches
	if len(m.search.matches) == 0 {
		m.search.selected = 0
		return
	}
	if m.search.selected >= len(m.search.matches) {
		m.search.selected = len(m.search.matches) - 1
	}
	if m.search.selected < 0 {
		m.search.selected = 0
	}
}

func (m BrowserModel) searchModalView() string {
	lines := []string{
		panelTitleStyle.Render("Search Domains"),
		"",
		subtitleStyle.Render("Fuzzy match on domain names. Enter opens the selected domain dashboard."),
		"",
		m.search.input.View(),
		"",
	}

	if len(m.search.matches) == 0 {
		lines = append(lines, subtitleStyle.Render("No matches"))
	} else {
		maxResults := maxInt(5, minInt(12, m.height-14))
		start, end := windowRange(len(m.search.matches), m.search.selected, maxResults)
		for i := start; i < end; i++ {
			match := m.search.matches[i]
			item := m.items[match.itemIndex]
			prefix := "  "
			style := itemStyle
			if i == m.search.selected {
				prefix = "› "
				style = selectedItemStyle
			}
			row := fmt.Sprintf("%s%s", prefix, item.Title)
			lines = append(lines, style.Render(truncateText(row, 80)))
			if item.Subtitle != "" {
				lines = append(lines, subtitleStyle.Render("   "+truncateText(item.Subtitle, 80)))
			}
		}
		lines = append(lines, "", subtitleStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.search.matches))))
	}

	lines = append(lines, "", footerStyle.Render("Type to search   enter: open   esc: cancel   j/k: move"))
	box := modalPanelStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(maxInt(70, m.width), maxInt(20, m.height), lipgloss.Center, lipgloss.Center, box)
}

func fuzzyScore(query, target string) (int, bool) {
	q := strings.ToLower(strings.TrimSpace(query))
	t := strings.ToLower(target)
	if q == "" {
		// Empty query returns all domains, stable-sorted alphabetically after equal score.
		return 0, true
	}

	qr := []rune(q)
	tr := []rune(t)
	ti := 0
	prev := -1
	score := 0

	for _, qc := range qr {
		found := -1
		for ti < len(tr) {
			if tr[ti] == qc {
				found = ti
				break
			}
			ti++
		}
		if found == -1 {
			return 0, false
		}

		score += 10
		if found == 0 {
			score += 12
		} else if found > 0 {
			switch tr[found-1] {
			case '.', '-', '_':
				score += 8
			}
		}
		if prev >= 0 {
			gap := found - prev - 1
			if gap == 0 {
				score += 10
			} else {
				score -= minInt(gap, 6)
			}
		}

		prev = found
		ti = found + 1
	}

	// Small preference for shorter labels when the match quality is similar.
	score -= minInt(len(tr)/8, 6)
	return score, true
}
