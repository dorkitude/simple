package cmd

import (
	"context"
	"fmt"

	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current identity",
	Long:  `Display information about the currently authenticated user or account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		resp, err := app.Client.Identity.Whoami(ctx)
		if err != nil {
			return fmt.Errorf("whoami failed: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		fmt.Println(ui.TitleStyle.Render("🌐 DNSimple Identity"))
		fmt.Println()

		if resp.Data.Account != nil {
			a := resp.Data.Account
			fmt.Println(ui.SuccessStyle.Render("Account Token"))
			fmt.Printf("  %-14s %d\n", "ID:", a.ID)
			fmt.Printf("  %-14s %s\n", "Email:", a.Email)
			fmt.Printf("  %-14s %s\n", "Plan:", a.PlanIdentifier)
		}

		if resp.Data.User != nil {
			u := resp.Data.User
			fmt.Println(ui.SuccessStyle.Render("User Token"))
			fmt.Printf("  %-14s %d\n", "ID:", u.ID)
			fmt.Printf("  %-14s %s\n", "Email:", u.Email)
		}

		fmt.Printf("  %-14s %s\n", "Account ID:", app.AccountID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
