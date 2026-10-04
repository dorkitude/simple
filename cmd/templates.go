package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// tplConfirm guards a command that changes live DNS or deletes data. Without
// --yes it prints what would happen and returns an error so the process exits
// non-zero; with --yes it returns nil and the caller proceeds.
func tplConfirm(cmd *cobra.Command, plan string, details ...string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}
	fmt.Println(ui.Warn("Would " + plan))
	for _, d := range details {
		fmt.Println("  " + d)
	}
	fmt.Println()
	// A refusal is expected, not a usage mistake: skip cobra's usage dump.
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	return fmt.Errorf("not confirmed: re-run with --yes to %s", plan)
}

// tplPageOptions builds SDK list options from --page / --per-page values.
func tplPageOptions(page, perPage int) *dnsimple.ListOptions {
	opts := &dnsimple.ListOptions{}
	if page > 0 {
		opts.Page = dnsimpleInt(page)
	}
	if perPage > 0 {
		opts.PerPage = dnsimpleInt(perPage)
	}
	return opts
}

func listTemplates(ctx context.Context, c *dnsimple.Client, accountID string, opts *dnsimple.ListOptions) ([]dnsimple.Template, error) {
	resp, err := c.Templates.ListTemplates(ctx, accountID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	return resp.Data, nil
}

func getTemplate(ctx context.Context, c *dnsimple.Client, accountID, template string) (*dnsimple.Template, error) {
	resp, err := c.Templates.GetTemplate(ctx, accountID, template)
	if err != nil {
		return nil, fmt.Errorf("failed to get template '%s': %w", template, err)
	}
	return resp.Data, nil
}

func createTemplate(ctx context.Context, c *dnsimple.Client, accountID, name, sid, description string) (*dnsimple.Template, error) {
	resp, err := c.Templates.CreateTemplate(ctx, accountID, dnsimple.Template{
		Name:        name,
		SID:         sid,
		Description: description,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}
	return resp.Data, nil
}

// updateTemplate PATCHes only the given fields. It builds the body itself
// because the SDK's Template has omitempty on every field, which would make
// it impossible to clear a description.
func updateTemplate(ctx context.Context, c *dnsimple.Client, accountID, template string, fields map[string]string) (*dnsimple.Template, error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("nothing to update: pass at least one of --name, --sid, --description")
	}
	path := fmt.Sprintf("/v2/%s/templates/%s", url.PathEscape(accountID), url.PathEscape(template))
	out := &dnsimple.TemplateResponse{}
	if _, err := c.Request(ctx, http.MethodPatch, path, fields, out, nil); err != nil {
		return nil, fmt.Errorf("failed to update template '%s': %w", template, err)
	}
	return out.Data, nil
}

func deleteTemplate(ctx context.Context, c *dnsimple.Client, accountID, template string) error {
	if _, err := c.Templates.DeleteTemplate(ctx, accountID, template); err != nil {
		return fmt.Errorf("failed to delete template '%s': %w", template, err)
	}
	return nil
}

func applyTemplate(ctx context.Context, c *dnsimple.Client, accountID, template, domain string) error {
	if _, err := c.Templates.ApplyTemplate(ctx, accountID, template, domain); err != nil {
		return fmt.Errorf("failed to apply template '%s' to '%s': %w", template, domain, err)
	}
	return nil
}

func printTemplate(t *dnsimple.Template) {
	fmt.Println(ui.TitleStyle.Render("📐 " + t.Name))
	fmt.Println()
	fmt.Printf("  %-14s %d\n", "ID:", t.ID)
	fmt.Printf("  %-14s %s\n", "SID:", t.SID)
	fmt.Printf("  %-14s %s\n", "Name:", t.Name)
	if t.Description != "" {
		fmt.Printf("  %-14s %s\n", "Description:", t.Description)
	}
	fmt.Printf("  %-14s %s\n", "Created:", t.CreatedAt)
	fmt.Printf("  %-14s %s\n", "Updated:", t.UpdatedAt)
}

var templatesCmd = &cobra.Command{
	Use:     "templates",
	Aliases: []string{"template", "tpl"},
	Short:   "Manage DNS templates",
	Long: `List, view, create, update, delete, and apply DNS templates.

A template is a reusable set of records that can be applied to any domain.
Templates are identified by numeric ID or by SID (short name).`,
}

var templatesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List templates",
	Long: `List all DNS templates in your account.

Examples:
  simple templates list
  simple tpl list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		templates, err := listTemplates(ctx, app.Client, app.AccountID, tplPageOptions(page, perPage))
		if err != nil {
			return err
		}
		if printJSON(templates) {
			return nil
		}
		if len(templates) == 0 {
			fmt.Println(ui.Warn("No templates found"))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📐 %d templates", len(templates))))
		fmt.Println()
		for _, t := range templates {
			fmt.Printf("  %-8d %-24s %s", t.ID, ui.AccentStyle.Render(t.SID), t.Name)
			if t.Description != "" {
				fmt.Printf("  %s", ui.SubtleStyle.Render(truncate(t.Description, 50)))
			}
			fmt.Println()
		}
		return nil
	},
}

var templatesGetCmd = &cobra.Command{
	Use:   "get <template>",
	Short: "Get template details",
	Long: `Display details for a template, by ID or SID.

Examples:
  simple templates get my-template
  simple templates get 296 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t, err := getTemplate(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		printTemplate(t)
		return nil
	},
}

var templatesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a template",
	Long: `Create a new, empty DNS template. Add records with 'templates records create'.

Examples:
  simple templates create --name "Google Workspace" --sid google-workspace
  simple templates create --name "Web" --sid web --description "A + www CNAME"`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		sid, _ := cmd.Flags().GetString("sid")
		description, _ := cmd.Flags().GetString("description")
		if name == "" || sid == "" {
			return fmt.Errorf("--name and --sid are required")
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t, err := createTemplate(ctx, app.Client, app.AccountID, name, sid, description)
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Template '%s' created (SID: %s, ID: %d)", t.Name, t.SID, t.ID)))
		return nil
	},
}

var templatesUpdateCmd = &cobra.Command{
	Use:   "update <template>",
	Short: "Update a template",
	Long: `Update a template's name, SID, or description. Only flags you pass are changed.

Examples:
  simple templates update web --name "Website"
  simple templates update web --description ""`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fields := map[string]string{}
		for _, f := range []string{"name", "sid", "description"} {
			if cmd.Flags().Changed(f) {
				v, _ := cmd.Flags().GetString(f)
				fields[f] = v
			}
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t, err := updateTemplate(ctx, app.Client, app.AccountID, args[0], fields)
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Template '%s' updated (SID: %s)", t.Name, t.SID)))
		return nil
	},
}

var templatesDeleteCmd = &cobra.Command{
	Use:   "delete <template>",
	Short: "Delete a template",
	Long: `PERMANENTLY delete a template and its records. Requires --yes.
Domains the template was already applied to keep their records.

Examples:
  simple templates delete web          # shows what would happen
  simple templates delete web --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		template := args[0]
		if err := tplConfirm(cmd, fmt.Sprintf("permanently delete template '%s'", template)); err != nil {
			return err
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deleteTemplate(ctx, app.Client, app.AccountID, template); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"template": template, "deleted": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Template '%s' deleted.", template)))
		return nil
	},
}

var templatesApplyCmd = &cobra.Command{
	Use:   "apply <template> <domain>",
	Short: "Apply a template to a domain",
	Long: `Apply a template to a domain, adding the template's records to the live zone.
Requires --yes; without it, lists the records that would be added.

Examples:
  simple templates apply web example.com          # preview
  simple templates apply web example.com --yes`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		template, domain := args[0], args[1]

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		if yes, _ := cmd.Flags().GetBool("yes"); !yes {
			// Read-only preview of the records the template would add.
			var details []string
			records, err := listTemplateRecords(ctx, app.Client, app.AccountID, template, nil)
			if err != nil {
				return err
			}
			for _, r := range records {
				details = append(details, formatTemplateRecordLine(r))
			}
			return tplConfirm(cmd, fmt.Sprintf("apply template '%s' to '%s' (%d records added to live DNS)", template, domain, len(records)), details...)
		}

		if err := applyTemplate(ctx, app.Client, app.AccountID, template, domain); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"template": template, "domain": domain, "applied": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Template '%s' applied to '%s'", template, domain)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(templatesCmd)

	templatesCmd.AddCommand(templatesListCmd)
	templatesListCmd.Flags().Int("page", 0, "Page number")
	templatesListCmd.Flags().Int("per-page", 0, "Results per page")

	templatesCmd.AddCommand(templatesGetCmd)

	templatesCmd.AddCommand(templatesCreateCmd)
	templatesCreateCmd.Flags().StringP("name", "n", "", "Template name (required)")
	templatesCreateCmd.Flags().String("sid", "", "Template short name / SID (required)")
	templatesCreateCmd.Flags().StringP("description", "d", "", "Template description")

	templatesCmd.AddCommand(templatesUpdateCmd)
	templatesUpdateCmd.Flags().StringP("name", "n", "", "New template name")
	templatesUpdateCmd.Flags().String("sid", "", "New template SID")
	templatesUpdateCmd.Flags().StringP("description", "d", "", "New description (empty clears it)")

	templatesCmd.AddCommand(templatesDeleteCmd)
	templatesDeleteCmd.Flags().BoolP("yes", "y", false, "Confirm deletion")

	templatesCmd.AddCommand(templatesApplyCmd)
	templatesApplyCmd.Flags().BoolP("yes", "y", false, "Confirm applying to live DNS")
}
