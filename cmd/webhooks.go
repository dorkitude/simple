package cmd

import (
	"context"
	"fmt"
	"net/url"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var webhooksCmd = &cobra.Command{
	Use:     "webhooks",
	Aliases: []string{"webhook", "hooks"},
	Short:   "Manage webhooks",
	Long: `List, view, create, and delete account webhooks.

Examples:
  simple webhooks list
  simple webhooks create https://example.com/dnsimple-hook
  simple webhooks delete 7 --yes`,
}

// ---------- logic (testable) ----------

func listWebhooks(ctx context.Context, c *dnsimple.Client, accountID string) ([]dnsimple.Webhook, error) {
	resp, err := c.Webhooks.ListWebhooks(ctx, accountID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhooks: %w", err)
	}
	return resp.Data, nil
}

func getWebhook(ctx context.Context, c *dnsimple.Client, accountID string, id int64) (*dnsimple.Webhook, error) {
	resp, err := c.Webhooks.GetWebhook(ctx, accountID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook: %w", err)
	}
	return resp.Data, nil
}

func createWebhook(ctx context.Context, c *dnsimple.Client, accountID, hookURL string) (*dnsimple.Webhook, error) {
	u, err := url.Parse(hookURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("invalid webhook URL %q (expected http(s)://host/...)", hookURL)
	}
	resp, err := c.Webhooks.CreateWebhook(ctx, accountID, dnsimple.Webhook{URL: hookURL})
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}
	return resp.Data, nil
}

func deleteWebhook(ctx context.Context, c *dnsimple.Client, accountID string, id int64) error {
	if _, err := c.Webhooks.DeleteWebhook(ctx, accountID, id); err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	return nil
}

// ---------- commands ----------

var webhooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "List webhooks",
	Long: `List all webhooks in your account.

Examples:
  simple webhooks list
  simple webhooks list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		hooks, err := listWebhooks(ctx, app.Client, app.AccountID)
		if err != nil {
			return err
		}
		if printJSON(hooks) {
			return nil
		}
		if len(hooks) == 0 {
			fmt.Println(ui.Warn("No webhooks found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🪝 %d webhooks", len(hooks))))
		fmt.Println()
		for _, h := range hooks {
			fmt.Printf("  %-10d %s\n", h.ID, ui.AccentStyle.Render(h.URL))
		}
		return nil
	},
}

var webhooksGetCmd = &cobra.Command{
	Use:   "get [webhook-id]",
	Short: "Get webhook details",
	Long: `Display a webhook.

Examples:
  simple webhooks get 7`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("webhook ID", args[0])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		h, err := getWebhook(ctx, app.Client, app.AccountID, id)
		if err != nil {
			return err
		}
		if printJSON(h) {
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🪝 Webhook %d", h.ID)))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", h.ID)
		fmt.Printf("  %-14s %s\n", "URL:", h.URL)
		return nil
	},
}

var webhooksCreateCmd = &cobra.Command{
	Use:   "create [url]",
	Short: "Create a webhook",
	Long: `Register a URL to receive DNSimple webhook events.

Examples:
  simple webhooks create https://example.com/dnsimple-hook`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		h, err := createWebhook(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(h) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Webhook created! (ID: %d) → %s", h.ID, h.URL)))
		return nil
	},
}

var webhooksDeleteCmd = &cobra.Command{
	Use:   "delete [webhook-id]",
	Short: "Delete a webhook (requires --yes)",
	Long: `PERMANENTLY delete a webhook. Requires --yes.

Examples:
  simple webhooks delete 7 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("webhook ID", args[0])
		if err != nil {
			return err
		}
		if err := confirmWithYes(cmd, "permanently delete a webhook", []string{fmt.Sprintf("%-14s %d", "Webhook ID:", id)}); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deleteWebhook(ctx, app.Client, app.AccountID, id); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"id": id, "deleted": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Webhook %d deleted.", id)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(webhooksCmd)
	webhooksCmd.AddCommand(webhooksListCmd)
	webhooksCmd.AddCommand(webhooksGetCmd)
	webhooksCmd.AddCommand(webhooksCreateCmd)
	webhooksCmd.AddCommand(webhooksDeleteCmd)
	webhooksDeleteCmd.Flags().Bool("yes", false, "Confirm the deletion")
}
