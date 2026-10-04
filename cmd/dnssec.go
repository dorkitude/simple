package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// DnssecStatus is the result of a DNSSEC query or change.
type DnssecStatus struct {
	Domain  string `json:"domain"`
	Enabled bool   `json:"enabled"`
	Changed *bool  `json:"changed,omitempty"`
}

// getDnssec reports whether DNSSEC is enabled for a domain.
func getDnssec(ctx context.Context, c *dnsimple.Client, accountID, domain string) (bool, error) {
	resp, err := c.Domains.GetDnssec(ctx, accountID, domain)
	if err != nil {
		return false, fmt.Errorf("failed to get DNSSEC status for '%s': %w", domain, err)
	}
	return resp.Data != nil && resp.Data.Enabled, nil
}

// setDnssec enables or disables DNSSEC for a domain.
func setDnssec(ctx context.Context, c *dnsimple.Client, accountID, domain string, enable bool) error {
	var err error
	if enable {
		_, err = c.Domains.EnableDnssec(ctx, accountID, domain)
	} else {
		_, err = c.Domains.DisableDnssec(ctx, accountID, domain)
	}
	if err != nil {
		action := "disable"
		if enable {
			action = "enable"
		}
		return fmt.Errorf("failed to %s DNSSEC for '%s': %w", action, domain, err)
	}
	return nil
}

var domainsDnssecCmd = &cobra.Command{
	Use:   "dnssec [domain] [on|off]",
	Short: "Show or set DNSSEC for a domain",
	Long: `Show or change DNSSEC signing for a domain.

With only a domain, shows whether DNSSEC is enabled.
With on/off, enables or disables DNSSEC. Turning DNSSEC off while DS records
are still published at the registry can make the domain unresolvable, so
"off" requires --yes.

Examples:
  simple domains dnssec example.com
  simple domains dnssec example.com on
  simple domains dnssec example.com off --yes
  simple domains dnssec example.com --json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]

		var want *bool
		if len(args) == 2 {
			v, err := parseOnOff(args[1])
			if err != nil {
				return fmt.Errorf("invalid DNSSEC setting %q (use on or off)", args[1])
			}
			want = &v
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		enabled, err := getDnssec(ctx, app.Client, app.AccountID, domain)
		if err != nil {
			return err
		}

		if want == nil {
			status := DnssecStatus{Domain: domain, Enabled: enabled}
			if printJSON(status) {
				return nil
			}
			style := ui.WarningStyle
			if enabled {
				style = ui.SuccessStyle
			}
			fmt.Println(ui.TitleStyle.Render("🔐 " + domain))
			fmt.Println()
			fmt.Printf("  %-14s %s\n", "DNSSEC:", style.Render(onOff(enabled)))
			return nil
		}

		changed := enabled != *want
		if changed && !*want {
			if err := requireYes(cmd,
				fmt.Sprintf("disable DNSSEC for '%s'", domain),
				"remove any DS records at the registry first, or resolvers will reject the domain",
			); err != nil {
				return err
			}
		}
		if changed {
			if err := setDnssec(ctx, app.Client, app.AccountID, domain, *want); err != nil {
				return err
			}
		}

		status := DnssecStatus{Domain: domain, Enabled: *want, Changed: &changed}
		if printJSON(status) {
			return nil
		}
		if changed {
			fmt.Println(ui.Success(fmt.Sprintf("DNSSEC turned %s for '%s'", onOff(*want), domain)))
		} else {
			fmt.Println(ui.Info(fmt.Sprintf("DNSSEC already %s for '%s'; nothing to do", onOff(*want), domain)))
		}
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsDnssecCmd)
	addYesFlag(domainsDnssecCmd, "required to turn DNSSEC off")
}
