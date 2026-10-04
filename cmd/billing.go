package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "View billing information",
	Long: `View account billing information.

Examples:
  simple billing charges
  simple billing charges --start-date 2026-01-01 --end-date 2026-03-31`,
}

func listCharges(ctx context.Context, c *dnsimple.Client, accountID string, opts dnsimple.ListChargesOptions) ([]dnsimple.Charge, error) {
	resp, err := c.Billing.ListCharges(ctx, accountID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list charges: %w", err)
	}
	return resp.Data, nil
}

var billingChargesCmd = &cobra.Command{
	Use:   "charges",
	Short: "List charges",
	Long: `List charges for the account, optionally within a date range.

Examples:
  simple billing charges
  simple billing charges --start-date 2026-01-01 --end-date 2026-06-30
  simple billing charges --sort invoiced:desc
  simple billing charges --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var opts dnsimple.ListChargesOptions
		opts.StartDate, _ = cmd.Flags().GetString("start-date")
		opts.EndDate, _ = cmd.Flags().GetString("end-date")
		opts.Sort, _ = cmd.Flags().GetString("sort")

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		charges, err := listCharges(ctx, app.Client, app.AccountID, opts)
		if err != nil {
			return err
		}
		if printJSON(charges) {
			return nil
		}
		if len(charges) == 0 {
			fmt.Println(ui.Warn("No charges found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("💳 %d charges", len(charges))))
		fmt.Println()
		for _, ch := range charges {
			fmt.Printf("  %-12s %-16s %10s  %s",
				shortDate(ch.InvoicedAt),
				ui.AccentStyle.Render(ch.Reference),
				"$"+ch.TotalAmount.StringFixed(2),
				ch.State,
			)
			if !ch.BalanceAmount.IsZero() {
				fmt.Printf("  %s", ui.WarningStyle.Render("balance: $"+ch.BalanceAmount.StringFixed(2)))
			}
			fmt.Println()
			for _, item := range ch.Items {
				fmt.Printf("      %s %s\n", ui.SubtleStyle.Render(fmt.Sprintf("%9s", "$"+item.Amount.StringFixed(2))), item.Description)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(billingCmd)
	billingCmd.AddCommand(billingChargesCmd)
	billingChargesCmd.Flags().String("start-date", "", "Only charges on or after this date (YYYY-MM-DD)")
	billingChargesCmd.Flags().String("end-date", "", "Only charges on or before this date (YYYY-MM-DD)")
	billingChargesCmd.Flags().String("sort", "", "Sort order (e.g. invoiced:asc, invoiced:desc)")
}
