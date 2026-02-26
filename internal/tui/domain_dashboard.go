package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dnsimple/dnsimple-go/dnsimple"
)

type domainDashSection int

const (
	domainSectionOverview domainDashSection = iota
	domainSectionRecords
	domainSectionZone
	domainSectionDiagnostics
	domainSectionActions
)

type domainDashboardExitMsg struct{}

type domainDashboardDeletedMsg struct {
	domain string
}

type domainDashboardLoadedMsg struct {
	domain   *dnsimple.Domain
	zone     *dnsimple.Zone
	records  []dnsimple.ZoneRecord
	warnings []string
	err      error
}

type domainDashboardRecordDetailMsg struct {
	record *dnsimple.ZoneRecord
	err    error
}

type domainDashboardZoneFileMsg struct {
	zoneFile string
	err      error
}

type domainDashboardZoneDistributionMsg struct {
	distributed bool
	err         error
}

type domainDashboardRecordDistributionMsg struct {
	recordID    int64
	distributed bool
	err         error
}

type domainDashboardMutationMsg struct {
	kind   string
	domain string
	status string
	reload bool
	exit   bool
	err    error
}

type confirmMutation int

const (
	mutationNone confirmMutation = iota
	mutationZoneActivate
	mutationZoneDeactivate
	mutationDeleteRecord
	mutationDeleteDomain
)

type confirmModal struct {
	visible bool
	busy    bool
	action  confirmMutation
	title   string
	body    string
	input   textinput.Model
	errMsg  string
}

type DomainDashboardModel struct {
	width    int
	height   int
	domain   string
	section  domainDashSection
	loading  bool
	spinner  spinner.Model
	errMsg   string
	status   string
	warnings []string

	dataDomain *dnsimple.Domain
	dataZone   *dnsimple.Zone
	records    []dnsimple.ZoneRecord

	selectedRecord int
	recordDetail   *dnsimple.ZoneRecord

	diagTitle string
	diagBody  string

	selectedAction int
	modal          confirmModal
}

func NewDomainDashboardModel(domain string) DomainDashboardModel {
	spin := spinner.New()
	spin.Spinner = spinner.Line
	spin.Style = subtitleStyle

	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = "type confirm"
	ti.CharLimit = 64
	ti.Width = 24

	return DomainDashboardModel{
		domain:  domain,
		spinner: spin,
		modal: confirmModal{
			input: ti,
		},
	}
}

func (m *DomainDashboardModel) Init() tea.Cmd {
	m.loading = true
	m.errMsg = ""
	m.status = ""
	return tea.Batch(m.spinner.Tick, m.loadDashboardCmd())
}

func (m *DomainDashboardModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *DomainDashboardModel) Update(msg tea.Msg) tea.Cmd {
	if m.modal.visible {
		switch msg.(type) {
		case tea.KeyMsg, spinner.TickMsg:
			return m.updateModal(msg)
		}
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return nil
		}

		switch msg.String() {
		case "o", "O":
			m.section = domainSectionOverview
			return nil
		case "c", "C":
			m.section = domainSectionRecords
			return nil
		case "z", "Z":
			m.section = domainSectionZone
			return nil
		case "g", "G":
			m.section = domainSectionDiagnostics
			return nil
		case "a", "A":
			m.section = domainSectionActions
			return nil
		case "R":
			m.loading = true
			m.errMsg = ""
			return tea.Batch(m.spinner.Tick, m.loadDashboardCmd())
		case "f":
			m.section = domainSectionDiagnostics
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadZoneFileCmd())
		case "x":
			if m.section == domainSectionRecords {
				if rec := m.selectedRecordPtr(); rec != nil {
					m.loading = true
					return tea.Batch(m.spinner.Tick, m.loadRecordDistributionCmd(rec.ID))
				}
				return nil
			}
			m.section = domainSectionDiagnostics
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadZoneDistributionCmd())
		case "D":
			if m.section == domainSectionRecords && m.selectedRecordPtr() != nil {
				m.openConfirm(mutationDeleteRecord, "Delete Record", "This will permanently delete the selected record from the zone.")
				return textinput.Blink
			}
		}

		switch {
		case matches(msg, keys.Back):
			return func() tea.Msg { return domainDashboardExitMsg{} }
		case matches(msg, keys.Up):
			m.moveSelection(-1)
			return nil
		case matches(msg, keys.Down):
			m.moveSelection(1)
			return nil
		case matches(msg, keys.Enter):
			return m.handleEnter()
		}

	case domainDashboardLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.warnings = msg.warnings
		m.dataDomain = msg.domain
		m.dataZone = msg.zone
		m.records = msg.records
		if m.selectedRecord >= len(m.records) {
			m.selectedRecord = maxInt(0, len(m.records)-1)
		}
		return nil
	case domainDashboardRecordDetailMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.recordDetail = msg.record
		m.section = domainSectionRecords
		return nil
	case domainDashboardZoneFileMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.section = domainSectionDiagnostics
		m.diagTitle = "Zone File"
		m.diagBody = msg.zoneFile
		return nil
	case domainDashboardZoneDistributionMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.section = domainSectionDiagnostics
		m.diagTitle = "Zone Distribution"
		status := "Not distributed yet"
		if msg.distributed {
			status = "Fully distributed"
		}
		m.diagBody = "Distributed: " + strconv.FormatBool(msg.distributed) + "\nStatus: " + status
		return nil
	case domainDashboardRecordDistributionMsg:
		m.loading = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.section = domainSectionDiagnostics
		status := "Not distributed yet"
		if msg.distributed {
			status = "Fully distributed"
		}
		m.diagTitle = fmt.Sprintf("Record %d Distribution", msg.recordID)
		m.diagBody = "Distributed: " + strconv.FormatBool(msg.distributed) + "\nStatus: " + status
		return nil
	case domainDashboardMutationMsg:
		m.loading = false
		m.modal.visible = false
		m.modal.busy = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			return nil
		}
		m.errMsg = ""
		m.status = msg.status
		if msg.exit {
			return func() tea.Msg { return domainDashboardDeletedMsg{domain: msg.domain} }
		}
		if msg.reload {
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadDashboardCmd())
		}
	case spinner.TickMsg:
		if m.loading || m.modal.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
	}

	return nil
}

func (m *DomainDashboardModel) updateModal(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if matches(msg, keys.Back) && !m.modal.busy {
			m.modal.visible = false
			m.modal.errMsg = ""
			m.modal.input.Blur()
			return nil
		}
		if matches(msg, keys.Enter) && !m.modal.busy {
			if strings.TrimSpace(m.modal.input.Value()) != "confirm" {
				m.modal.errMsg = "Type confirm to proceed"
				return nil
			}
			m.modal.busy = true
			m.modal.errMsg = ""
			return tea.Batch(m.spinner.Tick, m.mutationCmd())
		}
		var cmd tea.Cmd
		m.modal.input, cmd = m.modal.input.Update(msg)
		return cmd
	case spinner.TickMsg:
		if m.modal.busy {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *DomainDashboardModel) View() string {
	body := []string{
		titleStyle.Render("Domain Dashboard"),
		subtitleStyle.Render(m.domain + "  |  full-screen domain operations"),
		tabStyle.Render(m.domainSectionTabs()),
		"",
		panelStyle.Render(m.dashboardBody()),
		"",
		footerStyle.Render(m.footerText()),
	}
	content := frame(m.width, strings.Join(body, "\n"))
	if m.modal.visible {
		content = overlayDialog(content, m.confirmDialogView())
	}
	return content
}

func (m *DomainDashboardModel) domainSectionTabs() string {
	defs := []struct {
		s     domainDashSection
		label string
		key   string
	}{
		{domainSectionOverview, "Overview", "o"},
		{domainSectionRecords, "Records", "c"}, // reCords; r is refresh
		{domainSectionZone, "Zone", "z"},
		{domainSectionDiagnostics, "Diagnostics", "g"}, // diaGnostics
		{domainSectionActions, "Actions", "a"},
	}
	parts := make([]string, 0, len(defs))
	for _, d := range defs {
		if d.s == m.section {
			parts = append(parts, renderMnemonicTabLabel(d.label, d.key, activeTabStyle))
		} else {
			parts = append(parts, renderMnemonicTabLabel(d.label, d.key, tabStyle))
		}
	}
	return strings.Join(parts, " ")
}

func (m *DomainDashboardModel) dashboardBody() string {
	if m.loading && m.dataDomain == nil {
		return m.spinner.View() + " Loading dashboard..."
	}
	if m.errMsg != "" && m.dataDomain == nil {
		return errorStyle.Render("Failed to load dashboard") + "\n" + m.errMsg
	}

	switch m.section {
	case domainSectionRecords:
		return m.recordsSection()
	case domainSectionZone:
		return m.zoneSection()
	case domainSectionDiagnostics:
		return m.diagnosticsSection()
	case domainSectionActions:
		return m.actionsSection()
	default:
		return m.overviewSection()
	}
}

func (m *DomainDashboardModel) overviewSection() string {
	lines := []string{panelTitleStyle.Render("Overview"), ""}
	if m.dataDomain != nil {
		d := m.dataDomain
		lines = append(lines,
			"Domain ID: "+strconv.FormatInt(d.ID, 10),
			"Name: "+d.Name,
			"State: "+d.State,
			"Auto-Renew: "+strconv.FormatBool(d.AutoRenew),
			"Private WHOIS: "+strconv.FormatBool(d.PrivateWhois),
		)
		if d.ExpiresAt != "" {
			lines = append(lines, "Expires: "+d.ExpiresAt)
		}
		lines = append(lines, "Created: "+d.CreatedAt, "Updated: "+d.UpdatedAt)
	}
	lines = append(lines, "")
	if m.dataZone != nil {
		z := m.dataZone
		lines = append(lines,
			panelTitleStyle.Render("Zone Summary"),
			"",
			"Zone ID: "+strconv.FormatInt(z.ID, 10),
			"Active: "+strconv.FormatBool(z.Active),
			"Reverse: "+strconv.FormatBool(z.Reverse),
			"Secondary: "+strconv.FormatBool(z.Secondary),
			fmt.Sprintf("Records loaded: %d", len(m.records)),
		)
	} else {
		lines = append(lines, warningStyle.Render("Zone not available for this domain"))
	}
	if len(m.warnings) > 0 {
		lines = append(lines, "", warningStyle.Render("Warnings"))
		for _, w := range m.warnings {
			lines = append(lines, "- "+w)
		}
	}
	if m.status != "" {
		lines = append(lines, "", successStyle.Render(m.status))
	}
	if m.errMsg != "" {
		lines = append(lines, "", errorStyle.Render(m.errMsg))
	}
	return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
}

func (m *DomainDashboardModel) recordsSection() string {
	lines := []string{panelTitleStyle.Render("Records"), ""}
	if len(m.records) == 0 {
		lines = append(lines, subtitleStyle.Render("No records loaded or zone unavailable."))
		return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
	}

	listCap := maxInt(6, m.contentLineBudget()/2)
	start, end := windowRange(len(m.records), m.selectedRecord, listCap)
	for i := start; i < end; i++ {
		r := m.records[i]
		name := r.Name
		if name == "" {
			name = "@"
		}
		prefix := "  "
		style := itemStyle
		if i == m.selectedRecord {
			prefix = "› "
			style = selectedItemStyle
		}
		row := fmt.Sprintf("%s%-6s %-18s ttl=%d %s", prefix, r.Type, truncateText(name, 18), r.TTL, truncateText(r.Content, 42))
		lines = append(lines, style.Render(row))
	}
	lines = append(lines, subtitleStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.records))))

	if rec := m.selectedRecordPtr(); rec != nil {
		lines = append(lines, "", panelTitleStyle.Render("Selected Record"))
		name := rec.Name
		if name == "" {
			name = "@"
		}
		lines = append(lines,
			"ID: "+strconv.FormatInt(rec.ID, 10),
			"Type: "+rec.Type,
			"Name: "+name,
		)
		lines = append(lines, wrapLabelValue("Content: ", rec.Content, 80)...)
		lines = append(lines, "TTL: "+strconv.Itoa(rec.TTL))
		if rec.Priority != 0 {
			lines = append(lines, "Priority: "+strconv.Itoa(rec.Priority))
		}
		if m.recordDetail != nil && m.recordDetail.ID == rec.ID {
			lines = append(lines, "System: "+strconv.FormatBool(m.recordDetail.SystemRecord))
		}
	}
	if m.status != "" {
		lines = append(lines, "", successStyle.Render(m.status))
	}
	if m.errMsg != "" {
		lines = append(lines, "", errorStyle.Render(m.errMsg))
	}
	return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
}

func (m *DomainDashboardModel) zoneSection() string {
	lines := []string{panelTitleStyle.Render("Zone"), ""}
	if m.dataZone == nil {
		lines = append(lines, warningStyle.Render("Zone not available."))
		return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
	}
	z := m.dataZone
	lines = append(lines,
		"ID: "+strconv.FormatInt(z.ID, 10),
		"Name: "+z.Name,
		"Active: "+strconv.FormatBool(z.Active),
		"Reverse: "+strconv.FormatBool(z.Reverse),
		"Secondary: "+strconv.FormatBool(z.Secondary),
		"Created: "+z.CreatedAt,
		"Updated: "+z.UpdatedAt,
		"",
		subtitleStyle.Render("Use f for zone file, x for zone distribution, and the Actions tab for mutations."),
	)
	if m.status != "" {
		lines = append(lines, "", successStyle.Render(m.status))
	}
	if m.errMsg != "" {
		lines = append(lines, "", errorStyle.Render(m.errMsg))
	}
	return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
}

func (m *DomainDashboardModel) diagnosticsSection() string {
	lines := []string{panelTitleStyle.Render("Diagnostics"), ""}
	if m.diagTitle == "" && m.diagBody == "" {
		lines = append(lines,
			subtitleStyle.Render("No diagnostic output yet."),
			"",
			"f  Fetch zone file",
			"x  Check zone distribution",
			"In Records section, x checks selected record distribution",
		)
	} else {
		if m.diagTitle != "" {
			lines = append(lines, valueStyle.Render(m.diagTitle), "")
		}
		lines = append(lines, strings.Split(clipMultilineText(m.diagBody, maxInt(6, m.contentLineBudget()-4)), "\n")...)
	}
	if m.status != "" {
		lines = append(lines, "", successStyle.Render(m.status))
	}
	if m.errMsg != "" {
		lines = append(lines, "", errorStyle.Render(m.errMsg))
	}
	return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
}

func (m *DomainDashboardModel) actionsSection() string {
	lines := []string{panelTitleStyle.Render("Actions"), ""}
	actions := m.availableActions()
	for i, a := range actions {
		prefix := "  "
		style := itemStyle
		if i == m.selectedAction {
			prefix = "› "
			style = selectedItemStyle
		}
		line := prefix + a.Label
		if !a.Enabled {
			line += " (unavailable)"
		}
		lines = append(lines, style.Render(line))
		if a.Hint != "" {
			lines = append(lines, subtitleStyle.Render("   "+a.Hint))
		}
	}
	lines = append(lines, "", subtitleStyle.Render("Mutations open a confirm dialog and require typing 'confirm'."))
	if m.status != "" {
		lines = append(lines, "", successStyle.Render(m.status))
	}
	if m.errMsg != "" {
		lines = append(lines, "", errorStyle.Render(m.errMsg))
	}
	return clipMultilineText(strings.Join(lines, "\n"), m.contentLineBudget())
}

func (m *DomainDashboardModel) footerText() string {
	base := "esc: domains list   /: global domain search   R: refresh dashboard   o/c/z/g/a: section"
	switch m.section {
	case domainSectionRecords:
		return base + "   enter: record details   x: record distribution   D: delete record"
	case domainSectionDiagnostics:
		return base + "   f: zone file   x: zone distribution"
	case domainSectionActions:
		return base + "   enter: run action"
	default:
		return base
	}
}

func (m *DomainDashboardModel) BlocksGlobalKeys() bool {
	return m.modal.visible
}

func (m *DomainDashboardModel) ModalVisible() bool {
	return m.modal.visible
}

func (m *DomainDashboardModel) contentLineBudget() int {
	if m.height <= 0 {
		return 24
	}
	return maxInt(10, m.height-16)
}

func (m *DomainDashboardModel) moveSelection(delta int) {
	switch m.section {
	case domainSectionRecords:
		if len(m.records) == 0 {
			return
		}
		m.selectedRecord = minInt(maxInt(0, m.selectedRecord+delta), len(m.records)-1)
	case domainSectionActions:
		actions := m.availableActions()
		if len(actions) == 0 {
			return
		}
		m.selectedAction = minInt(maxInt(0, m.selectedAction+delta), len(actions)-1)
	}
}

func (m *DomainDashboardModel) handleEnter() tea.Cmd {
	switch m.section {
	case domainSectionRecords:
		rec := m.selectedRecordPtr()
		if rec == nil {
			return nil
		}
		m.loading = true
		return tea.Batch(m.spinner.Tick, m.loadRecordDetailCmd(rec.ID))
	case domainSectionActions:
		actions := m.availableActions()
		if len(actions) == 0 || m.selectedAction >= len(actions) {
			return nil
		}
		act := actions[m.selectedAction]
		if !act.Enabled {
			m.errMsg = act.DisabledReason
			return nil
		}
		switch act.ID {
		case "refresh":
			m.loading = true
			m.errMsg = ""
			return tea.Batch(m.spinner.Tick, m.loadDashboardCmd())
		case "zone_file":
			m.section = domainSectionDiagnostics
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadZoneFileCmd())
		case "zone_distribution":
			m.section = domainSectionDiagnostics
			m.loading = true
			return tea.Batch(m.spinner.Tick, m.loadZoneDistributionCmd())
		case "zone_activate":
			m.openConfirm(mutationZoneActivate, "Activate Zone DNS", "This will activate DNS services for this zone.")
			return textinput.Blink
		case "zone_deactivate":
			m.openConfirm(mutationZoneDeactivate, "Deactivate Zone DNS", "This will deactivate DNS services for this zone.")
			return textinput.Blink
		case "delete_record":
			m.openConfirm(mutationDeleteRecord, "Delete Record", "This will permanently delete the selected record.")
			return textinput.Blink
		case "delete_domain":
			m.openConfirm(mutationDeleteDomain, "Delete Domain", "This will permanently delete the domain from your account.")
			return textinput.Blink
		}
	}
	return nil
}

type dashboardAction struct {
	ID             string
	Label          string
	Hint           string
	Enabled        bool
	DisabledReason string
}

func (m *DomainDashboardModel) availableActions() []dashboardAction {
	recAvailable := m.selectedRecordPtr() != nil
	zoneAvailable := m.dataZone != nil
	return []dashboardAction{
		{ID: "refresh", Label: "Refresh dashboard", Hint: "Reload domain, zone, and records", Enabled: true},
		{ID: "zone_file", Label: "Fetch zone file", Hint: "Read-only", Enabled: zoneAvailable, DisabledReason: "Zone unavailable"},
		{ID: "zone_distribution", Label: "Check zone distribution", Hint: "Read-only", Enabled: zoneAvailable, DisabledReason: "Zone unavailable"},
		{ID: "zone_activate", Label: "Activate DNS for zone", Hint: "Mutation (confirm required)", Enabled: zoneAvailable, DisabledReason: "Zone unavailable"},
		{ID: "zone_deactivate", Label: "Deactivate DNS for zone", Hint: "Mutation (confirm required)", Enabled: zoneAvailable, DisabledReason: "Zone unavailable"},
		{ID: "delete_record", Label: "Delete selected record", Hint: "Mutation (confirm required)", Enabled: recAvailable, DisabledReason: "Select a record first"},
		{ID: "delete_domain", Label: "Delete domain", Hint: "Mutation (confirm required)", Enabled: true},
	}
}

func (m *DomainDashboardModel) selectedRecordPtr() *dnsimple.ZoneRecord {
	if m.selectedRecord < 0 || m.selectedRecord >= len(m.records) {
		return nil
	}
	return &m.records[m.selectedRecord]
}

func (m *DomainDashboardModel) openConfirm(action confirmMutation, title, body string) {
	m.modal.visible = true
	m.modal.busy = false
	m.modal.action = action
	m.modal.title = title
	m.modal.body = body
	if target := m.mutationTargetSummary(action); target != "" {
		m.modal.body += "\n\n" + target
	}
	m.modal.errMsg = ""
	m.modal.input.SetValue("")
	m.modal.input.Focus()
}

func (m *DomainDashboardModel) confirmDialogView() string {
	lines := []string{
		panelTitleStyle.Render(m.modal.title),
		"",
		m.modal.body,
		"",
		warningStyle.Render("Type 'confirm' to proceed."),
		m.modal.input.View(),
	}
	if m.modal.errMsg != "" {
		lines = append(lines, "", errorStyle.Render(m.modal.errMsg))
	}
	if m.modal.busy {
		lines = append(lines, "", m.spinner.View()+" Working...")
	}
	lines = append(lines, "", footerStyle.Render("Enter: confirm   esc: cancel"))
	box := modalPanelStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(maxInt(60, m.width), maxInt(18, m.height), lipgloss.Center, lipgloss.Center, box)
}

func (m *DomainDashboardModel) mutationCmd() tea.Cmd {
	action := m.modal.action
	domain := m.domain
	var recordID int64
	if rec := m.selectedRecordPtr(); rec != nil {
		recordID = rec.ID
	}
	return func() tea.Msg {
		ctx := context.Background()
		backend := getBackend()
		switch action {
		case mutationZoneActivate:
			err := backend.ActivateZoneDNS(ctx, domain)
			return domainDashboardMutationMsg{
				kind:   "zone_activate",
				status: "Zone DNS activated.",
				reload: true,
				err:    wrapErr("failed to activate zone", err),
			}
		case mutationZoneDeactivate:
			err := backend.DeactivateZoneDNS(ctx, domain)
			return domainDashboardMutationMsg{
				kind:   "zone_deactivate",
				status: "Zone DNS deactivated.",
				reload: true,
				err:    wrapErr("failed to deactivate zone", err),
			}
		case mutationDeleteRecord:
			err := backend.DeleteRecord(ctx, domain, recordID)
			return domainDashboardMutationMsg{
				kind:   "delete_record",
				status: fmt.Sprintf("Record %d deleted.", recordID),
				reload: true,
				err:    wrapErr("failed to delete record", err),
			}
		case mutationDeleteDomain:
			err := backend.DeleteDomain(ctx, domain)
			return domainDashboardMutationMsg{
				kind:   "delete_domain",
				domain: domain,
				status: "Domain deleted.",
				exit:   err == nil,
				err:    wrapErr("failed to delete domain", err),
			}
		default:
			return domainDashboardMutationMsg{err: fmt.Errorf("unsupported mutation")}
		}
	}
}

func (m *DomainDashboardModel) loadDashboardCmd() tea.Cmd {
	domain := m.domain
	return func() tea.Msg {
		ctx := context.Background()
		backend := getBackend()

		dataDomain, err := backend.GetDomain(ctx, domain)
		if err != nil {
			return domainDashboardLoadedMsg{err: err}
		}

		var zone *dnsimple.Zone
		var records []dnsimple.ZoneRecord
		var warnings []string

		respZone, err := backend.GetZone(ctx, domain)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("zone unavailable: %v", err))
		} else {
			zone = respZone
			respRecords, err := backend.ListRecords(ctx, domain)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("records unavailable: %v", err))
			} else {
				records = respRecords
			}
		}

		return domainDashboardLoadedMsg{
			domain:   dataDomain,
			zone:     zone,
			records:  records,
			warnings: warnings,
		}
	}
}

func (m *DomainDashboardModel) loadRecordDetailCmd(recordID int64) tea.Cmd {
	domain := m.domain
	return func() tea.Msg {
		resp, err := getBackend().GetRecord(context.Background(), domain, recordID)
		if err != nil {
			return domainDashboardRecordDetailMsg{err: err}
		}
		return domainDashboardRecordDetailMsg{record: resp}
	}
}

func (m *DomainDashboardModel) loadZoneFileCmd() tea.Cmd {
	domain := m.domain
	return func() tea.Msg {
		resp, err := getBackend().GetZoneFile(context.Background(), domain)
		if err != nil {
			return domainDashboardZoneFileMsg{err: err}
		}
		return domainDashboardZoneFileMsg{zoneFile: resp}
	}
}

func (m *DomainDashboardModel) loadZoneDistributionCmd() tea.Cmd {
	domain := m.domain
	return func() tea.Msg {
		resp, err := getBackend().CheckZoneDistribution(context.Background(), domain)
		if err != nil {
			return domainDashboardZoneDistributionMsg{err: err}
		}
		return domainDashboardZoneDistributionMsg{distributed: resp}
	}
}

func (m *DomainDashboardModel) loadRecordDistributionCmd(recordID int64) tea.Cmd {
	domain := m.domain
	return func() tea.Msg {
		resp, err := getBackend().CheckRecordDistribution(context.Background(), domain, recordID)
		if err != nil {
			return domainDashboardRecordDistributionMsg{err: err}
		}
		return domainDashboardRecordDistributionMsg{recordID: recordID, distributed: resp}
	}
}

func wrapErr(prefix string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func (m *DomainDashboardModel) mutationTargetSummary(action confirmMutation) string {
	lines := []string{}
	switch action {
	case mutationZoneActivate, mutationZoneDeactivate:
		lines = append(lines,
			"Target zone:",
			"  "+m.domain,
		)
	case mutationDeleteDomain:
		lines = append(lines,
			"Target domain:",
			"  "+m.domain,
		)
	case mutationDeleteRecord:
		rec := m.selectedRecordPtr()
		if rec == nil {
			lines = append(lines, "Target record: (none selected)")
			break
		}
		name := rec.Name
		if name == "" {
			name = "@"
		}
		lines = append(lines,
			"Target record:",
			fmt.Sprintf("  %s %s.%s", rec.Type, name, m.domain),
			fmt.Sprintf("  ID: %d", rec.ID),
		)
	}
	return strings.Join(lines, "\n")
}

func renderMnemonicLabel(label, key string) string {
	if key == "" {
		return label
	}
	i := strings.Index(strings.ToLower(label), strings.ToLower(key))
	if i < 0 {
		return label
	}
	j := i + len(key)
	return label[:i] + lipgloss.NewStyle().Underline(true).Render(label[i:j]) + label[j:]
}

func renderMnemonicTabLabel(label, key string, base lipgloss.Style) string {
	noPad := base.Copy().Padding(0, 0)
	under := noPad.Copy().Underline(true)

	if key == "" {
		return noPad.Render(" ") + noPad.Render(label) + noPad.Render(" ")
	}

	i := strings.Index(strings.ToLower(label), strings.ToLower(key))
	if i < 0 {
		return noPad.Render(" ") + noPad.Render(label) + noPad.Render(" ")
	}
	j := i + len(key)

	var out strings.Builder
	out.WriteString(noPad.Render(" "))
	if i > 0 {
		out.WriteString(noPad.Render(label[:i]))
	}
	out.WriteString(under.Render(label[i:j]))
	if j < len(label) {
		out.WriteString(noPad.Render(label[j:]))
	}
	out.WriteString(noPad.Render(" "))
	return out.String()
}
