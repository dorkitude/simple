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

// zoneNSUpdate is the body of PUT /v2/{account}/zones/{zone}/ns_records.
// The SDK has no method for this endpoint.
type zoneNSUpdate struct {
	NSNames  []string `json:"ns_names,omitempty"`
	NSSetIDs []int64  `json:"ns_set_ids,omitempty"`
}

// updateZoneNSRecords replaces the zone's apex NS records.
func updateZoneNSRecords(ctx context.Context, c *dnsimple.Client, accountID, zone string, body zoneNSUpdate) ([]dnsimple.ZoneRecord, error) {
	if len(body.NSNames) == 0 && len(body.NSSetIDs) == 0 {
		return nil, fmt.Errorf("give at least one name server or --ns-set-id")
	}
	path := fmt.Sprintf("/v2/%s/zones/%s/ns_records", url.PathEscape(accountID), url.PathEscape(zone))
	var out dnsimple.ZoneRecordsResponse
	if _, err := c.Request(ctx, http.MethodPut, path, body, &out, nil); err != nil {
		return nil, fmt.Errorf("failed to update NS records for '%s': %w", zone, err)
	}
	return out.Data, nil
}

var zonesNSCmd = &cobra.Command{
	Use:     "ns [zone] [name-server...]",
	Aliases: []string{"ns-records", "nameservers"},
	Short:   "Replace a zone's NS records",
	Long: `Replace the apex NS records in a zone with the given name servers and/or
name server sets. This changes how the zone is delegated inside DNSimple's
answers, so it requires --yes.

Examples:
  simple zones ns example.com ns1.dnsimple.com ns2.dnsimple-edge.net --yes
  simple zones ns example.com --ns-set-id 1,2 --yes`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zone := args[0]
		setIDs, _ := cmd.Flags().GetInt64Slice("ns-set-id")
		body := zoneNSUpdate{NSNames: args[1:], NSSetIDs: setIDs}
		if len(body.NSNames) == 0 && len(body.NSSetIDs) == 0 {
			return fmt.Errorf("give at least one name server or --ns-set-id")
		}
		plan := []string{}
		if len(body.NSNames) > 0 {
			plan = append(plan, fmt.Sprintf("set the NS records of '%s' to: %s", zone, strings.Join(body.NSNames, ", ")))
		}
		if len(body.NSSetIDs) > 0 {
			plan = append(plan, fmt.Sprintf("use name server sets %v for '%s'", body.NSSetIDs, zone))
		}
		plan = append(plan, "replace every existing apex NS record")
		if err := requireYes(cmd, plan...); err != nil {
			return err
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		records, err := updateZoneNSRecords(ctx, app.Client, app.AccountID, zone, body)
		if err != nil {
			return err
		}
		if printJSON(records) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("NS records updated for '%s'", zone)))
		fmt.Println()
		for _, r := range records {
			fmt.Printf("  %s %-6d %s\n", ui.RecordTypeStyle.Render(r.Type), r.TTL, r.Content)
		}
		return nil
	},
}

func init() {
	zonesCmd.AddCommand(zonesNSCmd)
	zonesNSCmd.Flags().Int64Slice("ns-set-id", nil, "Name server set IDs (comma-separated)")
	addYesFlag(zonesNSCmd, "required to replace the NS records")
}
