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

// RecordFields is the create/update body for a zone record. Unlike the SDK's
// ZoneRecordAttributes, whose int fields are omitempty, TTL and Priority are
// pointers so an explicit 0 (an MX record with priority 0) reaches the API.
// It also serves as one entry of a batch change (ID set only for updates).
type RecordFields struct {
	ID       int64    `json:"id,omitempty"`
	Type     string   `json:"type,omitempty"`
	Name     *string  `json:"name,omitempty"`
	Content  string   `json:"content,omitempty"`
	TTL      *int     `json:"ttl,omitempty"`
	Priority *int     `json:"priority,omitempty"`
	Regions  []string `json:"regions,omitempty"`
}

// recordFieldsFromFlags reads --ttl, --priority and --regions, keeping only
// the flags the user actually set (so 0 is a real value, not "unset").
func recordFieldsFromFlags(cmd *cobra.Command) RecordFields {
	var attrs RecordFields
	if cmd.Flags().Changed("ttl") {
		ttl, _ := cmd.Flags().GetInt("ttl")
		attrs.TTL = &ttl
	}
	if cmd.Flags().Changed("priority") {
		priority, _ := cmd.Flags().GetInt("priority")
		attrs.Priority = &priority
	}
	if cmd.Flags().Changed("regions") {
		regions, _ := cmd.Flags().GetStringSlice("regions")
		attrs.Regions = regions
	}
	return attrs
}

func zoneRecordsPath(accountID, zone string) string {
	return fmt.Sprintf("/v2/%s/zones/%s/records", url.PathEscape(accountID), url.PathEscape(zone))
}

// createZoneRecord posts a record directly (not via Zones.CreateRecord) so a
// zero TTL or priority is not dropped by the SDK's omitempty tags.
func createZoneRecord(ctx context.Context, c *dnsimple.Client, accountID, zone string, attrs RecordFields) (*dnsimple.ZoneRecordResponse, error) {
	attrs.ID = 0
	out := &dnsimple.ZoneRecordResponse{}
	if _, err := c.Request(ctx, http.MethodPost, zoneRecordsPath(accountID, zone), attrs, out, nil); err != nil {
		return nil, fmt.Errorf("failed to create record: %w", err)
	}
	return out, nil
}

// updateZoneRecord patches a record directly; see createZoneRecord.
func updateZoneRecord(ctx context.Context, c *dnsimple.Client, accountID, zone string, recordID int64, attrs RecordFields) (*dnsimple.ZoneRecordResponse, error) {
	attrs.ID = 0
	out := &dnsimple.ZoneRecordResponse{}
	path := fmt.Sprintf("%s/%d", zoneRecordsPath(accountID, zone), recordID)
	if _, err := c.Request(ctx, http.MethodPatch, path, attrs, out, nil); err != nil {
		return nil, fmt.Errorf("failed to update record: %w", err)
	}
	return out, nil
}

var recordsCmd = &cobra.Command{
	Use:     "records",
	Aliases: []string{"record", "rec"},
	Short:   "Manage DNS records",
	Long:    `List, view, create, update, and delete DNS records for a zone.`,
}

var recordsListCmd = &cobra.Command{
	Use:   "list [zone]",
	Short: "List records for a zone",
	Long: `List all DNS records for a zone.

Examples:
  simple records list example.com
  simple records list example.com --type A
  simple records list example.com --name www
  simple records list example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		zone := args[0]
		nameFilter, _ := cmd.Flags().GetString("name")
		typeFilter, _ := cmd.Flags().GetString("type")
		page, _ := cmd.Flags().GetInt("page")
		perPage, _ := cmd.Flags().GetInt("per-page")

		opts := &dnsimple.ZoneRecordListOptions{}
		if nameFilter != "" {
			opts.Name = dnsimpleString(nameFilter)
		}
		if typeFilter != "" {
			opts.Type = dnsimpleString(typeFilter)
		}
		if page > 0 {
			opts.Page = dnsimpleInt(page)
		}
		if perPage > 0 {
			opts.PerPage = dnsimpleInt(perPage)
		}

		resp, err := app.Client.Zones.ListRecords(ctx, app.AccountID, zone, opts)
		if err != nil {
			return fmt.Errorf("failed to list records: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		if len(resp.Data) == 0 {
			fmt.Println(ui.Warn("No records found"))
			return nil
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📋 %d records for %s", len(resp.Data), zone)))
		fmt.Println()

		for _, r := range resp.Data {
			name := r.Name
			if name == "" {
				name = "@"
			}

			sysMarker := ""
			if r.SystemRecord {
				sysMarker = ui.SubtleStyle.Render(" [sys]")
			}

			priorityStr := ""
			if r.Priority != 0 {
				priorityStr = fmt.Sprintf(" (pri: %d)", r.Priority)
			}

			fmt.Printf("  %s %-20s %-6d %s%s%s\n",
				ui.RecordTypeStyle.Render(r.Type),
				ui.AccentStyle.Render(name),
				r.TTL,
				truncate(r.Content, 50),
				priorityStr,
				sysMarker,
			)
		}

		return nil
	},
}

var recordsGetCmd = &cobra.Command{
	Use:   "get [zone] [record-id]",
	Short: "Get record details",
	Long:  `Display detailed information about a specific DNS record.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		zone := args[0]
		recordID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		resp, err := app.Client.Zones.GetRecord(ctx, app.AccountID, zone, recordID)
		if err != nil {
			return fmt.Errorf("failed to get record: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		r := resp.Data
		name := r.Name
		if name == "" {
			name = "@"
		}

		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("📋 %s %s.%s", r.Type, name, zone)))
		fmt.Println()
		fmt.Printf("  %-14s %d\n", "ID:", r.ID)
		fmt.Printf("  %-14s %s\n", "Type:", r.Type)
		fmt.Printf("  %-14s %s\n", "Name:", name)
		fmt.Printf("  %-14s %s\n", "Content:", r.Content)
		fmt.Printf("  %-14s %d\n", "TTL:", r.TTL)
		if r.Priority != 0 {
			fmt.Printf("  %-14s %d\n", "Priority:", r.Priority)
		}
		if len(r.Regions) > 0 {
			fmt.Printf("  %-14s %v\n", "Regions:", r.Regions)
		}
		fmt.Printf("  %-14s %v\n", "System:", r.SystemRecord)
		fmt.Printf("  %-14s %s\n", "Created:", r.CreatedAt)
		fmt.Printf("  %-14s %s\n", "Updated:", r.UpdatedAt)

		return nil
	},
}

var recordsCreateCmd = &cobra.Command{
	Use:   "create [zone]",
	Short: "Create a DNS record",
	Long: `Create a new DNS record in a zone.

Examples:
  simple records create example.com --type A --name www --content 1.2.3.4
  simple records create example.com --type CNAME --name blog --content example.com
  simple records create example.com --type MX --name "" --content mail.example.com --priority 10
  simple records create example.com --type MX --name "" --content mx.example.com --priority 0
  simple records create example.com --type A --name geo --content 1.2.3.4 --regions SV1,IAD
  simple records create example.com --type TXT --name @ --content "v=spf1 include:_spf.google.com ~all"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		zone := args[0]
		recordType, _ := cmd.Flags().GetString("type")
		name, _ := cmd.Flags().GetString("name")
		content, _ := cmd.Flags().GetString("content")

		if recordType == "" || content == "" {
			return fmt.Errorf("--type and --content are required")
		}

		attrs := recordFieldsFromFlags(cmd)
		attrs.Type = recordType
		attrs.Name = dnsimpleString(name)
		attrs.Content = content

		resp, err := createZoneRecord(ctx, app.Client, app.AccountID, zone, attrs)
		if err != nil {
			return err
		}

		if printJSON(resp.Data) {
			return nil
		}

		r := resp.Data
		displayName := r.Name
		if displayName == "" {
			displayName = "@"
		}
		fmt.Println(ui.Success(fmt.Sprintf("Created %s record '%s.%s' → %s (ID: %d)",
			r.Type, displayName, zone, r.Content, r.ID)))

		return nil
	},
}

var recordsUpdateCmd = &cobra.Command{
	Use:   "update [zone] [record-id]",
	Short: "Update a DNS record",
	Long: `Update an existing DNS record.

Examples:
  simple records update example.com 12345 --content 5.6.7.8
  simple records update example.com 12345 --name www2 --ttl 600
  simple records update example.com 12345 --priority 0
  simple records update example.com 12345 --regions global`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		zone := args[0]
		recordID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		attrs := recordFieldsFromFlags(cmd)
		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			attrs.Name = dnsimpleString(name)
		}
		if cmd.Flags().Changed("content") {
			content, _ := cmd.Flags().GetString("content")
			attrs.Content = content
		}

		resp, err := updateZoneRecord(ctx, app.Client, app.AccountID, zone, recordID, attrs)
		if err != nil {
			return err
		}

		if printJSON(resp.Data) {
			return nil
		}

		r := resp.Data
		displayName := r.Name
		if displayName == "" {
			displayName = "@"
		}
		fmt.Println(ui.Success(fmt.Sprintf("Updated %s record '%s.%s' → %s",
			r.Type, displayName, zone, r.Content)))

		return nil
	},
}

var recordsDeleteCmd = &cobra.Command{
	Use:   "delete [zone] [record-id]",
	Short: "Delete a DNS record",
	Long: `PERMANENTLY delete a DNS record from a zone.

Examples:
  simple records delete example.com 12345`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		zone := args[0]
		recordID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		_, err = app.Client.Zones.DeleteRecord(ctx, app.AccountID, zone, recordID)
		if err != nil {
			return fmt.Errorf("failed to delete record: %w", err)
		}

		fmt.Println(ui.Success(fmt.Sprintf("Record %d deleted from zone '%s'", recordID, zone)))
		return nil
	},
}

var recordsDistributionCmd = &cobra.Command{
	Use:   "distribution [zone] [record-id]",
	Short: "Check record distribution",
	Long: `Check if a record is fully distributed across DNSimple name servers.

Examples:
  simple records distribution example.com 12345`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}

		zone := args[0]
		recordID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		resp, err := app.Client.Zones.CheckZoneRecordDistribution(ctx, app.AccountID, zone, recordID)
		if err != nil {
			return fmt.Errorf("failed to check distribution: %w", err)
		}

		if printJSON(resp.Data) {
			return nil
		}

		if resp.Data.Distributed {
			fmt.Println(ui.Success(fmt.Sprintf("Record %d is fully distributed ✨", recordID)))
		} else {
			fmt.Println(ui.Warn(fmt.Sprintf("Record %d is NOT fully distributed yet", recordID)))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(recordsCmd)

	recordsCmd.AddCommand(recordsListCmd)
	recordsListCmd.Flags().String("name", "", "Filter by record name")
	recordsListCmd.Flags().String("type", "", "Filter by record type (A, AAAA, CNAME, MX, etc.)")
	recordsListCmd.Flags().Int("page", 0, "Page number")
	recordsListCmd.Flags().Int("per-page", 0, "Results per page")

	recordsCmd.AddCommand(recordsGetCmd)

	recordsCmd.AddCommand(recordsCreateCmd)
	recordsCreateCmd.Flags().StringP("type", "t", "", "Record type (A, AAAA, CNAME, MX, TXT, etc.)")
	recordsCreateCmd.Flags().StringP("name", "n", "", "Record name (@ or empty for apex)")
	recordsCreateCmd.Flags().StringP("content", "c", "", "Record content/value")
	recordsCreateCmd.Flags().Int("ttl", 0, "Time to live in seconds")
	recordsCreateCmd.Flags().Int("priority", 0, "Record priority (for MX, SRV); 0 is sent when given")
	recordsCreateCmd.Flags().StringSlice("regions", nil, "Regions to serve the record from (comma-separated, e.g. SV1,IAD or global)")

	recordsCmd.AddCommand(recordsUpdateCmd)
	recordsUpdateCmd.Flags().StringP("name", "n", "", "New record name")
	recordsUpdateCmd.Flags().StringP("content", "c", "", "New record content")
	recordsUpdateCmd.Flags().Int("ttl", 0, "New TTL")
	recordsUpdateCmd.Flags().Int("priority", 0, "New priority (0 is allowed)")
	recordsUpdateCmd.Flags().StringSlice("regions", nil, "New regions (comma-separated, e.g. SV1,IAD or global)")

	recordsCmd.AddCommand(recordsDeleteCmd)
	recordsCmd.AddCommand(recordsDistributionCmd)
}
