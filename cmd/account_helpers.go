package cmd

import (
	"fmt"
	"strconv"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// errNeedsYes is returned when a money-spending or destructive command runs without --yes.
type errNeedsYes struct{ action string }

func (e errNeedsYes) Error() string {
	return fmt.Sprintf("refusing to %s without --yes", e.action)
}

// confirmWithYes prints a preview of what would happen and returns an error
// unless the command's --yes flag is set.
func confirmWithYes(cmd *cobra.Command, action string, preview []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if yes {
		return nil
	}
	// A missing --yes is a deliberate stop, not a usage mistake.
	cmd.SilenceUsage = true
	fmt.Println(ui.Warn("This would " + action + ":"))
	for _, line := range preview {
		fmt.Println("  " + line)
	}
	fmt.Println()
	fmt.Println(ui.Info("Re-run with --yes to proceed."))
	return errNeedsYes{action: action}
}

// parseIDArg parses a numeric ID argument with a friendly error.
func parseIDArg(what, s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid %s %q (expected a positive number)", what, s)
	}
	return id, nil
}

// pageListOptions builds SDK list options from --page/--per-page/--sort values.
func pageListOptions(page, perPage int, sort string) *dnsimple.ListOptions {
	opts := &dnsimple.ListOptions{}
	if page > 0 {
		opts.Page = dnsimpleInt(page)
	}
	if perPage > 0 {
		opts.PerPage = dnsimpleInt(perPage)
	}
	if sort != "" {
		opts.Sort = dnsimpleString(sort)
	}
	return opts
}

// listOptionsFromFlags reads --page and --per-page (and --sort if defined).
func listOptionsFromFlags(cmd *cobra.Command) *dnsimple.ListOptions {
	page, _ := cmd.Flags().GetInt("page")
	perPage, _ := cmd.Flags().GetInt("per-page")
	sort, _ := cmd.Flags().GetString("sort")
	return pageListOptions(page, perPage, sort)
}

// addPageFlags registers --page and --per-page on a list command.
func addPageFlags(c *cobra.Command) {
	c.Flags().Int("page", 0, "Page number")
	c.Flags().Int("per-page", 0, "Results per page")
}

// printPagination prints a subtle "page X of Y" footer when there is more than one page.
func printPagination(p *dnsimple.Pagination) {
	if p == nil || p.TotalPages <= 1 {
		return
	}
	fmt.Println()
	fmt.Println(ui.SubtleStyle.Render(fmt.Sprintf("  page %d of %d (%d total) — use --page to see more",
		p.CurrentPage, p.TotalPages, p.TotalEntries)))
}

// shortDate trims an ISO timestamp to its date part.
func shortDate(s string) string {
	if len(s) >= 10 {
		return s[:10]
	}
	return s
}
