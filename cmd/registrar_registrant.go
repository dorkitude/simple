package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

func regListRegistrantChanges(ctx context.Context, c *dnsimple.Client, accountID string, opts *dnsimple.RegistrantChangeListOptions) (*dnsimple.RegistrantChangesListResponse, error) {
	resp, err := c.Registrar.ListRegistrantChange(ctx, accountID, opts)
	if err != nil {
		return nil, regError("list registrant changes", err)
	}
	return resp, nil
}

func regGetRegistrantChange(ctx context.Context, c *dnsimple.Client, accountID string, id int) (*dnsimple.RegistrantChange, error) {
	resp, err := c.Registrar.GetRegistrantChange(ctx, accountID, id)
	if err != nil {
		return nil, regError(fmt.Sprintf("get registrant change %d", id), err)
	}
	return resp.Data, nil
}

func regCheckRegistrantChange(ctx context.Context, c *dnsimple.Client, accountID, domain, contactID string) (*dnsimple.RegistrantChangeCheck, error) {
	resp, err := c.Registrar.CheckRegistrantChange(ctx, accountID, &dnsimple.CheckRegistrantChangeInput{DomainId: domain, ContactId: contactID})
	if err != nil {
		return nil, regError(fmt.Sprintf("check registrant change for '%s'", domain), err)
	}
	return resp.Data, nil
}

func regCreateRegistrantChange(ctx context.Context, c *dnsimple.Client, accountID, domain, contactID string, attrs map[string]string) (*dnsimple.RegistrantChange, error) {
	if attrs == nil {
		attrs = map[string]string{}
	}
	input := &dnsimple.CreateRegistrantChangeInput{DomainId: domain, ContactId: contactID, ExtendedAttributes: attrs}
	resp, err := c.Registrar.CreateRegistrantChange(ctx, accountID, input)
	if err != nil {
		return nil, regError(fmt.Sprintf("start registrant change for '%s'", domain), err)
	}
	return resp.Data, nil
}

func regDeleteRegistrantChange(ctx context.Context, c *dnsimple.Client, accountID string, id int) error {
	if _, err := c.Registrar.DeleteRegistrantChange(ctx, accountID, id); err != nil {
		return regError(fmt.Sprintf("cancel registrant change %d", id), err)
	}
	return nil
}

func regStateStyle(state string) string {
	switch state {
	case "completed":
		return ui.SuccessStyle.Render(state)
	case "cancelled", "cancelling":
		return ui.SubtleStyle.Render(state)
	}
	return ui.WarningStyle.Render(state)
}

func printRegistrantChanges(w io.Writer, changes []dnsimple.RegistrantChange) {
	if len(changes) == 0 {
		fmt.Fprintln(w, ui.Warn("No registrant changes found"))
		return
	}
	fmt.Fprintln(w, ui.TitleStyle.Render(fmt.Sprintf("👤 %d registrant changes", len(changes))))
	for _, rc := range changes {
		fmt.Fprintf(w, "  %-8s %-24s %s  %s\n",
			ui.AccentStyle.Render(strconv.Itoa(rc.Id)),
			fmt.Sprintf("domain %d → contact %d", rc.DomainId, rc.ContactId),
			regStateStyle(rc.State),
			ui.SubtleStyle.Render(rc.CreatedAt),
		)
	}
}

func printRegistrantChange(w io.Writer, rc *dnsimple.RegistrantChange) {
	fmt.Fprintln(w, ui.TitleStyle.Render(fmt.Sprintf("👤 Registrant change %d", rc.Id)))
	fmt.Fprintf(w, "  %-14s %d\n", "ID:", rc.Id)
	fmt.Fprintf(w, "  %-14s %d\n", "Domain ID:", rc.DomainId)
	fmt.Fprintf(w, "  %-14s %d\n", "Contact ID:", rc.ContactId)
	fmt.Fprintf(w, "  %-14s %s\n", "State:", regStateStyle(rc.State))
	fmt.Fprintf(w, "  %-14s %s\n", "Owner change:", regYesNo(rc.RegistryOwnerChange))
	if rc.IrtLockLiftedBy != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "IRT lifted:", rc.IrtLockLiftedBy)
	}
	keys := make([]string, 0, len(rc.ExtendedAttributes))
	for k := range rc.ExtendedAttributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "  %-14s %s\n", "Attr "+k+":", rc.ExtendedAttributes[k])
	}
	regPrintTimes(w, rc.CreatedAt, rc.UpdatedAt)
}

func printRegistrantChangeCheck(w io.Writer, domain string, chk *dnsimple.RegistrantChangeCheck) {
	fmt.Fprintln(w, ui.TitleStyle.Render("👤 Registrant change check "+domain))
	fmt.Fprintf(w, "  %-14s %d\n", "Domain ID:", chk.DomainId)
	fmt.Fprintf(w, "  %-14s %d\n", "Contact ID:", chk.ContactId)
	fmt.Fprintf(w, "  %-14s %s\n", "Owner change:", regYesNo(chk.RegistryOwnerChange))
	if len(chk.ExtendedAttributes) == 0 {
		fmt.Fprintln(w, ui.Info("No extended attributes needed"))
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.SubtleStyle.Render("  Extended attributes (pass with --extended-attribute name=value):"))
	for _, a := range chk.ExtendedAttributes {
		req := ""
		if a.Required {
			req = ui.WarningStyle.Render(" (required)")
		}
		fmt.Fprintf(w, "  %s%s  %s\n", ui.AccentStyle.Render(a.Name), req, ui.SubtleStyle.Render(a.Description))
		var opts []string
		for _, o := range a.Options {
			opts = append(opts, o.Value)
		}
		if len(opts) > 0 {
			fmt.Fprintf(w, "    %s\n", ui.SubtleStyle.Render("options: "+strings.Join(opts, ", ")))
		}
	}
}

func parseRegIntID(kind, s string) (int, error) {
	id, err := parseRegID(kind, s)
	return int(id), err
}

var registrarRegistrantChangesCmd = &cobra.Command{
	Use:     "registrant-changes",
	Aliases: []string{"registrant-change", "registrant", "rc"},
	Short:   "Change the registrant (owner contact) of registered domains",
	Long: `List, check, start, inspect, and cancel registrant changes.

Examples:
  simple registrar registrant-changes list
  simple registrar registrant-changes check example.com --contact-id 42
  simple registrar registrant-changes create example.com --contact-id 42 --yes
  simple registrar registrant-changes get 101
  simple registrar registrant-changes delete 101`,
}

var registrarRCListCmd = &cobra.Command{
	Use:   "list",
	Short: "List registrant changes",
	Long: `List registrant changes in the account.

Examples:
  simple registrar registrant-changes list
  simple registrar registrant-changes list --state new --domain example.com
  simple registrar registrant-changes list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		opts := &dnsimple.RegistrantChangeListOptions{}
		if v, _ := cmd.Flags().GetString("state"); v != "" {
			opts.State = dnsimpleString(v)
		}
		if v, _ := cmd.Flags().GetString("domain"); v != "" {
			opts.DomainId = dnsimpleString(v)
		}
		if v, _ := cmd.Flags().GetString("contact-id"); v != "" {
			opts.ContactId = dnsimpleString(v)
		}
		if v, _ := cmd.Flags().GetInt("page"); v > 0 {
			opts.Page = dnsimpleInt(v)
		}
		if v, _ := cmd.Flags().GetInt("per-page"); v > 0 {
			opts.PerPage = dnsimpleInt(v)
		}
		resp, err := regListRegistrantChanges(ctx, app.Client, app.AccountID, opts)
		if err != nil {
			return err
		}
		if printJSON(resp.Data) {
			return nil
		}
		printRegistrantChanges(os.Stdout, resp.Data)
		if resp.Pagination != nil && resp.Pagination.TotalPages > 1 {
			fmt.Println(ui.Info(fmt.Sprintf("Page %d of %d (%d total)", resp.Pagination.CurrentPage, resp.Pagination.TotalPages, resp.Pagination.TotalEntries)))
		}
		return nil
	},
}

var registrarRCGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Show a registrant change",
	Long: `Show the details and state of a registrant change.

Examples:
  simple registrar registrant-changes get 101`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseRegIntID("registrant change", args[0])
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		rc, err := regGetRegistrantChange(ctx, app.Client, app.AccountID, id)
		if err != nil {
			return err
		}
		if printJSON(rc) {
			return nil
		}
		printRegistrantChange(os.Stdout, rc)
		return nil
	},
}

var registrarRCCheckCmd = &cobra.Command{
	Use:   "check [domain]",
	Short: "Check what a registrant change needs",
	Long: `Check whether a registrant change is possible and which extended attributes it needs.

Examples:
  simple registrar registrant-changes check example.com --contact-id 42`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		contactID, _ := cmd.Flags().GetString("contact-id")
		if contactID == "" {
			return fmt.Errorf("--contact-id is required")
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		chk, err := regCheckRegistrantChange(ctx, app.Client, app.AccountID, args[0], contactID)
		if err != nil {
			return err
		}
		if printJSON(chk) {
			return nil
		}
		printRegistrantChangeCheck(os.Stdout, args[0], chk)
		return nil
	},
}

var registrarRCCreateCmd = &cobra.Command{
	Use:   "create [domain]",
	Short: "Start a registrant change (requires --yes)",
	Long: `Start changing a domain's registrant to another contact. Run 'check' first to
see which extended attributes the TLD needs.

Without --yes, shows what would happen and exits non-zero.

Examples:
  simple registrar registrant-changes create example.com --contact-id 42
  simple registrar registrant-changes create example.com --contact-id 42 --yes
  simple registrar registrant-changes create example.fr --contact-id 42 --extended-attribute x-fr-registrant-birth-date=1990-01-01 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		contactID, _ := cmd.Flags().GetString("contact-id")
		if contactID == "" {
			return fmt.Errorf("--contact-id is required")
		}
		pairs, _ := cmd.Flags().GetStringArray("extended-attribute")
		attrs, err := parseExtendedAttributes(pairs)
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		if !regConfirmed(cmd) {
			p := regPlan{Action: "change registrant of " + domain, Domain: domain,
				Details:  []regField{{"Domain", domain}, {"New contact", contactID}},
				Warnings: []string{"Some TLDs apply a 60-day transfer lock after a registrant change"}}
			keys := make([]string, 0, len(attrs))
			for k := range attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				p.Details = append(p.Details, regField{"Attr " + k, attrs[k]})
			}
			return regGate(cmd, p)
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		rc, err := regCreateRegistrantChange(ctx, app.Client, app.AccountID, domain, contactID, attrs)
		if err != nil {
			return err
		}
		if printJSON(rc) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Registrant change for '%s' started (ID: %d, state: %s)", domain, rc.Id, rc.State)))
		return nil
	},
}

var registrarRCDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"cancel"},
	Short:   "Cancel a pending registrant change",
	Long: `Cancel a registrant change that has not completed yet.

Examples:
  simple registrar registrant-changes delete 101`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseRegIntID("registrant change", args[0])
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := regDeleteRegistrantChange(ctx, app.Client, app.AccountID, id); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"id": id, "cancelled": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Registrant change %d cancelled", id)))
		return nil
	},
}

func init() {
	registrarCmd.AddCommand(registrarRegistrantChangesCmd)

	registrarRCListCmd.Flags().String("state", "", "Filter by state (new, pending, cancelling, cancelled, completed)")
	registrarRCListCmd.Flags().String("domain", "", "Filter by domain name or ID")
	registrarRCListCmd.Flags().String("contact-id", "", "Filter by contact ID")
	registrarRCListCmd.Flags().Int("page", 0, "Page number")
	registrarRCListCmd.Flags().Int("per-page", 0, "Results per page")
	registrarRegistrantChangesCmd.AddCommand(registrarRCListCmd)

	registrarRegistrantChangesCmd.AddCommand(registrarRCGetCmd)

	registrarRCCheckCmd.Flags().String("contact-id", "", "Contact ID of the new registrant (required)")
	registrarRegistrantChangesCmd.AddCommand(registrarRCCheckCmd)

	registrarRCCreateCmd.Flags().String("contact-id", "", "Contact ID of the new registrant (required)")
	registrarRCCreateCmd.Flags().StringArray("extended-attribute", nil, "TLD extended attribute as name=value (repeatable)")
	regAddYesFlag(registrarRCCreateCmd)
	registrarRegistrantChangesCmd.AddCommand(registrarRCCreateCmd)

	registrarRegistrantChangesCmd.AddCommand(registrarRCDeleteCmd)
}
