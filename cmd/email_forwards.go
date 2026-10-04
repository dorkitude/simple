package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

func listEmailForwards(ctx context.Context, c *dnsimple.Client, accountID, domain string, opts *dnsimple.ListOptions) ([]dnsimple.EmailForward, error) {
	resp, err := c.Domains.ListEmailForwards(ctx, accountID, domain, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list email forwards for '%s': %w", domain, err)
	}
	return resp.Data, nil
}

func getEmailForward(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) (*dnsimple.EmailForward, error) {
	resp, err := c.Domains.GetEmailForward(ctx, accountID, domain, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get email forward %d: %w", id, err)
	}
	return resp.Data, nil
}

// createEmailForward creates alias@domain -> destination. alias may be a bare
// local part ("hello") or a full address on the domain ("hello@example.com").
func createEmailForward(ctx context.Context, c *dnsimple.Client, accountID, domain, alias, destination string) (*dnsimple.EmailForward, error) {
	if alias == "" || destination == "" {
		return nil, fmt.Errorf("--from and --to are required")
	}
	if at := strings.LastIndex(alias, "@"); at >= 0 {
		if !strings.EqualFold(alias[at+1:], domain) {
			return nil, fmt.Errorf("--from %q is not on domain '%s'", alias, domain)
		}
		alias = alias[:at]
	}
	resp, err := c.Domains.CreateEmailForward(ctx, accountID, domain, dnsimple.EmailForward{
		AliasName:        alias,
		DestinationEmail: destination,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create email forward: %w", err)
	}
	return resp.Data, nil
}

func deleteEmailForward(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) error {
	if _, err := c.Domains.DeleteEmailForward(ctx, accountID, domain, id); err != nil {
		return fmt.Errorf("failed to delete email forward %d: %w", id, err)
	}
	return nil
}

var domainsForwardsCmd = &cobra.Command{
	Use:     "forwards",
	Aliases: []string{"forward", "email-forwards", "fwd"},
	Short:   "Manage email forwards",
	Long:    `List, view, create, and delete email forwards for a domain.`,
}

var domainsForwardsListCmd = &cobra.Command{
	Use:   "list [domain]",
	Short: "List email forwards",
	Long: `List the email forwards for a domain.

Examples:
  simple domains forwards list example.com
  simple domains forwards list example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		forwards, err := listEmailForwards(ctx, app.Client, app.AccountID, args[0], pageOptions(cmd))
		if err != nil {
			return err
		}
		if printJSON(forwards) {
			return nil
		}
		if len(forwards) == 0 {
			fmt.Println(ui.Warn("No email forwards found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📬 %d email forwards for %s", len(forwards), args[0])))
		fmt.Println()
		for _, f := range forwards {
			marker := ui.SuccessStyle.Render("●")
			if !f.Active {
				marker = ui.SubtleStyle.Render("○")
			}
			fmt.Printf("  %s %-10s %-32s → %s\n", marker,
				ui.SubtleStyle.Render(fmt.Sprint(f.ID)),
				ui.AccentStyle.Render(f.AliasEmail),
				f.DestinationEmail)
		}
		return nil
	},
}

var domainsForwardsGetCmd = &cobra.Command{
	Use:   "get [domain] [forward-id]",
	Short: "Get an email forward",
	Long: `Display one email forward.

Examples:
  simple domains forwards get example.com 17`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("email forward", args[1])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		f, err := getEmailForward(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(f) {
			return nil
		}
		fmt.Println(ui.TitleStyle.Render("📬 " + f.AliasEmail))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", f.ID)
		fmt.Printf("  %-14s %s\n", "From:", f.AliasEmail)
		fmt.Printf("  %-14s %s\n", "To:", f.DestinationEmail)
		fmt.Printf("  %-14s %v\n", "Active:", f.Active)
		fmt.Printf("  %-14s %s\n", "Created:", f.CreatedAt)
		fmt.Printf("  %-14s %s\n", "Updated:", f.UpdatedAt)
		return nil
	},
}

var domainsForwardsCreateCmd = &cobra.Command{
	Use:   "create [domain]",
	Short: "Create an email forward",
	Long: `Forward mail for an address on the domain to another address.

--from takes the local part ("hello") or the full address ("hello@example.com").

Examples:
  simple domains forwards create example.com --from hello --to me@gmail.com
  simple domains forwards create example.com --from sales@example.com --to team@corp.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		f, err := createEmailForward(ctx, app.Client, app.AccountID, args[0], from, to)
		if err != nil {
			return err
		}
		if printJSON(f) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Forwarding %s → %s (ID: %d)", f.AliasEmail, f.DestinationEmail, f.ID)))
		return nil
	},
}

var domainsForwardsDeleteCmd = &cobra.Command{
	Use:   "delete [domain] [forward-id]",
	Short: "Delete an email forward",
	Long: `Delete an email forward.

Examples:
  simple domains forwards delete example.com 17`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("email forward", args[1])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deleteEmailForward(ctx, app.Client, app.AccountID, args[0], id); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Email forward %d deleted from '%s'", id, args[0])))
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsForwardsCmd)

	domainsForwardsCmd.AddCommand(domainsForwardsListCmd)
	addPageFlags(domainsForwardsListCmd)

	domainsForwardsCmd.AddCommand(domainsForwardsGetCmd)

	domainsForwardsCmd.AddCommand(domainsForwardsCreateCmd)
	domainsForwardsCreateCmd.Flags().String("from", "", "Alias on the domain (local part or full address)")
	domainsForwardsCreateCmd.Flags().String("to", "", "Destination email address")

	domainsForwardsCmd.AddCommand(domainsForwardsDeleteCmd)
}
