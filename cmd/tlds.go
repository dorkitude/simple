package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var tldsCmd = &cobra.Command{
	Use:     "tlds",
	Aliases: []string{"tld"},
	Short:   "Browse supported TLDs",
	Long: `List the TLDs DNSimple supports and inspect their registration rules.

Examples:
  simple tlds list
  simple tlds get com
  simple tlds attributes uk`,
}

func normalizeTLD(s string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), ".")
}

func listTLDs(ctx context.Context, c *dnsimple.Client, opts *dnsimple.ListOptions) (*dnsimple.TldsResponse, error) {
	resp, err := c.Tlds.ListTlds(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list TLDs: %w", err)
	}
	return resp, nil
}

func getTLD(ctx context.Context, c *dnsimple.Client, tld string) (*dnsimple.Tld, error) {
	resp, err := c.Tlds.GetTld(ctx, normalizeTLD(tld))
	if err != nil {
		return nil, fmt.Errorf("failed to get TLD: %w", err)
	}
	return resp.Data, nil
}

func getTLDAttributes(ctx context.Context, c *dnsimple.Client, tld string) ([]dnsimple.TldExtendedAttribute, error) {
	resp, err := c.Tlds.GetTldExtendedAttributes(ctx, normalizeTLD(tld))
	if err != nil {
		return nil, fmt.Errorf("failed to get TLD extended attributes: %w", err)
	}
	return resp.Data, nil
}

func yesNo(b bool) string {
	if b {
		return ui.SuccessStyle.Render("yes")
	}
	return ui.SubtleStyle.Render("no")
}

var tldsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List supported TLDs",
	Long: `List the TLDs supported by DNSimple (paginated).

Examples:
  simple tlds list
  simple tlds list --page 2 --per-page 100
  simple tlds list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		resp, err := listTLDs(ctx, app.Client, listOptionsFromFlags(cmd))
		if err != nil {
			return err
		}
		if printJSON(resp.Data) {
			return nil
		}
		if len(resp.Data) == 0 {
			fmt.Println(ui.Warn("No TLDs found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🌍 %d TLDs", len(resp.Data))))
		fmt.Println()
		for _, t := range resp.Data {
			var flags []string
			if !t.RegistrationEnabled {
				flags = append(flags, "no-register")
			}
			if !t.TransferEnabled {
				flags = append(flags, "no-transfer")
			}
			if t.WhoisPrivacy {
				flags = append(flags, "whois-privacy")
			}
			if t.AutoRenewOnly {
				flags = append(flags, "auto-renew-only")
			}
			fmt.Printf("  %-24s min %dy  %s\n", ui.AccentStyle.Render("."+t.Tld), t.MinimumRegistration, ui.SubtleStyle.Render(strings.Join(flags, " ")))
		}
		printPagination(resp.Pagination)
		return nil
	},
}

var tldsGetCmd = &cobra.Command{
	Use:   "get [tld]",
	Short: "Get TLD details",
	Long: `Display registration rules for a TLD.

Examples:
  simple tlds get com
  simple tlds get .io --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t, err := getTLD(ctx, app.Client, args[0])
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		fmt.Println(ui.TitleStyle.Render("🌍 ." + t.Tld))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "Type:", t.TldType)
		fmt.Printf("  %-14s %d year(s)\n", "Min Term:", t.MinimumRegistration)
		fmt.Printf("  %-14s %s\n", "Register:", yesNo(t.RegistrationEnabled))
		fmt.Printf("  %-14s %s\n", "Renew:", yesNo(t.RenewalEnabled))
		fmt.Printf("  %-14s %s\n", "Transfer:", yesNo(t.TransferEnabled))
		fmt.Printf("  %-14s %s\n", "WHOIS Privacy:", yesNo(t.WhoisPrivacy))
		fmt.Printf("  %-14s %s\n", "AutoRenewOnly:", yesNo(t.AutoRenewOnly))
		fmt.Printf("  %-14s %s\n", "Trustee:", yesNo(t.TrusteeServiceEnabled))
		if t.TrusteeServiceRequired {
			fmt.Printf("  %-14s %s\n", "Trustee Req:", yesNo(true))
		}
		if t.DnssecInterfaceType != "" {
			fmt.Printf("  %-14s %s\n", "DNSSEC:", t.DnssecInterfaceType)
		}
		return nil
	},
}

var tldsAttributesCmd = &cobra.Command{
	Use:     "attributes [tld]",
	Aliases: []string{"attrs", "extended-attributes"},
	Short:   "Show a TLD's extended registration attributes",
	Long: `Show the extended attributes a TLD requires or accepts at registration.

Examples:
  simple tlds attributes uk
  simple tlds attributes us --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		attrs, err := getTLDAttributes(ctx, app.Client, args[0])
		if err != nil {
			return err
		}
		if printJSON(attrs) {
			return nil
		}
		tld := normalizeTLD(args[0])
		if len(attrs) == 0 {
			fmt.Println(ui.Info(fmt.Sprintf(".%s has no extended attributes", tld)))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🌍 .%s extended attributes", tld)))
		fmt.Println()
		for _, a := range attrs {
			req := ui.SubtleStyle.Render("optional")
			if a.Required {
				req = ui.WarningStyle.Render("required")
			}
			fmt.Printf("  %s  %s\n", ui.AccentStyle.Render(a.Name), req)
			if a.Description != "" {
				fmt.Printf("    %s\n", a.Description)
			}
			for _, o := range a.Options {
				line := fmt.Sprintf("    • %-12s %s", o.Value, o.Title)
				if o.Description != "" && o.Description != o.Title {
					line += ui.SubtleStyle.Render(" — " + o.Description)
				}
				fmt.Println(line)
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tldsCmd)
	tldsCmd.AddCommand(tldsListCmd)
	addPageFlags(tldsListCmd)
	tldsCmd.AddCommand(tldsGetCmd)
	tldsCmd.AddCommand(tldsAttributesCmd)
}
