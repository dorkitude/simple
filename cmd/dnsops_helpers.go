package cmd

import (
	"fmt"
	"strconv"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// errRefusedWithoutYes is returned when a destructive command runs without --yes.
var errRefusedWithoutYes = fmt.Errorf("refusing to continue without --yes")

// addYesFlag registers the --yes confirmation flag on a destructive command.
func addYesFlag(c *cobra.Command, what string) {
	c.Flags().BoolP("yes", "y", false, "Confirm: "+what)
}

// requireYes returns nil when --yes was given. Otherwise it prints what the
// command would do and returns errRefusedWithoutYes so the process exits non-zero.
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
	return errRefusedWithoutYes
}

// parseID parses a positive numeric API identifier.
func parseID(kind, s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s ID %q (must be a positive number)", kind, s)
	}
	return id, nil
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
