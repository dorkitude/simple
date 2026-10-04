package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/config"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var accountsCmd = &cobra.Command{
	Use:     "accounts",
	Aliases: []string{"account", "acct"},
	Short:   "List accounts and choose the active one",
	Long: `List the DNSimple accounts your token can access and pick which one
later commands use.

Examples:
  simple accounts list
  simple accounts use 1010`,
}

// ---------- logic (testable) ----------

func listAccounts(ctx context.Context, c *dnsimple.Client) ([]dnsimple.Account, error) {
	resp, err := c.Accounts.ListAccounts(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list accounts: %w", err)
	}
	return resp.Data, nil
}

// findAccount checks that id is one of the accounts the token can access.
func findAccount(ctx context.Context, c *dnsimple.Client, id int64) (*dnsimple.Account, error) {
	accts, err := listAccounts(ctx, c)
	if err != nil {
		return nil, err
	}
	for i := range accts {
		if accts[i].ID == id {
			return &accts[i], nil
		}
	}
	return nil, fmt.Errorf("account %d is not accessible with this token (see 'simple accounts list')", id)
}

// saveActiveAccount writes the account ID into the config so later commands use it.
func saveActiveAccount(id int64) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}
	cfg.AccountID = strconv.FormatInt(id, 10)
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	return nil
}

// ---------- commands ----------

var accountsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List accessible accounts",
	Long: `List the accounts this token can access. The active account is marked.

Examples:
  simple accounts list
  simple accounts list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		accts, err := listAccounts(ctx, app.Client)
		if err != nil {
			return err
		}
		if printJSON(accts) {
			return nil
		}
		if len(accts) == 0 {
			fmt.Println(ui.Warn("No accounts found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🏢 %d accounts", len(accts))))
		fmt.Println()
		for _, a := range accts {
			marker := "  "
			if strconv.FormatInt(a.ID, 10) == app.AccountID {
				marker = ui.SuccessStyle.Render("● ")
			}
			label := a.Email
			if a.Name != "" {
				label = a.Name + " <" + a.Email + ">"
			}
			fmt.Printf("%s%-10d %-40s %s\n", marker, a.ID, ui.AccentStyle.Render(label), ui.SubtleStyle.Render(a.PlanIdentifier))
		}
		return nil
	},
}

var accountsUseCmd = &cobra.Command{
	Use:   "use [account-id]",
	Short: "Set the active account",
	Long: `Save an account ID to the config so later commands use it by default
(the --account flag still overrides it).

Examples:
  simple accounts use 1010`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("account ID", args[0])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		acct, err := findAccount(ctx, app.Client, id)
		if err != nil {
			return err
		}
		if err := saveActiveAccount(id); err != nil {
			return err
		}
		if printJSON(acct) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Now using account %d (%s)", acct.ID, acct.Email)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(accountsCmd)
	accountsCmd.AddCommand(accountsListCmd)
	accountsCmd.AddCommand(accountsUseCmd)
}
