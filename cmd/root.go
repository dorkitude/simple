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
  auth        Authenticate with DNSimple
  whoami      Show current identity
  domains     Manage domains
  zones       Manage DNS zones
  records     Manage DNS records
`,
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
