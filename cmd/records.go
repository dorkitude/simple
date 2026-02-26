package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

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
		ttl, _ := cmd.Flags().GetInt("ttl")
		priority, _ := cmd.Flags().GetInt("priority")

		if recordType == "" || content == "" {
			return fmt.Errorf("--type and --content are required")
		}

		attrs := dnsimple.ZoneRecordAttributes{
			Type:    recordType,
			Name:    dnsimpleString(name),
			Content: content,
		}
		if ttl > 0 {
			attrs.TTL = ttl
		}
		if priority > 0 {
			attrs.Priority = priority
		}

		resp, err := app.Client.Zones.CreateRecord(ctx, app.AccountID, zone, attrs)
		if err != nil {
			return fmt.Errorf("failed to create record: %w", err)
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
  simple records update example.com 12345 --name www2 --ttl 600`,
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

		attrs := dnsimple.ZoneRecordAttributes{}

		if cmd.Flags().Changed("name") {
			name, _ := cmd.Flags().GetString("name")
			attrs.Name = dnsimpleString(name)
		}
		if cmd.Flags().Changed("content") {
			content, _ := cmd.Flags().GetString("content")
			attrs.Content = content
		}
		if cmd.Flags().Changed("ttl") {
			ttl, _ := cmd.Flags().GetInt("ttl")
			attrs.TTL = ttl
		}
		if cmd.Flags().Changed("priority") {
			priority, _ := cmd.Flags().GetInt("priority")
			attrs.Priority = priority
		}

		resp, err := app.Client.Zones.UpdateRecord(ctx, app.AccountID, zone, recordID, attrs)
		if err != nil {
			return fmt.Errorf("failed to update record: %w", err)
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
	recordsCreateCmd.Flags().Int("priority", 0, "Record priority (for MX, SRV)")

	recordsCmd.AddCommand(recordsUpdateCmd)
	recordsUpdateCmd.Flags().StringP("name", "n", "", "New record name")
	recordsUpdateCmd.Flags().StringP("content", "c", "", "New record content")
	recordsUpdateCmd.Flags().Int("ttl", 0, "New TTL")
	recordsUpdateCmd.Flags().Int("priority", 0, "New priority")

	recordsCmd.AddCommand(recordsDeleteCmd)
	recordsCmd.AddCommand(recordsDistributionCmd)
}
