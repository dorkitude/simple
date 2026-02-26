package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var zonesCmd = &cobra.Command{
	Use:     "zones",
	Aliases: []string{"zone"},
	Short:   "Manage DNS zones",
	Long:    `List, view, activate, deactivate, and inspect DNS zones.`,
}

var zonesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all zones",
	Long: `List all DNS zones in your account.

Examples:
  simple zones list
  simple zones list --filter example
  simple zones list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		filter, _ := cmd.Flags().GetString("filter")
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		opts := &dnsimple.ZoneListOptions{}
		if filter != "" {
			opts.NameLike = dnsimpleString(filter)
		}
		if page > 0 {
			opts.Page = dnsimpleInt(page)
		}
		if perPage > 0 {
			opts.PerPage = dnsimpleInt(perPage)
		}

		resp, err := app.Client.Zones.ListZones(ctx, app.AccountID, opts)
		if err != nil {
			return fmt.Errorf("failed to list zones: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		if len(resp.Data) == 0 {
			fmt.Println(ui.Warn("No zones found"))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🗂️  %d zones", len(resp.Data))))
		fmt.Println()

		for _, z := range resp.Data {
			activeMarker := ui.SuccessStyle.Render("●")
			if !z.Active {
				activeMarker = ui.SubtleStyle.Render("○")
			}

			extra := ""
			if z.Reverse {
				extra = ui.SubtleStyle.Render(" (reverse)")
			}
			if z.Secondary {
				extra += ui.SubtleStyle.Render(" (secondary)")
			}

			fmt.Printf("  %s %-30s%s\n", activeMarker, ui.AccentStyle.Render(z.Name), extra)
		}

		return nil
	},
}

var zonesGetCmd = &cobra.Command{
	Use:   "get [zone]",
	Short: "Get zone details",
	Long:  `Display detailed information about a specific zone.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		resp, err := app.Client.Zones.GetZone(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to get zone: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		z := resp.Data
		fmt.Println(ui.TitleStyle.Render("🗂️  " + z.Name))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", z.ID)
		fmt.Printf("  %-14s %s\n", "Name:", z.Name)
		fmt.Printf("  %-14s %v\n", "Active:", z.Active)
		fmt.Printf("  %-14s %v\n", "Reverse:", z.Reverse)
		fmt.Printf("  %-14s %v\n", "Secondary:", z.Secondary)
		fmt.Printf("  %-14s %s\n", "Created:", z.CreatedAt)
		fmt.Printf("  %-14s %s\n", "Updated:", z.UpdatedAt)

		return nil
	},
}

var zonesFileCmd = &cobra.Command{
	Use:   "file [zone]",
	Short: "Get zone file",
	Long: `Fetch and display the zone file for a zone.

Examples:
  simple zones file example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		resp, err := app.Client.Zones.GetZoneFile(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to get zone file: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		fmt.Println(ui.TitleStyle.Render("📄 Zone file: " + args[0]))
		fmt.Println()
		fmt.Println(resp.Data.Zone)

		return nil
	},
}

var zonesDistributionCmd = &cobra.Command{
	Use:   "distribution [zone]",
	Short: "Check zone distribution",
	Long: `Check if a zone is fully distributed across DNSimple name servers.

Examples:
  simple zones distribution example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		resp, err := app.Client.Zones.CheckZoneDistribution(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to check distribution: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		if resp.Data.Distributed {
			fmt.Println(ui.Success(fmt.Sprintf("Zone '%s' is fully distributed ✨", args[0])))
		} else {
			fmt.Println(ui.Warn(fmt.Sprintf("Zone '%s' is NOT fully distributed yet", args[0])))
		}

		return nil
	},
}

var zonesActivateCmd = &cobra.Command{
	Use:   "activate [zone]",
	Short: "Activate DNS for a zone",
	Long:  `Activate DNS services for a zone.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		_, err = app.Client.Zones.ActivateZoneDns(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to activate zone: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("DNS activated for zone '%s'", args[0])))
		return nil
	},
}

var zonesDeactivateCmd = &cobra.Command{
	Use:   "deactivate [zone]",
	Short: "Deactivate DNS for a zone",
	Long:  `Deactivate DNS services for a zone.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		_, err = app.Client.Zones.DeactivateZoneDns(ctx, app.AccountID, args[0])
		if err != nil {
			return fmt.Errorf("failed to deactivate zone: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("DNS deactivated for zone '%s'", args[0])))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(zonesCmd)

	zonesCmd.AddCommand(zonesListCmd)
	zonesListCmd.Flags().StringP("filter", "f", "", "Filter zones by name")
	zonesListCmd.Flags().Int("page", 0, "Page number")
	zonesListCmd.Flags().Int("per-page", 0, "Results per page")

	zonesCmd.AddCommand(zonesGetCmd)
	zonesCmd.AddCommand(zonesFileCmd)
	zonesCmd.AddCommand(zonesDistributionCmd)
	zonesCmd.AddCommand(zonesActivateCmd)
	zonesCmd.AddCommand(zonesDeactivateCmd)
}
