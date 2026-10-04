package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dorkitude/simple/internal/tui"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var (
	jsonOutput  bool
	accountFlag string
	sandboxFlag bool
	noColorFlag bool
)

// BinName returns the name this binary was invoked as.
func BinName() string {
	return filepath.Base(os.Args[0])
}

var rootCmd = &cobra.Command{
	Use:   "simple",
	Short: "A CLI for DNSimple DNS management",
	Long: ui.TitleStyle.Render("🌐 simple") + `
A beautiful CLI for managing DNS with DNSimple.

` + ui.SubtleStyle.Render("Commands:") + `
  auth          Authenticate with DNSimple
  whoami        Show current identity
  accounts      List accounts and pick the active one
  domains       Manage domains (DNSSEC, DS records, email forwards, pushes)
  pushes        Accept or reject domain pushes sent to you
  zones         Manage DNS zones (NS records, batch changes)
  records       Manage DNS records
  registrar     Register, transfer, renew; name servers, locks, privacy
  certificates  SSL certificates and Let's Encrypt
  contacts      Registrant contacts
  templates     Record templates
  services      One-click services
  secondary     Secondary DNS primaries and zones
  vanity        Vanity name servers
  webhooks      Webhooks
  tlds          Supported TLDs and their attributes
  billing       Billing charges
  analytics     DNS query analytics
`,
	SilenceErrors: true, // Execute prints errors itself, styled
	SilenceUsage:  true, // an API error is not a usage mistake
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.Run()
	},
}

// Execute runs the root command.
func Execute() {
	// Adapt the root command name to whatever binary name was used
	rootCmd.Use = BinName()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, ui.Err(err.Error()))
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	rootCmd.PersistentFlags().StringVar(&accountFlag, "account", "", "DNSimple account ID (overrides cached)")
	rootCmd.PersistentFlags().BoolVar(&sandboxFlag, "sandbox", false, "Use DNSimple sandbox API")
	rootCmd.PersistentFlags().BoolVar(&noColorFlag, "no-color", false, "Disable colored output")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
