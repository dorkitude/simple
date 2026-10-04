package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// initiateDomainPush starts moving a domain to another DNSimple account.
func initiateDomainPush(ctx context.Context, c *dnsimple.Client, accountID, domain, toEmail, toAccount string) (*dnsimple.DomainPush, error) {
	if toEmail == "" && toAccount == "" {
		return nil, fmt.Errorf("--to (the new account's email) or --to-account is required")
	}
	resp, err := c.Domains.InitiatePush(ctx, accountID, domain, dnsimple.DomainPushAttributes{
		NewAccountEmail:      toEmail,
		NewAccountIdentifier: toAccount,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to push '%s': %w", domain, err)
	}
	return resp.Data, nil
}

func listDomainPushes(ctx context.Context, c *dnsimple.Client, accountID string, opts *dnsimple.ListOptions) ([]dnsimple.DomainPush, error) {
	resp, err := c.Domains.ListPushes(ctx, accountID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pushes: %w", err)
	}
	return resp.Data, nil
}

func acceptDomainPush(ctx context.Context, c *dnsimple.Client, accountID string, pushID, contactID int64) error {
	if contactID <= 0 {
		return fmt.Errorf("--contact-id is required (the registrant contact in this account)")
	}
	if _, err := c.Domains.AcceptPush(ctx, accountID, pushID, dnsimple.DomainPushAttributes{ContactID: contactID}); err != nil {
		return fmt.Errorf("failed to accept push %d: %w", pushID, err)
	}
	return nil
}

func rejectDomainPush(ctx context.Context, c *dnsimple.Client, accountID string, pushID int64) error {
	if _, err := c.Domains.RejectPush(ctx, accountID, pushID); err != nil {
		return fmt.Errorf("failed to reject push %d: %w", pushID, err)
	}
	return nil
}

var domainsPushCmd = &cobra.Command{
	Use:   "push [domain]",
	Short: "Push a domain to another DNSimple account",
	Long: `Start moving a domain to another DNSimple account. The other account must
accept the push (simple pushes accept). Requires --yes.

Examples:
  simple domains push example.com --to friend@example.net --yes
  simple domains push example.com --to-account 1234 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		toEmail, _ := cmd.Flags().GetString("to")
		toAccount, _ := cmd.Flags().GetString("to-account")
		if toEmail == "" && toAccount == "" {
			return fmt.Errorf("--to (the new account's email) or --to-account is required")
		}
		target := toEmail
		if target == "" {
			target = "account " + toAccount
		}
		if err := requireYes(cmd,
			fmt.Sprintf("push '%s' to %s", args[0], target),
			"once accepted, the domain leaves this account",
		); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		p, err := initiateDomainPush(ctx, app.Client, app.AccountID, args[0], toEmail, toAccount)
		if err != nil {
			return err
		}
		if printJSON(p) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Push started for '%s' → %s (push ID: %d)", args[0], target, p.ID)))
		return nil
	},
}

var pushesCmd = &cobra.Command{
	Use:     "pushes",
	Aliases: []string{"push"},
	Short:   "Manage incoming domain pushes",
	Long:    `List, accept, and reject domain pushes sent to this account.`,
}

var pushesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending pushes",
	Long: `List domain pushes waiting for this account to accept or reject.

Examples:
  simple pushes list
  simple pushes list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		pushes, err := listDomainPushes(ctx, app.Client, app.AccountID, pageOptions(cmd))
		if err != nil {
			return err
		}
		if printJSON(pushes) {
			return nil
		}
		if len(pushes) == 0 {
			fmt.Println(ui.Info("No pending pushes"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📦 %d pushes", len(pushes))))
		fmt.Println()
		for _, p := range pushes {
			fmt.Printf("  %-10s %s %s\n",
				ui.AccentStyle.Render(fmt.Sprint(p.ID)),
				fmt.Sprintf("domain %d", p.DomainID),
				ui.SubtleStyle.Render("created "+shortDate(p.CreatedAt)))
		}
		return nil
	},
}

var pushesAcceptCmd = &cobra.Command{
	Use:   "accept [push-id]",
	Short: "Accept a domain push",
	Long: `Accept a domain push into this account, assigning a registrant contact.

Examples:
  simple pushes accept 42 --contact-id 7`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("push", args[0])
		if err != nil {
			return err
		}
		contactID, _ := cmd.Flags().GetInt64("contact-id")
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := acceptDomainPush(ctx, app.Client, app.AccountID, id, contactID); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Push %d accepted 🎉", id)))
		return nil
	},
}

var pushesRejectCmd = &cobra.Command{
	Use:   "reject [push-id]",
	Short: "Reject a domain push",
	Long: `Reject a domain push. The sender must start a new push to try again,
so this requires --yes.

Examples:
  simple pushes reject 42 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("push", args[0])
		if err != nil {
			return err
		}
		if err := requireYes(cmd, fmt.Sprintf("reject push %d", id)); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := rejectDomainPush(ctx, app.Client, app.AccountID, id); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Push %d rejected", id)))
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsPushCmd)
	domainsPushCmd.Flags().String("to", "", "Email of the receiving DNSimple account")
	domainsPushCmd.Flags().String("to-account", "", "Identifier of the receiving DNSimple account")
	addYesFlag(domainsPushCmd, "required to push the domain")

	rootCmd.AddCommand(pushesCmd)
	pushesCmd.AddCommand(pushesListCmd)
	addPageFlags(pushesListCmd)
	pushesCmd.AddCommand(pushesAcceptCmd)
	pushesAcceptCmd.Flags().Int64("contact-id", 0, "Registrant contact ID in this account")
	pushesCmd.AddCommand(pushesRejectCmd)
	addYesFlag(pushesRejectCmd, "required to reject the push")
}
