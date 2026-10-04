package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// The SDK has no Secondary DNS service; these calls use Client.Request.

// PrimaryServer is an external primary that DNSimple transfers zones from.
type PrimaryServer struct {
	ID                   int64    `json:"id"`
	AccountID            int64    `json:"account_id"`
	Name                 string   `json:"name"`
	IP                   string   `json:"ip"`
	Port                 int      `json:"port"`
	LinkedSecondaryZones []string `json:"linked_secondary_zones"`
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
}

// PrimaryServerAttributes is the body for creating a primary server.
type PrimaryServerAttributes struct {
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port string `json:"port,omitempty"`
}

type primaryServerResponse struct {
	Data *PrimaryServer `json:"data"`
}

type primaryServersResponse struct {
	Data       []PrimaryServer      `json:"data"`
	Pagination *dnsimple.Pagination `json:"pagination,omitempty"`
}

func secondaryPath(accountID string, parts ...string) string {
	p := fmt.Sprintf("/v2/%s/secondary_dns", url.PathEscape(accountID))
	for _, part := range parts {
		p += "/" + url.PathEscape(part)
	}
	return p
}

func listPrimaryServers(ctx context.Context, c *dnsimple.Client, accountID string, page, perPage int, sort string) ([]PrimaryServer, error) {
	path := secondaryPath(accountID, "primaries")
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprint(page))
	}
	if perPage > 0 {
		q.Set("per_page", fmt.Sprint(perPage))
	}
	if sort != "" {
		q.Set("sort", sort)
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	var out primaryServersResponse
	if _, err := c.Request(ctx, http.MethodGet, path, nil, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to list primary servers: %w", err)
	}
	return out.Data, nil
}

func getPrimaryServer(ctx context.Context, c *dnsimple.Client, accountID string, id int64) (*PrimaryServer, error) {
	var out primaryServerResponse
	if _, err := c.Request(ctx, http.MethodGet, secondaryPath(accountID, "primaries", fmt.Sprint(id)), nil, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to get primary server %d: %w", id, err)
	}
	return out.Data, nil
}

func createPrimaryServer(ctx context.Context, c *dnsimple.Client, accountID string, attrs PrimaryServerAttributes) (*PrimaryServer, error) {
	if attrs.Name == "" || attrs.IP == "" {
		return nil, fmt.Errorf("a name and --ip are required")
	}
	var out primaryServerResponse
	if _, err := c.Request(ctx, http.MethodPost, secondaryPath(accountID, "primaries"), attrs, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to create primary server: %w", err)
	}
	return out.Data, nil
}

func deletePrimaryServer(ctx context.Context, c *dnsimple.Client, accountID string, id int64) error {
	if _, err := c.Request(ctx, http.MethodDelete, secondaryPath(accountID, "primaries", fmt.Sprint(id)), nil, nil, nil); err != nil {
		return fmt.Errorf("failed to delete primary server %d: %w", id, err)
	}
	return nil
}

// linkPrimaryServer links (link=true) or unlinks a primary server and a secondary zone.
func linkPrimaryServer(ctx context.Context, c *dnsimple.Client, accountID string, id int64, zone string, link bool) (*PrimaryServer, error) {
	action := "unlink"
	if link {
		action = "link"
	}
	body := map[string]string{"zone": zone}
	var out primaryServerResponse
	if _, err := c.Request(ctx, http.MethodPut, secondaryPath(accountID, "primaries", fmt.Sprint(id), action), body, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to %s primary server %d and '%s': %w", action, id, zone, err)
	}
	return out.Data, nil
}

func createSecondaryZone(ctx context.Context, c *dnsimple.Client, accountID, name string) (*dnsimple.Zone, error) {
	var out dnsimple.ZoneResponse
	if _, err := c.Request(ctx, http.MethodPost, secondaryPath(accountID, "zones"), map[string]string{"name": name}, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to create secondary zone '%s': %w", name, err)
	}
	return out.Data, nil
}

func printPrimaryServer(p *PrimaryServer) {
	fmt.Println(ui.TitleStyle.Render("🛰️  " + p.Name))
	fmt.Println()
	fmt.Printf("  %-14s %d\n", "ID:", p.ID)
	fmt.Printf("  %-14s %s\n", "Name:", p.Name)
	fmt.Printf("  %-14s %s\n", "IP:", p.IP)
	fmt.Printf("  %-14s %d\n", "Port:", p.Port)
	zones := "none"
	if len(p.LinkedSecondaryZones) > 0 {
		zones = strings.Join(p.LinkedSecondaryZones, ", ")
	}
	fmt.Printf("  %-14s %s\n", "Linked Zones:", zones)
	fmt.Printf("  %-14s %s\n", "Created:", p.CreatedAt)
	fmt.Printf("  %-14s %s\n", "Updated:", p.UpdatedAt)
}

var secondaryCmd = &cobra.Command{
	Use:     "secondary",
	Aliases: []string{"secondary-dns", "sec"},
	Short:   "Manage secondary DNS",
	Long: `Manage secondary DNS: the external primary servers DNSimple transfers
zones from, and the secondary zones linked to them.`,
}

var secondaryPrimariesCmd = &cobra.Command{
	Use:     "primaries",
	Aliases: []string{"primary"},
	Short:   "Manage primary servers",
	Long:    `List, view, create, delete, link, and unlink primary servers.`,
}

var secondaryPrimariesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List primary servers",
	Long: `List the primary servers in the account.

Examples:
  simple secondary primaries list
  simple secondary primaries list --sort name:asc --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")
		sort, _ := cmd.Flags().GetString("sort")
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		servers, err := listPrimaryServers(ctx, app.Client, app.AccountID, page, perPage, sort)
		if err != nil {
			return err
		}
		if printJSON(servers) {
			return nil
		}
		if len(servers) == 0 {
			fmt.Println(ui.Warn("No primary servers found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🛰️  %d primary servers", len(servers))))
		fmt.Println()
		for _, p := range servers {
			linked := ""
			if len(p.LinkedSecondaryZones) > 0 {
				linked = ui.SubtleStyle.Render(" → " + strings.Join(p.LinkedSecondaryZones, ", "))
			}
			fmt.Printf("  %-10s %-24s %s:%d%s\n",
				ui.SubtleStyle.Render(fmt.Sprint(p.ID)),
				ui.AccentStyle.Render(p.Name), p.IP, p.Port, linked)
		}
		return nil
	},
}

var secondaryPrimariesGetCmd = &cobra.Command{
	Use:   "get [primary-id]",
	Short: "Get a primary server",
	Long: `Display one primary server.

Examples:
  simple secondary primaries get 4`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("primary server", args[0])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		p, err := getPrimaryServer(ctx, app.Client, app.AccountID, id)
		if err != nil {
			return err
		}
		if printJSON(p) {
			return nil
		}
		printPrimaryServer(p)
		return nil
	},
}

var secondaryPrimariesCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a primary server",
	Long: `Register an external primary server that DNSimple will transfer zones from.

Examples:
  simple secondary primaries create my-primary --ip 1.2.3.4
  simple secondary primaries create my-primary --ip 1.2.3.4 --port 5353`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, _ := cmd.Flags().GetString("ip")
		port, _ := cmd.Flags().GetInt("port")
		attrs := PrimaryServerAttributes{Name: args[0], IP: ip}
		if cmd.Flags().Changed("port") {
			attrs.Port = fmt.Sprint(port)
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		p, err := createPrimaryServer(ctx, app.Client, app.AccountID, attrs)
		if err != nil {
			return err
		}
		if printJSON(p) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Primary server '%s' created at %s:%d (ID: %d)", p.Name, p.IP, p.Port, p.ID)))
		return nil
	},
}

var secondaryPrimariesDeleteCmd = &cobra.Command{
	Use:   "delete [primary-id]",
	Short: "Delete a primary server",
	Long: `PERMANENTLY delete a primary server. Requires --yes.

Examples:
  simple secondary primaries delete 4 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("primary server", args[0])
		if err != nil {
			return err
		}
		if err := requireYes(cmd, fmt.Sprintf("permanently delete primary server %d", id)); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deletePrimaryServer(ctx, app.Client, app.AccountID, id); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("Primary server %d deleted", id)))
		return nil
	},
}

func newPrimaryLinkCmd(link bool) *cobra.Command {
	verb, prep := "unlink", "from"
	if link {
		verb, prep = "link", "to"
	}
	return &cobra.Command{
		Use:   verb + " [primary-id] [zone]",
		Short: fmt.Sprintf("%s a primary server %s a secondary zone", strings.ToUpper(verb[:1])+verb[1:], prep),
		Long: fmt.Sprintf(`%s a primary server %s a secondary zone.

Examples:
  simple secondary primaries %s 4 example.com`, strings.ToUpper(verb[:1])+verb[1:], prep, verb),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID("primary server", args[0])
			if err != nil {
				return err
			}
			ctx := context.Background()
			app, err := getApp(ctx)
			if err != nil {
				return err
			}
			p, err := linkPrimaryServer(ctx, app.Client, app.AccountID, id, args[1], link)
			if err != nil {
				return err
			}
			if printJSON(p) {
				return nil
			}
			fmt.Println(ui.Success(fmt.Sprintf("Primary server %d %sed %s '%s'", id, verb, prep, args[1])))
			return nil
		},
	}
}

var secondaryZonesCmd = &cobra.Command{
	Use:     "zones",
	Aliases: []string{"zone"},
	Short:   "Manage secondary zones",
}

var secondaryZonesCreateCmd = &cobra.Command{
	Use:   "create [zone]",
	Short: "Create a secondary zone",
	Long: `Create a secondary zone. On Solo or Teams plans this enables DNS for the
zone and is charged on the next renewal invoice.

Examples:
  simple secondary zones create example.com
  simple secondary primaries link 4 example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		z, err := createSecondaryZone(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(z) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Secondary zone '%s' created (ID: %d)", z.Name, z.ID)))
		fmt.Println(ui.Info("Link it to a primary: simple secondary primaries link <primary-id> " + z.Name))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(secondaryCmd)

	secondaryCmd.AddCommand(secondaryPrimariesCmd)
	secondaryPrimariesCmd.AddCommand(secondaryPrimariesListCmd)
	addPageFlags(secondaryPrimariesListCmd)
	secondaryPrimariesListCmd.Flags().String("sort", "", "Sort order (id:asc, id:desc, name:asc, name:desc)")

	secondaryPrimariesCmd.AddCommand(secondaryPrimariesGetCmd)

	secondaryPrimariesCmd.AddCommand(secondaryPrimariesCreateCmd)
	secondaryPrimariesCreateCmd.Flags().String("ip", "", "IP address of the primary server")
	secondaryPrimariesCreateCmd.Flags().Int("port", 53, "Port of the primary server")

	secondaryPrimariesCmd.AddCommand(secondaryPrimariesDeleteCmd)
	addYesFlag(secondaryPrimariesDeleteCmd, "required to delete the primary server")

	secondaryPrimariesCmd.AddCommand(newPrimaryLinkCmd(true))
	secondaryPrimariesCmd.AddCommand(newPrimaryLinkCmd(false))

	secondaryCmd.AddCommand(secondaryZonesCmd)
	secondaryZonesCmd.AddCommand(secondaryZonesCreateCmd)
}
