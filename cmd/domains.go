package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
	Use:     "domains",
	Aliases: []string{"domain", "dom"},
	Short:   "Manage domains",
	Long:    `List, view, create, and delete domains in your DNSimple account.`,
}

var domainsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all domains",
	Long: `List all domains in your account.

Examples:
  simple domains list
  simple domains list --filter example
  simple domains list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		filter, _ := cmd.Flags().GetString("filter")
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		opts := &dnsimple.DomainListOptions{}
		if filter != "" {
			opts.NameLike = dnsimpleString(filter)
		}
		if page > 0 {
			opts.Page = dnsimpleInt(page)
		}
		if perPage > 0 {
			opts.PerPage = dnsimpleInt(perPage)
		}

		resp, err := app.Client.Domains.ListDomains(ctx, app.AccountID, opts)
		if err != nil {
			return fmt.Errorf("failed to list domains: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		if len(resp.Data) == 0 {
			fmt.Println(ui.Warn("No domains found"))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🌐 %d domains", len(resp.Data))))
		fmt.Println()

		for _, d := range resp.Data {
			stateColor := ui.SuccessStyle
			if d.State != "registered" && d.State != "hosted" {
				stateColor = ui.WarningStyle
			}

			fmt.Printf("  %-30s %s",
				ui.AccentStyle.Render(d.Name),
				stateColor.Render(d.State),
			)
			if d.ExpiresAt != "" {
				fmt.Printf("  %s", ui.SubtleStyle.Render("expires: "+d.ExpiresAt[:10]))
			}
			if d.AutoRenew {
				fmt.Printf("  %s", ui.SubtleStyle.Render("↻"))
			}
			fmt.Println()
		}

		return nil
	},
}

var domainsGetCmd = &cobra.Command{
	Use:   "get [domain]",
	Short: "Get domain details",
	Long:  `Display detailed information about a specific domain.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		resp, err := app.Client.Domains.GetDomain(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to get domain: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		d := resp.Data
		fmt.Println(ui.TitleStyle.Render("🌐 " + d.Name))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", d.ID)
		fmt.Printf("  %-14s %s\n", "Name:", d.Name)
		if d.UnicodeName != d.Name && d.UnicodeName != "" {
			fmt.Printf("  %-14s %s\n", "Unicode:", d.UnicodeName)
		}
		fmt.Printf("  %-14s %s\n", "State:", d.State)
		fmt.Printf("  %-14s %v\n", "Auto-Renew:", d.AutoRenew)
		fmt.Printf("  %-14s %v\n", "Private WHOIS:", d.PrivateWhois)
		if d.ExpiresAt != "" {
			fmt.Printf("  %-14s %s\n", "Expires:", d.ExpiresAt)
		}
		fmt.Printf("  %-14s %s\n", "Created:", d.CreatedAt)
		fmt.Printf("  %-14s %s\n", "Updated:", d.UpdatedAt)

		return nil
	},
}

var domainsCreateCmd = &cobra.Command{
	Use:   "create [domain-name]",
	Short: "Add a domain to the account",
	Long: `Add a domain to your DNSimple account (does not register it).

Examples:
  simple domains create example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		domain := dnsimple.Domain{Name: args[0]}
		resp, err := app.Client.Domains.CreateDomain(ctx, app.AccountID, domain)
		if err != nil {
			return fmt.Errorf("failed to create domain: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		fmt.Println(ui.Success(fmt.Sprintf("Domain '%s' added! (ID: %d)", resp.Data.Name, resp.Data.ID)))
		return nil
	},
}

var domainsDeleteCmd = &cobra.Command{
	Use:   "delete [domain]",
	Short: "Delete a domain from the account",
	Long: `PERMANENTLY delete a domain from your account.

Examples:
  simple domains delete example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		_, err = app.Client.Domains.DeleteDomain(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to delete domain: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Domain '%s' deleted.", args[0])))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(domainsCmd)

	domainsCmd.AddCommand(domainsListCmd)
	domainsListCmd.Flags().StringP("filter", "f", "", "Filter domains by name")
	domainsListCmd.Flags().Int("page", 0, "Page number")
	domainsListCmd.Flags().Int("per-page", 0, "Results per page")

	domainsCmd.AddCommand(domainsGetCmd)
	domainsCmd.AddCommand(domainsCreateCmd)
	domainsCmd.AddCommand(domainsDeleteCmd)
}
