package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// templateRecordInput holds the fields for a new template record. Pointer
// fields are sent only when set, so an explicit 0 (TTL or priority) reaches
// the API instead of being dropped by the SDK's omitempty tags.
type templateRecordInput struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Content  string `json:"content"`
	TTL      *int   `json:"ttl,omitempty"`
	Priority *int   `json:"priority,omitempty"`
}

func listTemplateRecords(ctx context.Context, c *dnsimple.Client, accountID, template string, opts *dnsimple.ListOptions) ([]dnsimple.TemplateRecord, error) {
	resp, err := c.Templates.ListTemplateRecords(ctx, accountID, template, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list records for template '%s': %w", template, err)
	}
	return resp.Data, nil
}

func getTemplateRecord(ctx context.Context, c *dnsimple.Client, accountID, template string, recordID int64) (*dnsimple.TemplateRecord, error) {
	resp, err := c.Templates.GetTemplateRecord(ctx, accountID, template, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get record %d of template '%s': %w", recordID, template, err)
	}
	return resp.Data, nil
}

func createTemplateRecord(ctx context.Context, c *dnsimple.Client, accountID, template string, in templateRecordInput) (*dnsimple.TemplateRecord, error) {
	path := fmt.Sprintf("/v2/%s/templates/%s/records", url.PathEscape(accountID), url.PathEscape(template))
	out := &dnsimple.TemplateRecordResponse{}
	if _, err := c.Request(ctx, http.MethodPost, path, in, out, nil); err != nil {
		return nil, fmt.Errorf("failed to create record in template '%s': %w", template, err)
	}
	return out.Data, nil
}

func deleteTemplateRecord(ctx context.Context, c *dnsimple.Client, accountID, template string, recordID int64) error {
	if _, err := c.Templates.DeleteTemplateRecord(ctx, accountID, template, recordID); err != nil {
		return fmt.Errorf("failed to delete record %d of template '%s': %w", recordID, template, err)
	}
	return nil
}

func formatTemplateRecordLine(r dnsimple.TemplateRecord) string {
	name := r.Name
	if name == "" {
		name = "@"
	}
	pri := ""
	if r.Priority != 0 {
		pri = fmt.Sprintf(" (pri: %d)", r.Priority)
	}
	return fmt.Sprintf("%-8d %s %-20s %-6d %s%s",
		r.ID, ui.RecordTypeStyle.Render(r.Type), ui.AccentStyle.Render(name), r.TTL, truncate(r.Content, 50), pri)
}

func parseTemplateRecordID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid template record ID %q: %w", s, err)
	}
	return id, nil
}

var templateRecordsCmd = &cobra.Command{
	Use:     "records",
	Aliases: []string{"record", "rec"},
	Short:   "Manage records in a template",
	Long:    `List, view, create, and delete the records of a DNS template.`,
}

var templateRecordsListCmd = &cobra.Command{
	Use:   "list <template>",
	Short: "List a template's records",
	Long: `List the records in a template.

Examples:
  simple templates records list web
  simple tpl rec list web --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		records, err := listTemplateRecords(ctx, app.Client, app.AccountID, args[0], tplPageOptions(page, perPage))
		if err != nil {
			return err
		}
		if printJSON(records) {
			return nil
		}
		if len(records) == 0 {
			fmt.Println(ui.Warn("No records in template '" + args[0] + "'"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📐 %d records in template %s", len(records), args[0])))
		fmt.Println()
		for _, r := range records {
			fmt.Println("  " + formatTemplateRecordLine(r))
		}
		return nil
	},
}

var templateRecordsGetCmd = &cobra.Command{
	Use:   "get <template> <record-id>",
	Short: "Get a template record",
	Long: `Display details of one record in a template.

Examples:
  simple templates records get web 301`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseTemplateRecordID(args[1])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := getTemplateRecord(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		name := r.Name
		if name == "" {
			name = "@"
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📐 %s %s (template %s)", r.Type, name, args[0])))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", r.ID)
		fmt.Printf("  %-14s %d\n", "Template ID:", r.TemplateID)
		fmt.Printf("  %-14s %s\n", "Type:", r.Type)
		fmt.Printf("  %-14s %s\n", "Name:", name)
		fmt.Printf("  %-14s %s\n", "Content:", r.Content)
		fmt.Printf("  %-14s %d\n", "TTL:", r.TTL)
		if r.Priority != 0 {
			fmt.Printf("  %-14s %d\n", "Priority:", r.Priority)
		}
		fmt.Printf("  %-14s %s\n", "Created:", r.CreatedAt)
		fmt.Printf("  %-14s %s\n", "Updated:", r.UpdatedAt)
		return nil
	},
}

var templateRecordsCreateCmd = &cobra.Command{
	Use:   "create <template>",
	Short: "Add a record to a template",
	Long: `Add a record to a template. Does not change any domain until the template is applied.

Examples:
  simple templates records create web --type A --name "" --content 1.2.3.4
  simple templates records create web --type CNAME --name www --content example.com --ttl 3600
  simple templates records create mail --type MX --name "" --content mx.example.com --priority 0`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		in := templateRecordInput{}
		in.Type, _ = cmd.Flags().GetString("type")
		in.Name, _ = cmd.Flags().GetString("name")
		in.Content, _ = cmd.Flags().GetString("content")
		if in.Type == "" || in.Content == "" {
			return fmt.Errorf("--type and --content are required")
		}
		if in.Name == "@" {
			in.Name = ""
		}
		if cmd.Flags().Changed("ttl") {
			v, _ := cmd.Flags().GetInt("ttl")
			in.TTL = &v
		}
		if cmd.Flags().Changed("priority") {
			v, _ := cmd.Flags().GetInt("priority")
			in.Priority = &v
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := createTemplateRecord(ctx, app.Client, app.AccountID, args[0], in)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		name := r.Name
		if name == "" {
			name = "@"
		}
		fmt.Println(ui.Success(fmt.Sprintf("Added %s record '%s' → %s to template '%s' (ID: %d)", r.Type, name, r.Content, args[0], r.ID)))
		return nil
	},
}

var templateRecordsDeleteCmd = &cobra.Command{
	Use:   "delete <template> <record-id>",
	Short: "Delete a template record",
	Long: `PERMANENTLY delete a record from a template. Requires --yes.
Domains the template was already applied to are not changed.

Examples:
  simple templates records delete web 301        # shows what would happen
  simple templates records delete web 301 --yes`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		template := args[0]
		id, err := parseTemplateRecordID(args[1])
		if err != nil {
			return err
		}
		if err := tplConfirm(cmd, fmt.Sprintf("permanently delete record %d from template '%s'", id, template)); err != nil {
			return err
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deleteTemplateRecord(ctx, app.Client, app.AccountID, template, id); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"template": template, "record_id": id, "deleted": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Record %d deleted from template '%s'", id, template)))
		return nil
	},
}

func init() {
	templatesCmd.AddCommand(templateRecordsCmd)

	templateRecordsCmd.AddCommand(templateRecordsListCmd)
	templateRecordsListCmd.Flags().Int("page", 0, "Page number")
	templateRecordsListCmd.Flags().Int("per-page", 0, "Results per page")

	templateRecordsCmd.AddCommand(templateRecordsGetCmd)

	templateRecordsCmd.AddCommand(templateRecordsCreateCmd)
	templateRecordsCreateCmd.Flags().StringP("type", "t", "", "Record type (A, AAAA, CNAME, MX, TXT, etc.)")
	templateRecordsCreateCmd.Flags().StringP("name", "n", "", "Record name (@ or empty for apex)")
	templateRecordsCreateCmd.Flags().StringP("content", "c", "", "Record content/value")
	templateRecordsCreateCmd.Flags().Int("ttl", 0, "Time to live in seconds")
	templateRecordsCreateCmd.Flags().Int("priority", 0, "Record priority (for MX, SRV)")

	templateRecordsCmd.AddCommand(templateRecordsDeleteCmd)
	templateRecordsDeleteCmd.Flags().BoolP("yes", "y", false, "Confirm deletion")
}
