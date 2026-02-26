package cmd

import (
	"github.com/dorkitude/simple/internal/tui"
	"github.com/spf13/cobra"
)

var demoCmd = &cobra.Command{
	Use:   "demo",
	Short: "Launch the TUI in demo mode (fake data, no auth required)",
	Long:  `Launch the full TUI using a static in-memory demo backend so you can explore the interface safely.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return tui.RunDemo()
	},
}

func init() {
	rootCmd.AddCommand(demoCmd)
}
