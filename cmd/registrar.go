package cmd

import (
	"context"
	"errors"
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

var registrarCmd = &cobra.Command{
	Use:     "registrar",
	Aliases: []string{"reg"},
	Short:   "Register, transfer, renew, and manage registered domains",
	Long: `Work with the DNSimple registrar: check availability and prices, register,
transfer, renew, and restore domains, and manage name servers, transfer lock,
WHOIS privacy, auto-renewal, and registrant changes.

Commands that spend money or are hard to undo (register, transfer, renew,
restore, transfer-out, name server changes, registrant-change create) only
show what they would do unless you pass --yes.

Examples:
  simple registrar check example.com
  simple registrar prices example.com
  simple registrar register example.com --registrant-id 42 --yes
  simple reg nameservers example.com`,
}

// --- shared helpers -------------------------------------------------------

// regField is one labelled line of a dry-run plan.
type regField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// regPlan describes a confirm-gated action that was not confirmed.
type regPlan struct {
	DryRun   bool       `json:"dry_run"`
	Action   string     `json:"action"`
	Domain   string     `json:"domain,omitempty"`
	Details  []regField `json:"details,omitempty"`
	Warnings []string   `json:"warnings,omitempty"`
}

// errRegNotConfirmed is returned when a gated command runs without --yes.
var errRegNotConfirmed = errors.New("not confirmed; re-run with --yes to proceed")

func regAddYesFlag(c *cobra.Command) {
	c.Flags().BoolP("yes", "y", false, "Confirm the action (without it, only show what would happen)")
}

func regConfirmed(cmd *cobra.Command) bool {
	yes, _ := cmd.Flags().GetBool("yes")
	return yes
}

// printRegPlan prints a plan for an unconfirmed action.
func printRegPlan(w io.Writer, p regPlan) {
	fmt.Fprintln(w, ui.TitleStyle.Render("🧪 Dry run: "+p.Action))
	for _, f := range p.Details {
		fmt.Fprintf(w, "  %-14s %s\n", f.Label+":", f.Value)
	}
	for _, warn := range p.Warnings {
		fmt.Fprintln(w, ui.Warn(warn))
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, ui.Info("Nothing was changed. Re-run with --yes to do it."))
}

// regGate returns nil when --yes was passed; otherwise it prints the plan
// (as JSON with --json) and returns errRegNotConfirmed.
func regGate(cmd *cobra.Command, p regPlan) error {
	if regConfirmed(cmd) {
		return nil
	}
	p.DryRun = true
	if !printJSON(p) {
		printRegPlan(os.Stdout, p)
	}
	return errRegNotConfirmed
}

// regError turns registrar API errors into readable messages.
func regError(action string, err error) error {
	var apiErr *dnsimple.ErrorResponse
	if errors.As(err, &apiErr) {
		msg := apiErr.Message
		if len(apiErr.AttributeErrors) > 0 {
			keys := make([]string, 0, len(apiErr.AttributeErrors))
			for k := range apiErr.AttributeErrors {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var parts []string
			for _, k := range keys {
				parts = append(parts, k+": "+strings.Join(apiErr.AttributeErrors[k], ", "))
			}
			msg += " (" + strings.Join(parts, "; ") + ")"
		}
		return fmt.Errorf("failed to %s: %s", action, msg)
	}
	return fmt.Errorf("failed to %s: %w", action, err)
}

// parseExtendedAttributes turns ["k=v", ...] into a map.
func parseExtendedAttributes(pairs []string) (map[string]string, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	m := make(map[string]string, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		k = strings.TrimSpace(k)
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid extended attribute %q (use name=value)", p)
		}
		m[k] = v
	}
	return m, nil
}

func regMoney(v float64) string {
	return fmt.Sprintf("$%.2f", v)
}

func regYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func regPrintTimes(w io.Writer, created, updated string) {
	if created != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Created:", created)
	}
	if updated != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Updated:", updated)
	}
}

func parseRegID(kind, s string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s ID %q (must be a positive number)", kind, s)
	}
	return id, nil
}

// --- check / prices / research ---------------------------------------------

func regCheckDomain(ctx context.Context, c *dnsimple.Client, accountID, domain string) (*dnsimple.DomainCheck, error) {
	resp, err := c.Registrar.CheckDomain(ctx, accountID, domain)
	if err != nil {
		return nil, regError(fmt.Sprintf("check '%s'", domain), err)
	}
	return resp.Data, nil
}

func printDomainCheck(w io.Writer, d *dnsimple.DomainCheck) {
	fmt.Fprintln(w, ui.TitleStyle.Render("🔎 "+d.Domain))
	avail := ui.WarningStyle.Render("no")
	if d.Available {
		avail = ui.SuccessStyle.Render("yes")
	}
	fmt.Fprintf(w, "  %-14s %s\n", "Available:", avail)
	fmt.Fprintf(w, "  %-14s %s\n", "Premium:", regYesNo(d.Premium))
}

func regGetPrices(ctx context.Context, c *dnsimple.Client, accountID, domain string) (*dnsimple.DomainPrice, error) {
	resp, err := c.Registrar.GetDomainPrices(ctx, accountID, domain)
	if err != nil {
		return nil, regError(fmt.Sprintf("get prices for '%s'", domain), err)
	}
	return resp.Data, nil
}

func printDomainPrices(w io.Writer, p *dnsimple.DomainPrice) {
	fmt.Fprintln(w, ui.TitleStyle.Render("💰 "+p.Domain))
	fmt.Fprintf(w, "  %-14s %s\n", "Premium:", regYesNo(p.Premium))
	fmt.Fprintf(w, "  %-14s %s\n", "Registration:", regMoney(p.RegistrationPrice))
	fmt.Fprintf(w, "  %-14s %s\n", "Renewal:", regMoney(p.RenewalPrice))
	fmt.Fprintf(w, "  %-14s %s\n", "Transfer:", regMoney(p.TransferPrice))
	fmt.Fprintf(w, "  %-14s %s\n", "Restore:", regMoney(p.RestorePrice))
	if p.TrusteePrice != nil {
		fmt.Fprintf(w, "  %-14s %s\n", "Trustee:", regMoney(*p.TrusteePrice))
	}
}

func regResearch(ctx context.Context, c *dnsimple.Client, accountID, domain string) (*dnsimple.DomainResearchStatus, error) {
	resp, err := c.Domains.GetDomainResearchStatus(ctx, accountID, domain)
	if err != nil {
		return nil, regError(fmt.Sprintf("research '%s'", domain), err)
	}
	return resp.Data, nil
}

func printResearch(w io.Writer, r *dnsimple.DomainResearchStatus) {
	fmt.Fprintln(w, ui.TitleStyle.Render("🔬 "+r.Domain))
	style := ui.WarningStyle
	if r.Availability == "available" {
		style = ui.SuccessStyle
	}
	fmt.Fprintf(w, "  %-14s %s\n", "Availability:", style.Render(r.Availability))
	if r.RequestID != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Request ID:", ui.SubtleStyle.Render(r.RequestID))
	}
	for _, e := range r.Errors {
		fmt.Fprintln(w, ui.Warn(e))
	}
}

var registrarCheckCmd = &cobra.Command{
	Use:   "check [domain]",
	Short: "Check whether a domain is available to register",
	Long: `Check whether a domain is available for registration, and whether it is premium.

Examples:
  simple registrar check example.com
  simple registrar check example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		d, err := regCheckDomain(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(d) {
			return nil
		}
		printDomainCheck(os.Stdout, d)
		return nil
	},
}

var registrarPricesCmd = &cobra.Command{
	Use:     "prices [domain]",
	Aliases: []string{"price"},
	Short:   "Show registration, renewal, transfer, and restore prices",
	Long: `Show what DNSimple charges to register, renew, transfer, or restore a domain.

Examples:
  simple registrar prices example.com
  simple registrar prices example.io --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		p, err := regGetPrices(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(p) {
			return nil
		}
		printDomainPrices(os.Stdout, p)
		return nil
	},
}

var registrarResearchCmd = &cobra.Command{
	Use:   "research [domain]",
	Short: "Research a domain's availability (domain research API)",
	Long: `Look up a domain's availability via the DNSimple domain research API.

Examples:
  simple registrar research example.com
  simple registrar research example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := regResearch(ctx, app.Client, app.AccountID, args[0])
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		printResearch(os.Stdout, r)
		return nil
	},
}

// --- register / transfer common flags --------------------------------------

// regAcquireFlags are the flags shared by register and transfer.
type regAcquireFlags struct {
	RegistrantID       int
	WhoisPrivacy       bool
	AutoRenew          bool
	Trustee            *bool
	PremiumPrice       string
	ExtendedAttributes map[string]string
}

func addRegAcquireFlags(c *cobra.Command) {
	c.Flags().Int("registrant-id", 0, "Contact ID of the registrant (required)")
	c.Flags().Bool("whois-privacy", false, "Enable WHOIS privacy")
	c.Flags().Bool("auto-renew", false, "Enable auto-renewal")
	c.Flags().Bool("trustee", false, "Use the trustee service (for TLDs that require local presence)")
	c.Flags().String("premium-price", "", "Premium price to accept (required for premium domains)")
	c.Flags().StringArray("extended-attribute", nil, "TLD extended attribute as name=value (repeatable)")
	regAddYesFlag(c)
}

func readRegAcquireFlags(cmd *cobra.Command) (*regAcquireFlags, error) {
	f := &regAcquireFlags{}
	f.RegistrantID, _ = cmd.Flags().GetInt("registrant-id")
	if f.RegistrantID <= 0 {
		return nil, fmt.Errorf("--registrant-id is required (a contact ID; see your DNSimple contacts)")
	}
	f.WhoisPrivacy, _ = cmd.Flags().GetBool("whois-privacy")
	f.AutoRenew, _ = cmd.Flags().GetBool("auto-renew")
	if cmd.Flags().Changed("trustee") {
		v, _ := cmd.Flags().GetBool("trustee")
		f.Trustee = &v
	}
	f.PremiumPrice, _ = cmd.Flags().GetString("premium-price")
	pairs, _ := cmd.Flags().GetStringArray("extended-attribute")
	attrs, err := parseExtendedAttributes(pairs)
	if err != nil {
		return nil, err
	}
	f.ExtendedAttributes = attrs
	return f, nil
}

func (f *regAcquireFlags) planDetails() []regField {
	out := []regField{
		{"Registrant", strconv.Itoa(f.RegistrantID)},
		{"WHOIS privacy", onOff(f.WhoisPrivacy)},
		{"Auto-renew", onOff(f.AutoRenew)},
	}
	if f.Trustee != nil {
		out = append(out, regField{"Trustee", onOff(*f.Trustee)})
	}
	if f.PremiumPrice != "" {
		out = append(out, regField{"Premium price", f.PremiumPrice})
	}
	keys := make([]string, 0, len(f.ExtendedAttributes))
	for k := range f.ExtendedAttributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, regField{"Attr " + k, f.ExtendedAttributes[k]})
	}
	return out
}

// regPricePreview adds a price line (read-only lookup) to a plan, ignoring lookup errors.
func regPricePreview(ctx context.Context, c *dnsimple.Client, accountID, domain string, pick func(*dnsimple.DomainPrice) float64, p *regPlan, premiumPrice string) {
	price, err := regGetPrices(ctx, c, accountID, domain)
	if err != nil {
		p.Warnings = append(p.Warnings, "Could not look up price: "+err.Error())
		return
	}
	p.Details = append(p.Details, regField{"Price", regMoney(pick(price))})
	if price.Premium && premiumPrice == "" {
		p.Warnings = append(p.Warnings, "This is a premium domain; pass --premium-price "+fmt.Sprintf("%.2f", pick(price))+" to accept the price")
	}
}

// --- register --------------------------------------------------------------

func regRegister(ctx context.Context, c *dnsimple.Client, accountID, domain string, f *regAcquireFlags) (*dnsimple.DomainRegistration, error) {
	input := &dnsimple.RegisterDomainInput{
		RegistrantID:       f.RegistrantID,
		EnableWhoisPrivacy: f.WhoisPrivacy,
		EnableAutoRenewal:  f.AutoRenew,
		Trustee:            f.Trustee,
		ExtendedAttributes: f.ExtendedAttributes,
		PremiumPrice:       f.PremiumPrice,
	}
	resp, err := c.Registrar.RegisterDomain(ctx, accountID, domain, input)
	if err != nil {
		return nil, regError(fmt.Sprintf("register '%s'", domain), err)
	}
	return resp.Data, nil
}

func regGetRegistration(ctx context.Context, c *dnsimple.Client, accountID, domain, id string) (*dnsimple.DomainRegistration, error) {
	resp, err := c.Registrar.GetDomainRegistration(ctx, accountID, domain, id)
	if err != nil {
		return nil, regError(fmt.Sprintf("get registration %s for '%s'", id, domain), err)
	}
	return resp.Data, nil
}

func printRegistration(w io.Writer, domain string, r *dnsimple.DomainRegistration) {
	fmt.Fprintln(w, ui.TitleStyle.Render("📝 Registration "+domain))
	fmt.Fprintf(w, "  %-14s %d\n", "ID:", r.ID)
	fmt.Fprintf(w, "  %-14s %d\n", "Domain ID:", r.DomainID)
	fmt.Fprintf(w, "  %-14s %s\n", "State:", r.State)
	fmt.Fprintf(w, "  %-14s %d\n", "Registrant:", r.RegistrantID)
	fmt.Fprintf(w, "  %-14s %d year(s)\n", "Period:", r.Period)
	fmt.Fprintf(w, "  %-14s %s\n", "Auto-Renew:", onOff(r.AutoRenew))
	fmt.Fprintf(w, "  %-14s %s\n", "WHOIS privacy:", onOff(r.WhoisPrivacy))
	fmt.Fprintf(w, "  %-14s %s\n", "Trustee:", onOff(r.Trustee))
	regPrintTimes(w, r.CreatedAt, r.UpdatedAt)
}

var registrarRegisterCmd = &cobra.Command{
	Use:   "register [domain]",
	Short: "Register a domain (charges your account; requires --yes)",
	Long: `Register a domain through DNSimple. This charges your account.

Without --yes, shows the price and what would be registered, and exits non-zero.

Examples:
  simple registrar register example.com --registrant-id 42
  simple registrar register example.com --registrant-id 42 --whois-privacy --auto-renew --yes
  simple registrar register example.io --registrant-id 42 --premium-price 199.00 --yes
  simple registrar register example.us --registrant-id 42 --extended-attribute us_nexus=C11 --extended-attribute us_purpose=P1 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		f, err := readRegAcquireFlags(cmd)
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if !regConfirmed(cmd) {
			p := regPlan{Action: "register " + domain, Domain: domain,
				Details: append([]regField{{"Domain", domain}}, f.planDetails()...)}
			regPricePreview(ctx, app.Client, app.AccountID, domain, func(p *dnsimple.DomainPrice) float64 { return p.RegistrationPrice }, &p, f.PremiumPrice)
			return regGate(cmd, p)
		}
		r, err := regRegister(ctx, app.Client, app.AccountID, domain, f)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Registration of '%s' submitted (ID: %d, state: %s)", domain, r.ID, r.State)))
		return nil
	},
}

var registrarRegistrationCmd = &cobra.Command{
	Use:   "registration [domain] [registration-id]",
	Short: "Show a domain registration",
	Long: `Show the details and state of a domain registration.

Examples:
  simple registrar registration example.com 361
  simple registrar registration example.com 361 --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := regGetRegistration(ctx, app.Client, app.AccountID, args[0], args[1])
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		printRegistration(os.Stdout, args[0], r)
		return nil
	},
}

// --- transfers -------------------------------------------------------------

func regTransfer(ctx context.Context, c *dnsimple.Client, accountID, domain, authCode string, f *regAcquireFlags) (*dnsimple.DomainTransfer, error) {
	input := &dnsimple.TransferDomainInput{
		RegistrantID:       f.RegistrantID,
		AuthCode:           authCode,
		EnableWhoisPrivacy: f.WhoisPrivacy,
		EnableAutoRenewal:  f.AutoRenew,
		Trustee:            f.Trustee,
		ExtendedAttributes: f.ExtendedAttributes,
		PremiumPrice:       f.PremiumPrice,
	}
	resp, err := c.Registrar.TransferDomain(ctx, accountID, domain, input)
	if err != nil {
		return nil, regError(fmt.Sprintf("transfer '%s'", domain), err)
	}
	return resp.Data, nil
}

func regGetTransfer(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) (*dnsimple.DomainTransfer, error) {
	resp, err := c.Registrar.GetDomainTransfer(ctx, accountID, domain, id)
	if err != nil {
		return nil, regError(fmt.Sprintf("get transfer %d for '%s'", id, domain), err)
	}
	return resp.Data, nil
}

func regCancelTransfer(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) (*dnsimple.DomainTransfer, error) {
	resp, err := c.Registrar.CancelDomainTransfer(ctx, accountID, domain, id)
	if err != nil {
		return nil, regError(fmt.Sprintf("cancel transfer %d for '%s'", id, domain), err)
	}
	return resp.Data, nil
}

func regTransferOut(ctx context.Context, c *dnsimple.Client, accountID, domain string) error {
	if _, err := c.Registrar.TransferDomainOut(ctx, accountID, domain); err != nil {
		return regError(fmt.Sprintf("authorize transfer out of '%s'", domain), err)
	}
	return nil
}

func printTransfer(w io.Writer, domain string, t *dnsimple.DomainTransfer) {
	fmt.Fprintln(w, ui.TitleStyle.Render("🚚 Transfer "+domain))
	fmt.Fprintf(w, "  %-14s %d\n", "ID:", t.ID)
	fmt.Fprintf(w, "  %-14s %d\n", "Domain ID:", t.DomainID)
	fmt.Fprintf(w, "  %-14s %s\n", "State:", t.State)
	if t.StatusDescription != "" {
		fmt.Fprintf(w, "  %-14s %s\n", "Status:", t.StatusDescription)
	}
	fmt.Fprintf(w, "  %-14s %d\n", "Registrant:", t.RegistrantID)
	fmt.Fprintf(w, "  %-14s %s\n", "Auto-Renew:", onOff(t.AutoRenew))
	fmt.Fprintf(w, "  %-14s %s\n", "WHOIS privacy:", onOff(t.WhoisPrivacy))
	fmt.Fprintf(w, "  %-14s %s\n", "Trustee:", onOff(t.Trustee))
	regPrintTimes(w, t.CreatedAt, t.UpdatedAt)
}

var registrarTransferCmd = &cobra.Command{
	Use:   "transfer [domain]",
	Short: "Transfer a domain in to DNSimple (charges your account; requires --yes)",
	Long: `Start a transfer of a domain from another registrar to DNSimple. This charges your account.

Without --yes, shows the price and what would be transferred, and exits non-zero.

Examples:
  simple registrar transfer example.com --registrant-id 42 --auth-code 'xjfjfjvhc293'
  simple registrar transfer example.com --registrant-id 42 --auth-code 'xjfjfjvhc293' --whois-privacy --auto-renew --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		f, err := readRegAcquireFlags(cmd)
		if err != nil {
			return err
		}
		authCode, _ := cmd.Flags().GetString("auth-code")
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if !regConfirmed(cmd) {
			auth := "(none)"
			if authCode != "" {
				auth = "(provided)"
			}
			p := regPlan{Action: "transfer " + domain + " to DNSimple", Domain: domain,
				Details: append([]regField{{"Domain", domain}, {"Auth code", auth}}, f.planDetails()...)}
			regPricePreview(ctx, app.Client, app.AccountID, domain, func(p *dnsimple.DomainPrice) float64 { return p.TransferPrice }, &p, f.PremiumPrice)
			return regGate(cmd, p)
		}
		t, err := regTransfer(ctx, app.Client, app.AccountID, domain, authCode, f)
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Transfer of '%s' started (ID: %d, state: %s)", domain, t.ID, t.State)))
		return nil
	},
}

var registrarTransferStatusCmd = &cobra.Command{
	Use:   "transfer-status [domain] [transfer-id]",
	Short: "Show the status of a domain transfer",
	Long: `Show the details and state of an inbound domain transfer.

Examples:
  simple registrar transfer-status example.com 361
  simple registrar transfer-status example.com 361 --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseRegID("transfer", args[1])
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t, err := regGetTransfer(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		printTransfer(os.Stdout, args[0], t)
		return nil
	},
}

var registrarTransferCancelCmd = &cobra.Command{
	Use:   "transfer-cancel [domain] [transfer-id]",
	Short: "Cancel a pending domain transfer",
	Long: `Cancel an inbound domain transfer that has not completed yet.

Examples:
  simple registrar transfer-cancel example.com 361`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseRegID("transfer", args[1])
		if err != nil {
			return err
		}
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		t, err := regCancelTransfer(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(t) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Transfer %d of '%s' cancelled (state: %s)", id, args[0], t.State)))
		return nil
	},
}

var registrarTransferOutCmd = &cobra.Command{
	Use:   "transfer-out [domain]",
	Short: "Authorize transferring a domain away from DNSimple (requires --yes)",
	Long: `Authorize a transfer out: DNSimple emails the auth code to the registrant so
another registrar can take the domain.

Without --yes, shows what would happen and exits non-zero.

Examples:
  simple registrar transfer-out example.com
  simple registrar transfer-out example.com --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cmd.SilenceUsage = true
		if !regConfirmed(cmd) {
			return regGate(cmd, regPlan{Action: "authorize transfer out of " + domain, Domain: domain,
				Details:  []regField{{"Domain", domain}},
				Warnings: []string{"The auth code will be sent to the registrant; the domain can then leave DNSimple"}})
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := regTransferOut(ctx, app.Client, app.AccountID, domain); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"domain": domain, "transfer_out_authorized": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Transfer out of '%s' authorized; the auth code is on its way to the registrant", domain)))
		return nil
	},
}

// --- renew / restore -------------------------------------------------------

func regRenew(ctx context.Context, c *dnsimple.Client, accountID, domain string, period int, premiumPrice string) (*dnsimple.DomainRenewal, error) {
	resp, err := c.Registrar.RenewDomain(ctx, accountID, domain, &dnsimple.RenewDomainInput{Period: period, PremiumPrice: premiumPrice})
	if err != nil {
		return nil, regError(fmt.Sprintf("renew '%s'", domain), err)
	}
	return resp.Data, nil
}

func regGetRenewal(ctx context.Context, c *dnsimple.Client, accountID, domain, id string) (*dnsimple.DomainRenewal, error) {
	resp, err := c.Registrar.GetDomainRenewal(ctx, accountID, domain, id)
	if err != nil {
		return nil, regError(fmt.Sprintf("get renewal %s for '%s'", id, domain), err)
	}
	return resp.Data, nil
}

// regRestore restores an expired domain. The SDK reuses the renewal input and
// response types for restores; the response carries id, domain_id and state.
func regRestore(ctx context.Context, c *dnsimple.Client, accountID, domain, premiumPrice string) (*dnsimple.DomainRenewal, error) {
	resp, err := c.Registrar.RestoreDomain(ctx, accountID, domain, &dnsimple.RenewDomainInput{PremiumPrice: premiumPrice})
	if err != nil {
		return nil, regError(fmt.Sprintf("restore '%s'", domain), err)
	}
	return resp.Data, nil
}

func regGetRestore(ctx context.Context, c *dnsimple.Client, accountID, domain, id string) (*dnsimple.DomainRestore, error) {
	resp, err := c.Registrar.GetDomainRestore(ctx, accountID, domain, id)
	if err != nil {
		return nil, regError(fmt.Sprintf("get restore %s for '%s'", id, domain), err)
	}
	return resp.Data, nil
}

func printRenewal(w io.Writer, domain string, r *dnsimple.DomainRenewal) {
	fmt.Fprintln(w, ui.TitleStyle.Render("🔄 Renewal "+domain))
	fmt.Fprintf(w, "  %-14s %d\n", "ID:", r.ID)
	fmt.Fprintf(w, "  %-14s %d\n", "Domain ID:", r.DomainID)
	fmt.Fprintf(w, "  %-14s %s\n", "State:", r.State)
	fmt.Fprintf(w, "  %-14s %d year(s)\n", "Period:", r.Period)
	regPrintTimes(w, r.CreatedAt, r.UpdatedAt)
}

func printRestore(w io.Writer, domain string, r *dnsimple.DomainRestore) {
	fmt.Fprintln(w, ui.TitleStyle.Render("♻️ Restore "+domain))
	fmt.Fprintf(w, "  %-14s %d\n", "ID:", r.ID)
	fmt.Fprintf(w, "  %-14s %d\n", "Domain ID:", r.DomainID)
	fmt.Fprintf(w, "  %-14s %s\n", "State:", r.State)
	regPrintTimes(w, r.CreatedAt, r.UpdatedAt)
}

var registrarRenewCmd = &cobra.Command{
	Use:   "renew [domain]",
	Short: "Renew a domain (charges your account; requires --yes)",
	Long: `Renew a domain registered with DNSimple. This charges your account.

Without --yes, shows the price and period, and exits non-zero.

Examples:
  simple registrar renew example.com
  simple registrar renew example.com --period 2 --yes
  simple registrar renew example.io --premium-price 199.00 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		period, _ := cmd.Flags().GetInt("period")
		if cmd.Flags().Changed("period") && period <= 0 {
			return fmt.Errorf("--period must be at least 1 year")
		}
		premium, _ := cmd.Flags().GetString("premium-price")
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if !regConfirmed(cmd) {
			p := regPlan{Action: "renew " + domain, Domain: domain,
				Details: []regField{{"Domain", domain}, {"Period", fmt.Sprintf("%d year(s)", period)}}}
			if premium != "" {
				p.Details = append(p.Details, regField{"Premium price", premium})
			}
			regPricePreview(ctx, app.Client, app.AccountID, domain, func(pr *dnsimple.DomainPrice) float64 { return pr.RenewalPrice * float64(period) }, &p, premium)
			return regGate(cmd, p)
		}
		r, err := regRenew(ctx, app.Client, app.AccountID, domain, period, premium)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Renewal of '%s' for %d year(s) submitted (ID: %d, state: %s)", domain, r.Period, r.ID, r.State)))
		return nil
	},
}

var registrarRenewalCmd = &cobra.Command{
	Use:   "renewal [domain] [renewal-id]",
	Short: "Show a domain renewal",
	Long: `Show the details and state of a domain renewal.

Examples:
  simple registrar renewal example.com 1
  simple registrar renewal example.com 1 --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := regGetRenewal(ctx, app.Client, app.AccountID, args[0], args[1])
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		printRenewal(os.Stdout, args[0], r)
		return nil
	},
}

var registrarRestoreCmd = &cobra.Command{
	Use:   "restore [domain]",
	Short: "Restore an expired domain (charges your account; requires --yes)",
	Long: `Restore a domain that expired and is in its redemption period. This charges your account.

Without --yes, shows the price and exits non-zero.

Examples:
  simple registrar restore example.com
  simple registrar restore example.com --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		premium, _ := cmd.Flags().GetString("premium-price")
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if !regConfirmed(cmd) {
			p := regPlan{Action: "restore " + domain, Domain: domain, Details: []regField{{"Domain", domain}}}
			if premium != "" {
				p.Details = append(p.Details, regField{"Premium price", premium})
			}
			regPricePreview(ctx, app.Client, app.AccountID, domain, func(pr *dnsimple.DomainPrice) float64 { return pr.RestorePrice }, &p, premium)
			return regGate(cmd, p)
		}
		r, err := regRestore(ctx, app.Client, app.AccountID, domain, premium)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Restore of '%s' submitted (ID: %d, state: %s)", domain, r.ID, r.State)))
		return nil
	},
}

var registrarRestoreStatusCmd = &cobra.Command{
	Use:   "restore-status [domain] [restore-id]",
	Short: "Show a domain restore",
	Long: `Show the details and state of a domain restore.

Examples:
  simple registrar restore-status example.com 43
  simple registrar restore-status example.com 43 --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := regGetRestore(ctx, app.Client, app.AccountID, args[0], args[1])
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		printRestore(os.Stdout, args[0], r)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(registrarCmd)

	registrarCmd.AddCommand(registrarCheckCmd)
	registrarCmd.AddCommand(registrarPricesCmd)
	registrarCmd.AddCommand(registrarResearchCmd)

	addRegAcquireFlags(registrarRegisterCmd)
	registrarCmd.AddCommand(registrarRegisterCmd)
	registrarCmd.AddCommand(registrarRegistrationCmd)

	addRegAcquireFlags(registrarTransferCmd)
	registrarTransferCmd.Flags().String("auth-code", "", "Authorization (EPP) code from the current registrar")
	registrarCmd.AddCommand(registrarTransferCmd)
	registrarCmd.AddCommand(registrarTransferStatusCmd)
	registrarCmd.AddCommand(registrarTransferCancelCmd)
	regAddYesFlag(registrarTransferOutCmd)
	registrarCmd.AddCommand(registrarTransferOutCmd)

	registrarRenewCmd.Flags().Int("period", 1, "Renewal period in years")
	registrarRenewCmd.Flags().String("premium-price", "", "Premium price to accept (required for premium domains)")
	regAddYesFlag(registrarRenewCmd)
	registrarCmd.AddCommand(registrarRenewCmd)
	registrarCmd.AddCommand(registrarRenewalCmd)

	registrarRestoreCmd.Flags().String("premium-price", "", "Premium price to accept (required for premium domains)")
	regAddYesFlag(registrarRestoreCmd)
	registrarCmd.AddCommand(registrarRestoreCmd)
	registrarCmd.AddCommand(registrarRestoreStatusCmd)
}
