package tui

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/client"
	"github.com/dorkitude/simple/internal/config"
)

type authStep int

const (
	authStepWelcome authStep = iota
	authStepConfigDir
	authStepTokenInput
	authStepValidating
	authStepSuccess
	authStepError
)

type tokenValidatedMsg struct {
	token string
	data  *dnsimple.WhoamiData
	err   error
}

type authDoneMsg struct {
	data *dnsimple.WhoamiData
}

type AuthModel struct {
	step         authStep
	width        int
	height       int
	configDir    string
	pathInput    textinput.Model
	tokenInput   textinput.Model
	spinner      spinner.Model
	validating   bool
	validToken   string
	whoami       *dnsimple.WhoamiData
	errMsg       string
	tokenHelpURL string
}

func NewAuthModel() AuthModel {
	cfgDir, err := config.ResolveConfigDir()
	if err != nil {
		cfgDir = "~/.config/dnsimplectl"
	}

	pathInput := textinput.New()
	pathInput.Placeholder = cfgDir
	pathInput.SetValue(cfgDir)
	pathInput.Prompt = "> "
	pathInput.CharLimit = 512
	pathInput.Width = 64

	tokenInput := textinput.New()
	tokenInput.Placeholder = "Paste your DNSimple API token"
	tokenInput.Prompt = "> "
	tokenInput.CharLimit = 512
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'
	tokenInput.Width = 64

	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = subtitleStyle

	return AuthModel{
		step:         authStepWelcome,
		configDir:    cfgDir,
		pathInput:    pathInput,
		tokenInput:   tokenInput,
		spinner:      spin,
		tokenHelpURL: "https://dnsimple.com/a/YOUR_ACCOUNT_ID/account/access_tokens",
	}
}

func (m *AuthModel) Init() tea.Cmd {
	return nil
}

func (m *AuthModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

func (m *AuthModel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.step {
		case authStepWelcome:
			if matches(msg, keys.Enter) {
				m.step = authStepTokenInput
				m.tokenInput.Focus()
				return textinput.Blink
			}
			if msg.String() == "c" {
				m.step = authStepConfigDir
				m.pathInput.SetValue(m.configDir)
				m.pathInput.Focus()
				return textinput.Blink
			}
		case authStepConfigDir:
			if matches(msg, keys.Back) {
				m.step = authStepWelcome
				m.pathInput.Blur()
				return nil
			}
			if matches(msg, keys.Enter) {
				path := strings.TrimSpace(m.pathInput.Value())
				if path == "" {
					m.errMsg = "config directory cannot be empty"
					m.step = authStepError
					return nil
				}
				config.SetConfigDir(path)
				cfgDir, err := config.ResolveConfigDir()
				if err != nil {
					m.errMsg = err.Error()
					m.step = authStepError
					return nil
				}
				m.configDir = cfgDir
				m.pathInput.SetValue(cfgDir)
				m.pathInput.Blur()
				m.step = authStepWelcome
				return nil
			}
			var cmd tea.Cmd
			m.pathInput, cmd = m.pathInput.Update(msg)
			return cmd
		case authStepTokenInput:
			if matches(msg, keys.Back) {
				m.step = authStepWelcome
				m.tokenInput.Blur()
				return nil
			}
			if matches(msg, keys.Enter) {
				token := strings.TrimSpace(m.tokenInput.Value())
				if token == "" {
					m.errMsg = "token cannot be empty"
					m.step = authStepError
					return nil
				}
				m.step = authStepValidating
				m.validating = true
				m.errMsg = ""
				m.validToken = ""
				return tea.Batch(m.spinner.Tick, validateTokenCmd(token))
			}
			var cmd tea.Cmd
			m.tokenInput, cmd = m.tokenInput.Update(msg)
			return cmd
		case authStepSuccess:
			if matches(msg, keys.Enter) {
				return func() tea.Msg { return authDoneMsg{data: m.whoami} }
			}
		case authStepError:
			if matches(msg, keys.Enter) {
				m.step = authStepTokenInput
				m.tokenInput.Focus()
				return textinput.Blink
			}
			if matches(msg, keys.Back) {
				m.step = authStepWelcome
				return nil
			}
		}
	case tokenValidatedMsg:
		m.validating = false
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			m.step = authStepError
			return nil
		}
		m.validToken = msg.token
		m.whoami = msg.data
		if err := persistAuth(msg.token, msg.data); err != nil {
			m.errMsg = err.Error()
			m.step = authStepError
			return nil
		}
		m.step = authStepSuccess
		return nil
	case spinner.TickMsg:
		if m.step == authStepValidating {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return cmd
		}
	}

	if m.step == authStepTokenInput {
		var cmd tea.Cmd
		m.tokenInput, cmd = m.tokenInput.Update(msg)
		return cmd
	}
	if m.step == authStepConfigDir {
		var cmd tea.Cmd
		m.pathInput, cmd = m.pathInput.Update(msg)
		return cmd
	}
	return nil
}

func (m AuthModel) View() string {
	switch m.step {
	case authStepWelcome:
		return m.viewWelcome()
	case authStepConfigDir:
		return m.viewConfigDir()
	case authStepTokenInput:
		return m.viewTokenInput()
	case authStepValidating:
		return m.viewValidating()
	case authStepSuccess:
		return m.viewSuccess()
	case authStepError:
		return m.viewError()
	default:
		return ""
	}
}

func (m AuthModel) tokenPathPreview() string {
	return filepath.Join(m.configDir, "token")
}

func (m AuthModel) configPathPreview() string {
	return filepath.Join(m.configDir, "config.json")
}

func (m AuthModel) viewWelcome() string {
	body := strings.Join([]string{
		titleStyle.Render("DNSimple Setup"),
		subtitleStyle.Render("Welcome. This TUI will store your API token locally and verify it before continuing."),
		"",
		panelStyle.Render(strings.Join([]string{
			panelTitleStyle.Render("Credential Storage"),
			"",
			labelStyle.Render("Config dir: ") + valueStyle.Render(m.configDir),
			labelStyle.Render("Token file: ") + codeStyle.Render(m.tokenPathPreview()),
			labelStyle.Render("Config file: ") + codeStyle.Render(m.configPathPreview()),
		}, "\n")),
		"",
		footerStyle.Render("Enter: continue   c: change config path   q: quit"),
	}, "\n")
	return frame(m.width, body)
}

func (m AuthModel) viewConfigDir() string {
	body := strings.Join([]string{
		titleStyle.Render("Config Directory"),
		subtitleStyle.Render("Choose where credentials should be stored for this session."),
		"",
		panelStyle.Render(strings.Join([]string{
			panelTitleStyle.Render("Path"),
			"",
			m.pathInput.View(),
			"",
			subtitleStyle.Render("This updates both token and config file locations."),
		}, "\n")),
		"",
		footerStyle.Render("Enter: apply   esc: back   q: quit"),
	}, "\n")
	return frame(m.width, body)
}

func (m AuthModel) viewTokenInput() string {
	body := strings.Join([]string{
		titleStyle.Render("API Token"),
		subtitleStyle.Render("Paste a DNSimple API token. Input is masked."),
		"",
		panelStyle.Render(strings.Join([]string{
			panelTitleStyle.Render("Token"),
			"",
			m.tokenInput.View(),
			"",
			labelStyle.Render("Get a token at: ") + linkStyle.Render(m.tokenHelpURL),
			labelStyle.Render("Will be saved to: ") + codeStyle.Render(m.tokenPathPreview()),
		}, "\n")),
		"",
		footerStyle.Render("Enter: validate   esc: back   q: quit"),
	}, "\n")
	return frame(m.width, body)
}

func (m AuthModel) viewValidating() string {
	body := strings.Join([]string{
		titleStyle.Render("Validating Token"),
		"",
		panelStyle.Render(m.spinner.View() + " Checking your token with the DNSimple Whoami API..."),
		"",
		footerStyle.Render("q: quit"),
	}, "\n")
	return frame(m.width, body)
}

func (m AuthModel) viewSuccess() string {
	lines := []string{
		titleStyle.Render("Authentication Complete"),
		"",
		panelStyle.Render(strings.Join([]string{
			panelTitleStyle.Render("Account"),
			"",
			successStyle.Render("Token validated and saved"),
			labelStyle.Render("Token file: ") + codeStyle.Render(m.tokenPathPreview()),
			labelStyle.Render("Config file: ") + codeStyle.Render(m.configPathPreview()),
			"",
			renderWhoamiSummary(m.whoami),
		}, "\n")),
		"",
		footerStyle.Render("Enter: go to home   q: quit"),
	}
	return frame(m.width, strings.Join(lines, "\n"))
}

func (m AuthModel) viewError() string {
	body := strings.Join([]string{
		titleStyle.Render("Authentication Error"),
		"",
		panelStyle.Render(strings.Join([]string{
			errorStyle.Render("Could not complete authentication"),
			"",
			m.errMsg,
		}, "\n")),
		"",
		footerStyle.Render("Enter: retry token entry   esc: welcome   q: quit"),
	}, "\n")
	return frame(m.width, body)
}

func validateTokenCmd(token string) tea.Cmd {
	return func() tea.Msg {
		data, err := client.ValidateToken(context.Background(), token, false)
		return tokenValidatedMsg{token: token, data: data, err: err}
	}
}

func persistAuth(token string, whoami *dnsimple.WhoamiData) error {
	if err := config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	cfg := &config.Config{}
	cfg.Sandbox = false

	if whoami != nil {
		if whoami.Account != nil {
			cfg.AccountID = strconv.FormatInt(whoami.Account.ID, 10)
		} else if whoami.User != nil {
			// Best-effort account resolution for user tokens.
			if app, err := client.NewFromFlags(context.Background(), "", false); err == nil {
				cfg.AccountID = app.AccountID
			}
		}
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

func renderWhoamiSummary(whoami *dnsimple.WhoamiData) string {
	if whoami == nil {
		return subtitleStyle.Render("No identity details available yet.")
	}
	if whoami.Account != nil {
		a := whoami.Account
		return strings.Join([]string{
			labelStyle.Render("Type: ") + valueStyle.Render("Account token"),
			labelStyle.Render("Email: ") + valueStyle.Render(a.Email),
			labelStyle.Render("ID: ") + valueStyle.Render(strconv.FormatInt(a.ID, 10)),
			labelStyle.Render("Plan: ") + valueStyle.Render(a.PlanIdentifier),
		}, "\n")
	}
	if whoami.User != nil {
		u := whoami.User
		return strings.Join([]string{
			labelStyle.Render("Type: ") + valueStyle.Render("User token"),
			labelStyle.Render("Email: ") + valueStyle.Render(u.Email),
			labelStyle.Render("ID: ") + valueStyle.Render(strconv.FormatInt(u.ID, 10)),
			labelStyle.Render("Plan: ") + valueStyle.Render("n/a"),
		}, "\n")
	}
	return subtitleStyle.Render("Whoami returned no account or user details.")
}
