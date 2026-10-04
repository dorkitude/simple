package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var contactsCmd = &cobra.Command{
	Use:     "contacts",
	Aliases: []string{"contact"},
	Short:   "Manage registrant contacts",
	Long: `List, view, create, update, and delete the contacts used for domain registration.

Examples:
  simple contacts list
  simple contacts get 42
  simple contacts create --first-name Ada --last-name Lovelace --email ada@example.com ...
  simple contacts update 42 --phone +1.5555550100
  simple contacts delete 42 --yes`,
}

// contactFlagFields maps flag names to the Contact field they set.
var contactFlagFields = []struct {
	flag  string
	usage string
	set   func(*dnsimple.Contact, string)
}{
	{"label", "Label for this contact", func(c *dnsimple.Contact, v string) { c.Label = v }},
	{"first-name", "First name", func(c *dnsimple.Contact, v string) { c.FirstName = v }},
	{"last-name", "Last name", func(c *dnsimple.Contact, v string) { c.LastName = v }},
	{"job-title", "Job title", func(c *dnsimple.Contact, v string) { c.JobTitle = v }},
	{"organization", "Organization name", func(c *dnsimple.Contact, v string) { c.Organization = v }},
	{"address1", "Address line 1", func(c *dnsimple.Contact, v string) { c.Address1 = v }},
	{"address2", "Address line 2", func(c *dnsimple.Contact, v string) { c.Address2 = v }},
	{"city", "City", func(c *dnsimple.Contact, v string) { c.City = v }},
	{"state-province", "State or province", func(c *dnsimple.Contact, v string) { c.StateProvince = v }},
	{"postal-code", "Postal code", func(c *dnsimple.Contact, v string) { c.PostalCode = v }},
	{"country", "Country (ISO 3166 alpha-2, e.g. US)", func(c *dnsimple.Contact, v string) { c.Country = v }},
	{"phone", "Phone (e.g. +1.4015550100)", func(c *dnsimple.Contact, v string) { c.Phone = v }},
	{"fax", "Fax", func(c *dnsimple.Contact, v string) { c.Fax = v }},
	{"email", "Email address", func(c *dnsimple.Contact, v string) { c.Email = v }},
}

func addContactFlags(c *cobra.Command) {
	for _, f := range contactFlagFields {
		c.Flags().String(f.flag, "", f.usage)
	}
}

// contactFromFlags builds a Contact from the flags that were explicitly set.
// It returns the number of fields set.
func contactFromFlags(fs *pflag.FlagSet) (dnsimple.Contact, int) {
	var c dnsimple.Contact
	n := 0
	for _, f := range contactFlagFields {
		if fs.Changed(f.flag) {
			v, _ := fs.GetString(f.flag)
			f.set(&c, v)
			n++
		}
	}
	return c, n
}

// requiredContactFlags are the fields DNSimple requires to create a contact.
var requiredContactFlags = []string{"first-name", "last-name", "address1", "city", "state-province", "postal-code", "country", "email", "phone"}

func missingContactFlags(fs *pflag.FlagSet) []string {
	var missing []string
	for _, name := range requiredContactFlags {
		if v, _ := fs.GetString(name); strings.TrimSpace(v) == "" {
			missing = append(missing, "--"+name)
		}
	}
	return missing
}

// ---------- logic (testable) ----------

func listContacts(ctx context.Context, c *dnsimple.Client, accountID string, opts *dnsimple.ListOptions) (*dnsimple.ContactsResponse, error) {
	resp, err := c.Contacts.ListContacts(ctx, accountID, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list contacts: %w", err)
	}
	return resp, nil
}

func getContact(ctx context.Context, c *dnsimple.Client, accountID string, id int64) (*dnsimple.Contact, error) {
	resp, err := c.Contacts.GetContact(ctx, accountID, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact: %w", err)
	}
	return resp.Data, nil
}

func createContact(ctx context.Context, c *dnsimple.Client, accountID string, contact dnsimple.Contact) (*dnsimple.Contact, error) {
	resp, err := c.Contacts.CreateContact(ctx, accountID, contact)
	if err != nil {
		return nil, fmt.Errorf("failed to create contact: %w", err)
	}
	return resp.Data, nil
}

func updateContact(ctx context.Context, c *dnsimple.Client, accountID string, id int64, contact dnsimple.Contact) (*dnsimple.Contact, error) {
	resp, err := c.Contacts.UpdateContact(ctx, accountID, id, contact)
	if err != nil {
		return nil, fmt.Errorf("failed to update contact: %w", err)
	}
	return resp.Data, nil
}

func deleteContact(ctx context.Context, c *dnsimple.Client, accountID string, id int64) error {
	if _, err := c.Contacts.DeleteContact(ctx, accountID, id); err != nil {
		return fmt.Errorf("failed to delete contact: %w", err)
	}
	return nil
}

// ---------- output ----------

func contactDisplayName(c *dnsimple.Contact) string {
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	if name == "" {
		name = c.Email
	}
	return name
}

func printContact(c *dnsimple.Contact) {
	fmt.Println(ui.TitleStyle.Render("👤 " + contactDisplayName(c)))
	fmt.Println()
	row := func(k, v string) {
		if v != "" {
			fmt.Printf("  %-14s %s\n", k+":", v)
		}
	}
	fmt.Printf("  %-14s %d\n", "ID:", c.ID)
	row("Label", c.Label)
	row("First Name", c.FirstName)
	row("Last Name", c.LastName)
	row("Job Title", c.JobTitle)
	row("Organization", c.Organization)
	row("Email", c.Email)
	row("Phone", c.Phone)
	row("Fax", c.Fax)
	row("Address", c.Address1)
	row("Address 2", c.Address2)
	row("City", c.City)
	row("State", c.StateProvince)
	row("Postal Code", c.PostalCode)
	row("Country", c.Country)
	row("Created", c.CreatedAt)
	row("Updated", c.UpdatedAt)
}

// ---------- commands ----------

var contactsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List contacts",
	Long: `List all contacts in your account.

Examples:
  simple contacts list
  simple contacts list --sort label:asc
  simple contacts list --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		resp, err := listContacts(ctx, app.Client, app.AccountID, listOptionsFromFlags(cmd))
		if err != nil {
			return err
		}
		if printJSON(resp.Data) {
			return nil
		}
		if len(resp.Data) == 0 {
			fmt.Println(ui.Warn("No contacts found"))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("👤 %d contacts", len(resp.Data))))
		fmt.Println()
		for i := range resp.Data {
			c := &resp.Data[i]
			fmt.Printf("  %-10d %-30s %s", c.ID, ui.AccentStyle.Render(contactDisplayName(c)), c.Email)
			if c.Label != "" {
				fmt.Printf("  %s", ui.SubtleStyle.Render("["+c.Label+"]"))
			}
			fmt.Println()
		}
		printPagination(resp.Pagination)
		return nil
	},
}

var contactsGetCmd = &cobra.Command{
	Use:   "get [contact-id]",
	Short: "Get contact details",
	Long: `Display a contact.

Examples:
  simple contacts get 42
  simple contacts get 42 --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("contact ID", args[0])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		c, err := getContact(ctx, app.Client, app.AccountID, id)
		if err != nil {
			return err
		}
		if printJSON(c) {
			return nil
		}
		printContact(c)
		return nil
	},
}

var contactsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a contact",
	Long: `Create a registrant contact. Required: --first-name, --last-name,
--address1, --city, --state-province, --postal-code, --country, --email, --phone.

Examples:
  simple contacts create --label work --first-name Ada --last-name Lovelace \
    --email ada@example.com --phone +1.4015550100 --address1 "1 Main St" \
    --city Providence --state-province RI --postal-code 02903 --country US`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if missing := missingContactFlags(cmd.Flags()); len(missing) > 0 {
			return fmt.Errorf("missing required flags: %s", strings.Join(missing, ", "))
		}
		contact, _ := contactFromFlags(cmd.Flags())
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		c, err := createContact(ctx, app.Client, app.AccountID, contact)
		if err != nil {
			return err
		}
		if printJSON(c) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Contact '%s' created! (ID: %d)", contactDisplayName(c), c.ID)))
		return nil
	},
}

var contactsUpdateCmd = &cobra.Command{
	Use:   "update [contact-id]",
	Short: "Update a contact",
	Long: `Update a contact. Only the flags you pass are changed.

Examples:
  simple contacts update 42 --phone +1.4015550199
  simple contacts update 42 --label personal --email me@example.com`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("contact ID", args[0])
		if err != nil {
			return err
		}
		contact, n := contactFromFlags(cmd.Flags())
		if n == 0 {
			return fmt.Errorf("nothing to update; pass at least one field flag (see --help)")
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		c, err := updateContact(ctx, app.Client, app.AccountID, id, contact)
		if err != nil {
			return err
		}
		if printJSON(c) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Contact %d updated.", c.ID)))
		return nil
	},
}

var contactsDeleteCmd = &cobra.Command{
	Use:   "delete [contact-id]",
	Short: "Delete a contact (requires --yes)",
	Long: `PERMANENTLY delete a contact. Requires --yes. Contacts in use by a
registered domain cannot be deleted.

Examples:
  simple contacts delete 42 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("contact ID", args[0])
		if err != nil {
			return err
		}
		if err := confirmWithYes(cmd, "permanently delete a contact", []string{fmt.Sprintf("%-14s %d", "Contact ID:", id)}); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		if err := deleteContact(ctx, app.Client, app.AccountID, id); err != nil {
			return err
		}
		if printJSON(map[string]interface{}{"id": id, "deleted": true}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Contact %d deleted.", id)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(contactsCmd)

	contactsCmd.AddCommand(contactsListCmd)
	addPageFlags(contactsListCmd)
	contactsListCmd.Flags().String("sort", "", "Sort order (e.g. id:asc, label:desc, email:asc)")

	contactsCmd.AddCommand(contactsGetCmd)

	contactsCmd.AddCommand(contactsCreateCmd)
	addContactFlags(contactsCreateCmd)

	contactsCmd.AddCommand(contactsUpdateCmd)
	addContactFlags(contactsUpdateCmd)

	contactsCmd.AddCommand(contactsDeleteCmd)
	contactsDeleteCmd.Flags().Bool("yes", false, "Confirm the deletion")
}
