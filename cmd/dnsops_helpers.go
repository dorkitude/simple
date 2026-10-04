package cmd

import (
	"fmt"
	"strconv"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// errNeedsYes is returned when a destructive command runs without --yes.
var errNeedsYes = fmt.Errorf("refusing to continue without --yes")

// addYesFlag registers the --yes confirmation flag on a destructive command.
func addYesFlag(c *cobra.Command, what string) {
	c.Flags().BoolP("yes", "y", false, "Confirm: "+what)
}

// requireYes returns nil when --yes was given. Otherwise it prints what the
// command would do and returns errNeedsYes so the process exits non-zero.
func requireYes(cmd *cobra.Command, plan ...string) error {
	if yes, _ := cmd.Flags().GetBool("yes"); yes {
		return nil
	}
	cmd.SilenceUsage = true
	fmt.Println(ui.Warn("This would:"))
	for _, p := range plan {
		fmt.Printf("  %s %s\n", ui.AccentStyle.Render("•"), p)
	}
	fmt.Println(ui.Info("Re-run with --yes to do it."))
	return errNeedsYes
}

// parseID parses a positive numeric API identifier.
func parseID(kind, s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s ID %q (must be a positive number)", kind, s)
	}
	return id, nil
}

// addPageFlags registers --page and --per-page on a list command.
func addPageFlags(c *cobra.Command) {
	c.Flags().Int("page", 0, "Page number")
	c.Flags().Int("per-page", 0, "Results per page")
}

// pageOptions builds SDK list options from --page and --per-page.
func pageOptions(cmd *cobra.Command) *dnsimple.ListOptions {
	opts := &dnsimple.ListOptions{}
	if page, _ := cmd.Flags().GetInt("page"); page > 0 {
		opts.Page = dnsimpleInt(page)
	}
	if perPage, _ := cmd.Flags().GetInt("per-page"); perPage > 0 {
		opts.PerPage = dnsimpleInt(perPage)
	}
	return opts
}

// shortDate trims an API timestamp down to its date part.
func shortDate(ts string) string {
	if len(ts) >= 10 {
		return ts[:10]
	}
	return ts
}
