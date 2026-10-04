package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// AutoRenewStatus is the result of an auto-renew query or change.
type AutoRenewStatus struct {
	Domain    string `json:"domain"`
	State     string `json:"state,omitempty"`
	AutoRenew bool   `json:"auto_renew"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Changed   *bool  `json:"changed,omitempty"`
}

// parseOnOff turns a user-supplied toggle word into a bool.
func parseOnOff(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "enable", "enabled", "true", "yes", "1":
		return true, nil
	case "off", "disable", "disabled", "false", "no", "0":
		return false, nil
	}
	return false, fmt.Errorf("invalid auto-renew setting %q (use on or off)", s)
}

// autoRenewError turns registrar API errors into friendlier messages.
func autoRenewError(domain, action string, err error) error {
	var apiErr *dnsimple.ErrorResponse
	if errors.As(err, &apiErr) && apiErr.HTTPResponse != nil {
		switch apiErr.HTTPResponse.StatusCode {
		case http.StatusNotFound:
			return fmt.Errorf("domain '%s' is not registered with DNSimple in this account (API: %s)", domain, apiErr.Message)
		case http.StatusPaymentRequired, http.StatusBadRequest:
			return fmt.Errorf("failed to %s auto-renew for '%s': %s", action, domain, apiErr.Message)
		}
	}
	return fmt.Errorf("failed to %s auto-renew for '%s': %w", action, domain, err)
}

// getAutoRenewStatus fetches the current auto-renew status for a domain.
func getAutoRenewStatus(ctx context.Context, c *dnsimple.Client, accountID, domain string) (*AutoRenewStatus, error) {
	resp, err := c.Domains.GetDomain(ctx, accountID, domain)
	if err != nil {
		var apiErr *dnsimple.ErrorResponse
		if errors.As(err, &apiErr) && apiErr.HTTPResponse != nil && apiErr.HTTPResponse.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("domain '%s' not found in this DNSimple account (API: %s)", domain, apiErr.Message)
		}
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	d := resp.Data
	return &AutoRenewStatus{
		Domain:    d.Name,
		State:     d.State,
		AutoRenew: d.AutoRenew,
		ExpiresAt: d.ExpiresAt,
	}, nil
}

// setAutoRenew enables or disables auto-renewal via the registrar API.
func setAutoRenew(ctx context.Context, c *dnsimple.Client, accountID, domain string, enable bool) error {
	var err error
	if enable {
		_, err = c.Registrar.EnableDomainAutoRenewal(ctx, accountID, domain)
	} else {
		_, err = c.Registrar.DisableDomainAutoRenewal(ctx, accountID, domain)
	}
	if err != nil {
		action := "disable"
		if enable {
			action = "enable"
		}
		return autoRenewError(domain, action, err)
	}
	return nil
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func printAutoRenewStatus(s *AutoRenewStatus) {
	fmt.Println(ui.TitleStyle.Render("↻ " + s.Domain))
	fmt.Println()
	style := ui.WarningStyle
	if s.AutoRenew {
		style = ui.SuccessStyle
	}
	fmt.Printf("  %-14s %s\n", "Auto-Renew:", style.Render(onOff(s.AutoRenew)))
	if s.State != "" {
		fmt.Printf("  %-14s %s\n", "State:", s.State)
	}
	if s.ExpiresAt != "" {
		fmt.Printf("  %-14s %s\n", "Expires:", s.ExpiresAt)
	}
	if s.State != "" && s.State != "registered" {
		fmt.Println()
		fmt.Println(ui.Warn(fmt.Sprintf("'%s' is %s, not registered with DNSimple; auto-renew does not apply", s.Domain, s.State)))
	}
}

var domainsAutoRenewCmd = &cobra.Command{
	Use:     "autorenew [domain] [on|off]",
	Aliases: []string{"auto-renew", "renewal"},
	Short:   "Show or set auto-renew for a registered domain",
	Long: `Show or change auto-renewal for a domain registered with DNSimple.

With only a domain, shows the current auto-renew status.
With on/off, enables or disables auto-renewal via the registrar API.

Examples:
  simple domains autorenew example.com
  simple domains autorenew example.com on
  simple domains autorenew example.com off
  simple domains autorenew example.com --json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]

		var want *bool
		if len(args) == 2 {
			v, err := parseOnOff(args[1])
			if err != nil {
				return err
			}
			want = &v
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		if want == nil {
			status, err := getAutoRenewStatus(ctx, app.Client, app.AccountID, domain)
			if err != nil {
				return err
			}
			if printJSON(status) {
				return nil
			}
			printAutoRenewStatus(status)
			return nil
		}

		before, err := getAutoRenewStatus(ctx, app.Client, app.AccountID, domain)
		if err != nil {
			return err
		}

		if before.State == "hosted" {
			return fmt.Errorf("domain '%s' is hosted (DNS only), not registered with DNSimple; auto-renew does not apply", before.Domain)
		}

		changed := before.AutoRenew != *want
		if changed {
			if err := setAutoRenew(ctx, app.Client, app.AccountID, domain, *want); err != nil {
				return err
			}
		}

		after := *before
		after.AutoRenew = *want
		after.Changed = &changed

		if printJSON(after) {
			return nil
		}

		if changed {
			fmt.Println(ui.Success(fmt.Sprintf("Auto-renew turned %s for '%s'", onOff(*want), after.Domain)))
		} else {
			fmt.Println(ui.Info(fmt.Sprintf("Auto-renew already %s for '%s'; nothing to do", onOff(*want), after.Domain)))
		}
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsAutoRenewCmd)
}
