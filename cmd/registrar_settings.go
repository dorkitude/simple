package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// --- name server delegation ------------------------------------------------

// RegDelegation is the name server delegation of a registered domain.
type RegDelegation struct {
	Domain      string                      `json:"domain"`
	NameServers []string                    `json:"name_servers"`
	Vanity      []dnsimple.VanityNameServer `json:"vanity_name_servers,omitempty"`
}

func regGetDelegation(ctx context.Context, c *dnsimple.Client, accountID, domain string) ([]string, error) {
	resp, err := c.Registrar.GetDomainDelegation(ctx, accountID, domain)
	if err != nil {
		return nil, regError(fmt.Sprintf("get name servers for '%s'", domain), err)
	}
	if resp.Data == nil {
		return []string{}, nil
	}
	return []string(*resp.Data), nil
}

func regSetDelegation(ctx context.Context, c *dnsimple.Client, accountID, domain string, ns []string) ([]string, error) {
	d := dnsimple.Delegation(ns)
	resp, err := c.Registrar.ChangeDomainDelegation(ctx, accountID, domain, &d)
	if err != nil {
		return nil, regError(fmt.Sprintf("change name servers for '%s'", domain), err)
	}
	if resp.Data == nil {
		return ns, nil
	}
	return []string(*resp.Data), nil
}

func regDelegationToVanity(ctx context.Context, c *dnsimple.Client, accountID, domain string, ns []string) ([]dnsimple.VanityNameServer, error) {
	d := dnsimple.Delegation(ns)
	resp, err := c.Registrar.ChangeDomainDelegationToVanity(ctx, accountID, domain, &d)
	if err != nil {
		return nil, regError(fmt.Sprintf("delegate '%s' to vanity name servers", domain), err)
	}
	return resp.Data, nil
}

func regDelegationFromVanity(ctx context.Context, c *dnsimple.Client, accountID, domain string) error {
	if _, err := c.Registrar.ChangeDomainDelegationFromVanity(ctx, accountID, domain); err != nil {
		return regError(fmt.Sprintf("remove vanity name servers from '%s'", domain), err)
	}
	return nil
}

func printDelegation(w io.Writer, d *RegDelegation) {
	fmt.Fprintln(w, ui.TitleStyle.Render("🧭 "+d.Domain+" name servers"))
	if len(d.Vanity) > 0 {
		for _, v := range d.Vanity {
			fmt.Fprintf(w, "  %-30s %s\n", ui.AccentStyle.Render(v.Name), ui.SubtleStyle.Render(strings.TrimSpace(v.IPv4+" "+v.IPv6)))
		}
		return
	}
	if len(d.NameServers) == 0 {
		fmt.Fprintln(w, ui.Warn("No name servers delegated"))
		return
	}
	for _, ns := range d.NameServers {
		fmt.Fprintf(w, "  %s\n", ui.AccentStyle.Render(ns))
	}
}

var registrarNameServersCmd = &cobra.Command{
	Use:     "nameservers [domain] [ns...]",
	Aliases: []string{"ns", "delegation"},
	Short:   "Show or change a registered domain's name servers",
	Long: `Show or change the name servers a registered domain is delegated to.

With only a domain, shows the current delegation.
With name servers, replaces the delegation (requires --yes).
With --to-vanity and name servers, delegates to vanity name servers (requires --yes).
With --from-vanity, removes vanity delegation (requires --yes).

Examples:
  simple registrar nameservers example.com
  simple registrar nameservers example.com ns1.dnsimple.com ns2.dnsimple-edge.net ns3.dnsimple.com ns4.dnsimple-edge.org --yes
  simple registrar nameservers example.com --to-vanity ns1.example.com ns2.example.com --yes
  simple registrar nameservers example.com --from-vanity --yes`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain, ns := args[0], args[1:]
		toVanity, _ := cmd.Flags().GetBool("to-vanity")
		fromVanity, _ := cmd.Flags().GetBool("from-vanity")
		switch {
		case toVanity && fromVanity:
			return fmt.Errorf("--to-vanity and --from-vanity cannot be used together")
		case toVanity && len(ns) == 0:
			return fmt.Errorf("--to-vanity needs the vanity name servers to delegate to")
		case fromVanity && len(ns) > 0:
			return fmt.Errorf("--from-vanity takes no name servers")
		}
		cmd.SilenceUsage = true

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		if len(ns) == 0 && !fromVanity {
			current, err := regGetDelegation(ctx, app.Client, app.AccountID, domain)
			if err != nil {
				return err
			}
			d := &RegDelegation{Domain: domain, NameServers: current}
			if printJSON(d) {
				return nil
			}
			printDelegation(os.Stdout, d)
			return nil
		}

		if !regConfirmed(cmd) {
			p := regPlan{Domain: domain}
			if current, err := regGetDelegation(ctx, app.Client, app.AccountID, domain); err == nil {
				p.Details = append(p.Details, regField{"Current", strings.Join(current, ", ")})
			}
			switch {
			case fromVanity:
				p.Action = "remove vanity name servers from " + domain
				p.Details = append(p.Details, regField{"New", "DNSimple default name servers"})
			case toVanity:
				p.Action = "delegate " + domain + " to vanity name servers"
				p.Details = append(p.Details, regField{"New", strings.Join(ns, ", ")})
			default:
				p.Action = "change name servers of " + domain
				p.Details = append(p.Details, regField{"New", strings.Join(ns, ", ")})
			}
			p.Warnings = append(p.Warnings, "Changing delegation can take the domain offline if the new name servers do not serve its zone")
			return regGate(cmd, p)
		}

		switch {
		case fromVanity:
			if err := regDelegationFromVanity(ctx, app.Client, app.AccountID, domain); err != nil {
				return err
			}
			if printJSON(map[string]interface{}{"domain": domain, "vanity": false}) {
				return nil
			}
			fmt.Println(ui.Success(fmt.Sprintf("Vanity name servers removed from '%s'", domain)))
		case toVanity:
			v, err := regDelegationToVanity(ctx, app.Client, app.AccountID, domain, ns)
			if err != nil {
				return err
			}
			d := &RegDelegation{Domain: domain, NameServers: ns, Vanity: v}
			if printJSON(d) {
				return nil
			}
			fmt.Println(ui.Success(fmt.Sprintf("'%s' delegated to vanity name servers", domain)))
			printDelegation(os.Stdout, d)
		default:
			got, err := regSetDelegation(ctx, app.Client, app.AccountID, domain, ns)
			if err != nil {
				return err
			}
			d := &RegDelegation{Domain: domain, NameServers: got}
			if printJSON(d) {
				return nil
			}
			fmt.Println(ui.Success(fmt.Sprintf("Name servers updated for '%s'", domain)))
			printDelegation(os.Stdout, d)
		}
		return nil
	},
}

// --- transfer lock ---------------------------------------------------------

// RegToggle is the on/off state of a registrar setting.
type RegToggle struct {
	Domain  string `json:"domain"`
	Setting string `json:"setting"`
	Enabled bool   `json:"enabled"`
}

func regGetTransferLock(ctx context.Context, c *dnsimple.Client, accountID, domain string) (bool, error) {
	resp, err := c.Registrar.GetDomainTransferLock(ctx, accountID, domain)
	if err != nil {
		return false, regError(fmt.Sprintf("get transfer lock for '%s'", domain), err)
	}
	return resp.Data != nil && resp.Data.Enabled, nil
}

func regSetTransferLock(ctx context.Context, c *dnsimple.Client, accountID, domain string, enable bool) (bool, error) {
	var resp *dnsimple.DomainTransferLockResponse
	var err error
	if enable {
		resp, err = c.Registrar.EnableDomainTransferLock(ctx, accountID, domain)
	} else {
		resp, err = c.Registrar.DisableDomainTransferLock(ctx, accountID, domain)
	}
	if err != nil {
		return false, regError(fmt.Sprintf("turn transfer lock %s for '%s'", onOff(enable), domain), err)
	}
	if resp.Data == nil {
		return enable, nil
	}
	return resp.Data.Enabled, nil
}

func printRegToggle(w io.Writer, emoji string, t *RegToggle) {
	fmt.Fprintln(w, ui.TitleStyle.Render(emoji+" "+t.Domain))
	style := ui.WarningStyle
	if t.Enabled {
		style = ui.SuccessStyle
	}
	fmt.Fprintf(w, "  %-14s %s\n", t.Setting+":", style.Render(onOff(t.Enabled)))
}

var registrarTransferLockCmd = &cobra.Command{
	Use:     "transfer-lock [domain] [on|off]",
	Aliases: []string{"lock"},
	Short:   "Show or set a domain's transfer lock",
	Long: `Show or change the registrar transfer lock, which blocks transfers away from DNSimple.

Examples:
  simple registrar transfer-lock example.com
  simple registrar transfer-lock example.com on
  simple registrar transfer-lock example.com off`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		var want *bool
		if len(args) == 2 {
			v, err := parseOnOff(args[1])
			if err != nil {
				return fmt.Errorf("invalid transfer lock setting %q (use on or off)", args[1])
			}
			want = &v
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t := &RegToggle{Domain: domain, Setting: "Transfer lock"}
		if want == nil {
			t.Enabled, err = regGetTransferLock(ctx, app.Client, app.AccountID, domain)
		} else {
			t.Enabled, err = regSetTransferLock(ctx, app.Client, app.AccountID, domain, *want)
		}
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		if want != nil {
			fmt.Println(ui.Success(fmt.Sprintf("Transfer lock turned %s for '%s'", onOff(t.Enabled), domain)))
			return nil
		}
		printRegToggle(os.Stdout, "🔒", t)
		return nil
	},
}

// --- WHOIS privacy ---------------------------------------------------------

func regSetWhoisPrivacy(ctx context.Context, c *dnsimple.Client, accountID, domain string, enable bool) (*dnsimple.WhoisPrivacy, error) {
	var resp *dnsimple.WhoisPrivacyResponse
	var err error
	if enable {
		resp, err = c.Registrar.EnableWhoisPrivacy(ctx, accountID, domain)
	} else {
		resp, err = c.Registrar.DisableWhoisPrivacy(ctx, accountID, domain)
	}
	if err != nil {
		return nil, regError(fmt.Sprintf("turn WHOIS privacy %s for '%s'", onOff(enable), domain), err)
	}
	if resp.Data == nil {
		return &dnsimple.WhoisPrivacy{Enabled: enable}, nil
	}
	return resp.Data, nil
}

// regGetWhoisPrivacy reads WHOIS privacy from the domain record (the
// registrar API has no read endpoint for it any more).
func regGetWhoisPrivacy(ctx context.Context, c *dnsimple.Client, accountID, domain string) (bool, error) {
	resp, err := c.Domains.GetDomain(ctx, accountID, domain)
	if err != nil {
		return false, regError(fmt.Sprintf("get domain '%s'", domain), err)
	}
	return resp.Data.PrivateWhois, nil
}

var registrarWhoisPrivacyCmd = &cobra.Command{
	Use:     "whois-privacy [domain] [on|off]",
	Aliases: []string{"privacy", "whois"},
	Short:   "Show or set WHOIS privacy for a registered domain",
	Long: `Show or change WHOIS privacy, which hides the registrant's contact details.

Examples:
  simple registrar whois-privacy example.com
  simple registrar whois-privacy example.com on
  simple registrar whois-privacy example.com off`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		var want *bool
		if len(args) == 2 {
			v, err := parseOnOff(args[1])
			if err != nil {
				return fmt.Errorf("invalid WHOIS privacy setting %q (use on or off)", args[1])
			}
			want = &v
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if want == nil {
			on, err := regGetWhoisPrivacy(ctx, app.Client, app.AccountID, domain)
			if err != nil {
				return err
			}
			t := &RegToggle{Domain: domain, Setting: "WHOIS privacy", Enabled: on}
			if printJSON(t) {
				return nil
			}
			printRegToggle(os.Stdout, "🕶️", t)
			return nil
		}
		wp, err := regSetWhoisPrivacy(ctx, app.Client, app.AccountID, domain, *want)
		if err != nil {
			return err
		}
		if printJSON(wp) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("WHOIS privacy turned %s for '%s'", onOff(*want), domain)))
		if wp.ExpiresOn != "" {
			fmt.Printf("  %-14s %s\n", "Expires:", wp.ExpiresOn)
		}
		return nil
	},
}

// --- auto-renewal ----------------------------------------------------------

var registrarAutoRenewCmd = &cobra.Command{
	Use:     "autorenew [domain] [on|off]",
	Aliases: []string{"auto-renew"},
	Short:   "Show or set auto-renew for a registered domain",
	Long: `Show or change auto-renewal for a domain registered with DNSimple.
Same as 'domains autorenew'.

Examples:
  simple registrar autorenew example.com
  simple registrar autorenew example.com on
  simple registrar autorenew example.com off`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return domainsAutoRenewCmd.RunE(cmd, args)
	},
}

func init() {
	registrarNameServersCmd.Flags().Bool("to-vanity", false, "Delegate to the given vanity name servers")
	registrarNameServersCmd.Flags().Bool("from-vanity", false, "Remove vanity name server delegation")
	regAddYesFlag(registrarNameServersCmd)
	registrarCmd.AddCommand(registrarNameServersCmd)

	registrarCmd.AddCommand(registrarTransferLockCmd)
	registrarCmd.AddCommand(registrarWhoisPrivacyCmd)
	registrarCmd.AddCommand(registrarAutoRenewCmd)
}
