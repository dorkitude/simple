package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

func enableVanity(ctx context.Context, c *dnsimple.Client, accountID, domain string) ([]dnsimple.VanityNameServer, error) {
	resp, err := c.VanityNameServers.EnableVanityNameServers(ctx, accountID, domain)
	if err != nil {
		return nil, fmt.Errorf("failed to enable vanity name servers for '%s': %w", domain, err)
	}
	return resp.Data, nil
}

func disableVanity(ctx context.Context, c *dnsimple.Client, accountID, domain string) error {
	if _, err := c.VanityNameServers.DisableVanityNameServers(ctx, accountID, domain); err != nil {
		return fmt.Errorf("failed to disable vanity name servers for '%s': %w", domain, err)
	}
	return nil
}

var vanityCmd = &cobra.Command{
	Use:   "vanity",
	Short: "Manage vanity name servers",
	Long: `Enable or disable vanity name servers (ns1.yourdomain.com and friends)
for a domain. This creates or removes the name server records in the zone;
point the registry delegation at them separately.`,
}

var vanityEnableCmd = &cobra.Command{
	Use:   "enable [domain]",
	Short: "Enable vanity name servers",
	Long: `Enable vanity name servers for a domain.

Examples:
  simple vanity enable example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		servers, err := enableVanity(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(servers) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Vanity name servers enabled for '%s' ✨", args[0])))
		fmt.Println()
		for _, s := range servers {
			fmt.Printf("  %-24s %s  %s\n", ui.AccentStyle.Render(s.Name), s.IPv4, ui.SubtleStyle.Render(s.IPv6))
		}
		return nil
	},
}

var vanityDisableCmd = &cobra.Command{
	Use:   "disable [domain]",
	Short: "Disable vanity name servers",
	Long: `Disable vanity name servers for a domain. If the registry still delegates
to them, the domain stops resolving, so this requires --yes.

Examples:
  simple vanity disable example.com --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireYes(cmd,
			fmt.Sprintf("disable vanity name servers for '%s'", args[0]),
			"move the registry delegation off them first, or the domain stops resolving",
		); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := disableVanity(ctx, app.Client, app.AccountID, args[0]); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Vanity name servers disabled for '%s'", args[0])))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(vanityCmd)
	vanityCmd.AddCommand(vanityEnableCmd)
	vanityCmd.AddCommand(vanityDisableCmd)
	addYesFlag(vanityDisableCmd, "required to disable vanity name servers")
}
