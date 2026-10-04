package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// parseServiceSettings turns repeated --setting k=v flags into a map.
func parseServiceSettings(pairs []string) (map[string]string, error) {
	out := map[string]string{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --setting %q (use key=value)", p)
		}
		out[k] = v
	}
	return out, nil
}

func listServices(ctx context.Context, c *dnsimple.Client, opts *dnsimple.ListOptions) ([]dnsimple.Service, error) {
	resp, err := c.Services.ListServices(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}
	return resp.Data, nil
}

func getService(ctx context.Context, c *dnsimple.Client, service string) (*dnsimple.Service, error) {
	resp, err := c.Services.GetService(ctx, service)
	if err != nil {
		return nil, fmt.Errorf("failed to get service '%s': %w", service, err)
	}
	return resp.Data, nil
}

func listAppliedServices(ctx context.Context, c *dnsimple.Client, accountID, domain string, opts *dnsimple.ListOptions) ([]dnsimple.Service, error) {
	resp, err := c.Services.AppliedServices(ctx, accountID, domain, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list services applied to '%s': %w", domain, err)
	}
	return resp.Data, nil
}

// applyService applies a one-click service to a domain. It posts the body
// itself because the SDK's DomainServiceSettings has no json tag and would
// serialize as {"Settings": ...}, which the API does not read.
func applyService(ctx context.Context, c *dnsimple.Client, accountID, service, domain string, settings map[string]string) error {
	path := fmt.Sprintf("/v2/%s/domains/%s/services/%s",
		url.PathEscape(accountID), url.PathEscape(domain), url.PathEscape(service))
	var payload interface{}
	if len(settings) > 0 {
		payload = map[string]interface{}{"settings": settings}
	}
	if _, err := c.Request(ctx, http.MethodPost, path, payload, nil, nil); err != nil {
		return fmt.Errorf("failed to apply service '%s' to '%s': %w", service, domain, err)
	}
	return nil
}

func unapplyService(ctx context.Context, c *dnsimple.Client, accountID, service, domain string) error {
	if _, err := c.Services.UnapplyService(ctx, accountID, service, domain); err != nil {
		return fmt.Errorf("failed to unapply service '%s' from '%s': %w", service, domain, err)
	}
	return nil
}

func printServiceList(title string, services []dnsimple.Service) {
	fmt.Println(ui.TitleStyle.Render(title))
	fmt.Println()
	for _, s := range services {
		fmt.Printf("  %-28s %s", ui.AccentStyle.Render(s.SID), s.Name)
		if s.RequiresSetup {
			fmt.Printf("  %s", ui.SubtleStyle.Render("[settings]"))
		}
		fmt.Println()
	}
}

var servicesCmd = &cobra.Command{
	Use:     "services",
	Aliases: []string{"service", "svc"},
	Short:   "Browse and apply one-click services",
	Long: `Browse DNSimple one-click services and apply or remove them on domains.

Applying a service adds its records to the domain's live zone.`,
}

var servicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available one-click services",
	Long: `List all one-click services DNSimple offers.

Examples:
  simple services list
  simple svc list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		services, err := listServices(ctx, app.Client, tplPageOptions(page, perPage))
		if err != nil {
			return err
		}
		if printJSON(services) {
			return nil
		}
		if len(services) == 0 {
			fmt.Println(ui.Warn("No services found"))
			return nil
		}
		printServiceList(fmt.Sprintf("🧩 %d services", len(services)), services)
		return nil
	},
}

var servicesGetCmd = &cobra.Command{
	Use:   "get <service>",
	Short: "Get service details",
	Long: `Show a one-click service and the settings it needs, by ID or SID.

Examples:
  simple services get google-apps
  simple services get heroku --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		s, err := getService(ctx, app.Client, args[0])
		if err != nil {
			return err
		}
		if printJSON(s) {
			return nil
		}
		fmt.Println(ui.TitleStyle.Render("🧩 " + s.Name))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", s.ID)
		fmt.Printf("  %-14s %s\n", "SID:", s.SID)
		if s.Description != "" {
			fmt.Printf("  %-14s %s\n", "Description:", s.Description)
		}
		if s.DefaultSubdomain != "" {
			fmt.Printf("  %-14s %s\n", "Subdomain:", s.DefaultSubdomain)
		}
		fmt.Printf("  %-14s %v\n", "Needs Setup:", s.RequiresSetup)
		if s.SetupDescription != "" {
			fmt.Printf("  %-14s %s\n", "Setup:", s.SetupDescription)
		}
		if len(s.Settings) > 0 {
			fmt.Println()
			fmt.Println(ui.SubtleStyle.Render("  Settings (pass with --setting name=value):"))
			for _, st := range s.Settings {
				fmt.Printf("    %-20s %s", ui.AccentStyle.Render(st.Name), st.Label)
				if st.Example != "" {
					fmt.Printf("  %s", ui.SubtleStyle.Render("e.g. "+st.Example))
				}
				fmt.Println()
			}
		}
		return nil
	},
}

var servicesAppliedCmd = &cobra.Command{
	Use:   "applied <domain>",
	Short: "List services applied to a domain",
	Long: `List the one-click services currently applied to a domain.

Examples:
  simple services applied example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		services, err := listAppliedServices(ctx, app.Client, app.AccountID, args[0], tplPageOptions(page, perPage))
		if err != nil {
			return err
		}
		if printJSON(services) {
			return nil
		}
		if len(services) == 0 {
			fmt.Println(ui.Info("No services applied to '" + args[0] + "'"))
			return nil
		}
		printServiceList(fmt.Sprintf("🧩 %d services applied to %s", len(services), args[0]), services)
		return nil
	},
}

var servicesApplyCmd = &cobra.Command{
	Use:   "apply <service> <domain>",
	Short: "Apply a one-click service to a domain",
	Long: `Apply a one-click service to a domain, adding its records to live DNS.
Requires --yes. See 'services get <service>' for the settings it needs.

Examples:
  simple services apply google-apps example.com          # preview
  simple services apply google-apps example.com --yes
  simple services apply heroku example.com --setting apexdomain=apex.herokudns.com --setting wwwdomain=www.herokudns.com --yes`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, domain := args[0], args[1]
		pairs, _ := cmd.Flags().GetStringArray("setting")
		settings, err := parseServiceSettings(pairs)
		if err != nil {
			return err
		}

		var details []string
		keys := make([]string, 0, len(settings))
		for k := range settings {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			details = append(details, fmt.Sprintf("%-14s %s", k+":", settings[k]))
		}
		if err := tplConfirm(cmd, fmt.Sprintf("apply service '%s' to '%s' (adds records to live DNS)", service, domain), details...); err != nil {
			return err
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := applyService(ctx, app.Client, app.AccountID, service, domain, settings); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"service": service, "domain": domain, "applied": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Service '%s' applied to '%s'", service, domain)))
		return nil
	},
}

var servicesUnapplyCmd = &cobra.Command{
	Use:   "unapply <service> <domain>",
	Short: "Remove a one-click service from a domain",
	Long: `Remove a one-click service from a domain, deleting its records from live DNS.
Requires --yes.

Examples:
  simple services unapply google-apps example.com          # preview
  simple services unapply google-apps example.com --yes`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		service, domain := args[0], args[1]
		if err := tplConfirm(cmd, fmt.Sprintf("unapply service '%s' from '%s' (removes its records from live DNS)", service, domain)); err != nil {
			return err
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := unapplyService(ctx, app.Client, app.AccountID, service, domain); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"service": service, "domain": domain, "unapplied": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Service '%s' removed from '%s'", service, domain)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(servicesCmd)

	servicesCmd.AddCommand(servicesListCmd)
	servicesListCmd.Flags().Int("page", 0, "Page number")
	servicesListCmd.Flags().Int("per-page", 0, "Results per page")

	servicesCmd.AddCommand(servicesGetCmd)

	servicesCmd.AddCommand(servicesAppliedCmd)
	servicesAppliedCmd.Flags().Int("page", 0, "Page number")
	servicesAppliedCmd.Flags().Int("per-page", 0, "Results per page")

	servicesCmd.AddCommand(servicesApplyCmd)
	servicesApplyCmd.Flags().StringArray("setting", nil, "Service setting as key=value (repeatable)")
	servicesApplyCmd.Flags().BoolP("yes", "y", false, "Confirm applying to live DNS")

	servicesCmd.AddCommand(servicesUnapplyCmd)
	servicesUnapplyCmd.Flags().BoolP("yes", "y", false, "Confirm removing from live DNS")
}
