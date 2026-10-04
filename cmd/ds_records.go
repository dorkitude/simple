package cmd

import (
	"context"
	"fmt"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

func listDSRecords(ctx context.Context, c *dnsimple.Client, accountID, domain string, opts *dnsimple.ListOptions) ([]dnsimple.DelegationSignerRecord, error) {
	resp, err := c.Domains.ListDelegationSignerRecords(ctx, accountID, domain, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list DS records for '%s': %w", domain, err)
	}
	return resp.Data, nil
}

func getDSRecord(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) (*dnsimple.DelegationSignerRecord, error) {
	resp, err := c.Domains.GetDelegationSignerRecord(ctx, accountID, domain, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get DS record %d: %w", id, err)
	}
	return resp.Data, nil
}

func createDSRecord(ctx context.Context, c *dnsimple.Client, accountID, domain string, attrs dnsimple.DelegationSignerRecord) (*dnsimple.DelegationSignerRecord, error) {
	if attrs.Algorithm == "" {
		return nil, fmt.Errorf("--algorithm is required")
	}
	if attrs.PublicKey == "" && (attrs.Digest == "" || attrs.DigestType == "" || attrs.Keytag == "") {
		return nil, fmt.Errorf("give --digest, --digest-type and --keytag, or --public-key (for TLDs that take keys)")
	}
	resp, err := c.Domains.CreateDelegationSignerRecord(ctx, accountID, domain, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to create DS record for '%s': %w", domain, err)
	}
	return resp.Data, nil
}

func deleteDSRecord(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) error {
	if _, err := c.Domains.DeleteDelegationSignerRecord(ctx, accountID, domain, id); err != nil {
		return fmt.Errorf("failed to delete DS record %d: %w", id, err)
	}
	return nil
}

var domainsDSCmd = &cobra.Command{
	Use:     "ds",
	Aliases: []string{"ds-records"},
	Short:   "Manage delegation signer (DS) records",
	Long: `List, view, create, and delete the DS records DNSimple publishes at the
registry for a domain (used with DNSSEC hosted elsewhere).`,
}

var domainsDSListCmd = &cobra.Command{
	Use:   "list [domain]",
	Short: "List DS records for a domain",
	Long: `List the delegation signer records for a domain.

Examples:
  simple domains ds list example.com
  simple domains ds list example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		records, err := listDSRecords(ctx, app.Client, app.AccountID, args[0], pageOptions(cmd))
		if err != nil {
			return err
		}
		if printJSON(records) {
			return nil
		}
		if len(records) == 0 {
			fmt.Println(ui.Warn("No DS records found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🔏 %d DS records for %s", len(records), args[0])))
		fmt.Println()
		for _, r := range records {
			fmt.Printf("  %-10s %s %s %s %s\n",
				ui.AccentStyle.Render(fmt.Sprint(r.ID)),
				ui.SubtleStyle.Render("keytag "+r.Keytag),
				ui.SubtleStyle.Render("alg "+r.Algorithm),
				ui.SubtleStyle.Render("digest-type "+r.DigestType),
				truncate(r.Digest, 40),
			)
		}
		return nil
	},
}

var domainsDSGetCmd = &cobra.Command{
	Use:   "get [domain] [ds-id]",
	Short: "Get a DS record",
	Long: `Display one delegation signer record.

Examples:
  simple domains ds get example.com 24`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("DS record", args[1])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := getDSRecord(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		printDSRecord(args[0], r)
		return nil
	},
}

func printDSRecord(domain string, r *dnsimple.DelegationSignerRecord) {
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🔏 DS %d for %s", r.ID, domain)))
	fmt.Println()
	fmt.Printf("  %-14s %d\n", "ID:", r.ID)
	fmt.Printf("  %-14s %s\n", "Algorithm:", r.Algorithm)
	if r.Keytag != "" {
		fmt.Printf("  %-14s %s\n", "Keytag:", r.Keytag)
	}
	if r.DigestType != "" {
		fmt.Printf("  %-14s %s\n", "Digest Type:", r.DigestType)
	}
	if r.Digest != "" {
		fmt.Printf("  %-14s %s\n", "Digest:", r.Digest)
	}
	if r.PublicKey != "" {
		fmt.Printf("  %-14s %s\n", "Public Key:", r.PublicKey)
	}
	fmt.Printf("  %-14s %s\n", "Created:", r.CreatedAt)
	fmt.Printf("  %-14s %s\n", "Updated:", r.UpdatedAt)
}

var domainsDSCreateCmd = &cobra.Command{
	Use:   "create [domain]",
	Short: "Create a DS record",
	Long: `Create a delegation signer record for a domain.

Most TLDs take --digest, --digest-type and --keytag; a few take --public-key.

Examples:
  simple domains ds create example.com --algorithm 13 --digest-type 2 --keytag 2371 \
    --digest 9F4A...C2E1
  simple domains ds create example.de --algorithm 13 --public-key "mdsswUyr3..."`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		attrs := dnsimple.DelegationSignerRecord{}
		attrs.Algorithm, _ = cmd.Flags().GetString("algorithm")
		attrs.Digest, _ = cmd.Flags().GetString("digest")
		attrs.DigestType, _ = cmd.Flags().GetString("digest-type")
		attrs.Keytag, _ = cmd.Flags().GetString("keytag")
		attrs.PublicKey, _ = cmd.Flags().GetString("public-key")

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := createDSRecord(ctx, app.Client, app.AccountID, args[0], attrs)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Created DS record for '%s' (ID: %d, keytag %s)", args[0], r.ID, r.Keytag)))
		return nil
	},
}

var domainsDSDeleteCmd = &cobra.Command{
	Use:   "delete [domain] [ds-id]",
	Short: "Delete a DS record",
	Long: `Delete a delegation signer record. If the domain is still signed with
that key, resolvers may stop validating it, so this requires --yes.

Examples:
  simple domains ds delete example.com 24 --yes`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("DS record", args[1])
		if err != nil {
			return err
		}
		if err := requireYes(cmd, fmt.Sprintf("delete DS record %d from '%s' at the registry", id, args[0])); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deleteDSRecord(ctx, app.Client, app.AccountID, args[0], id); err != nil {
			return err
		}
		fmt.Println(ui.Success(fmt.Sprintf("DS record %d deleted from '%s'", id, args[0])))
		return nil
	},
}

func init() {
	domainsCmd.AddCommand(domainsDSCmd)

	domainsDSCmd.AddCommand(domainsDSListCmd)
	addPageFlags(domainsDSListCmd)

	domainsDSCmd.AddCommand(domainsDSGetCmd)

	domainsDSCmd.AddCommand(domainsDSCreateCmd)
	domainsDSCreateCmd.Flags().String("algorithm", "", "DNSSEC algorithm number (e.g. 8, 13)")
	domainsDSCreateCmd.Flags().String("digest", "", "Digest of the DNSKEY")
	domainsDSCreateCmd.Flags().String("digest-type", "", "Digest type (e.g. 2 for SHA-256)")
	domainsDSCreateCmd.Flags().String("keytag", "", "Key tag")
	domainsDSCreateCmd.Flags().String("public-key", "", "Public key (only for TLDs that require it)")

	domainsDSCmd.AddCommand(domainsDSDeleteCmd)
	addYesFlag(domainsDSDeleteCmd, "required to delete the DS record")
}
