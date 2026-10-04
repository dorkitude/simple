package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// AnalyticsResult is the JSON shape of a DNS analytics query.
type AnalyticsResult struct {
	Headers    []string                             `json:"headers"`
	Rows       [][]interface{}                      `json:"rows"`
	Query      dnsimple.DnsAnalyticsQueryParameters `json:"query"`
	Pagination *dnsimple.Pagination                 `json:"pagination,omitempty"`
}

type analyticsParams struct {
	Groupings, StartDate, EndDate, Sort string
	Page, PerPage                       int
}

func queryDNSAnalytics(ctx context.Context, c *dnsimple.Client, accountID string, p analyticsParams) (*AnalyticsResult, error) {
	acct, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID %q", accountID)
	}
	opts := &dnsimple.DnsAnalyticsOptions{ListOptions: *pageListOptions(p.Page, p.PerPage, p.Sort)}
	if p.Groupings != "" {
		opts.Groupings = dnsimpleString(p.Groupings)
	}
	if p.StartDate != "" {
		opts.StartDate = dnsimpleString(p.StartDate)
	}
	if p.EndDate != "" {
		opts.EndDate = dnsimpleString(p.EndDate)
	}
	resp, err := c.DnsAnalytics.Query(ctx, acct, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query DNS analytics: %w", err)
	}
	return &AnalyticsResult{
		Headers:    resp.RowAndHeaderData.Headers,
		Rows:       resp.RowAndHeaderData.Rows,
		Query:      resp.Query,
		Pagination: resp.Pagination,
	}, nil
}

func formatAnalyticsCell(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return "-"
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case string:
		return x
	}
	return fmt.Sprint(v)
}

var analyticsCmd = &cobra.Command{
	Use:     "analytics",
	Aliases: []string{"dns-analytics"},
	Short:   "Query DNS query-volume analytics",
	Long: `Query DNS analytics: the number of DNS queries answered, optionally
grouped by date and/or zone.

Examples:
  simple analytics
  simple analytics --groupings zone_name
  simple analytics --groupings date,zone_name --start-date 2026-09-01 --end-date 2026-09-30
  simple analytics --groupings date --sort date:desc --page 2
  simple analytics --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var p analyticsParams
		p.Groupings, _ = cmd.Flags().GetString("groupings")
		p.StartDate, _ = cmd.Flags().GetString("start-date")
		p.EndDate, _ = cmd.Flags().GetString("end-date")
		p.Sort, _ = cmd.Flags().GetString("sort")
		p.Page, _ = cmd.Flags().GetInt("page")
		p.PerPage, _ = cmd.Flags().GetInt("per-page")

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		res, err := queryDNSAnalytics(ctx, app.Client, app.AccountID, p)
		if err != nil {
			return err
		}
		if printJSON(res) {
			return nil
		}
		title := "📊 DNS analytics"
		if res.Query.StartDate != "" || res.Query.EndDate != "" {
			title += fmt.Sprintf(" (%s → %s)", res.Query.StartDate, res.Query.EndDate)
		}
		fmt.Println(ui.TitleStyle.Render(title))
		fmt.Println()
		if len(res.Rows) == 0 {
			fmt.Println(ui.Warn("No analytics data for this range"))
			return nil
		}

		widths := make([]int, len(res.Headers))
		for i, h := range res.Headers {
			widths[i] = len(h)
		}
		cells := make([][]string, len(res.Rows))
		for r, row := range res.Rows {
			cells[r] = make([]string, len(res.Headers))
			for i := range res.Headers {
				var v interface{}
				if i < len(row) {
					v = row[i]
				}
				s := formatAnalyticsCell(v)
				cells[r][i] = s
				if len(s) > widths[i] {
					widths[i] = len(s)
				}
			}
		}
		var hdr strings.Builder
		for i, h := range res.Headers {
			hdr.WriteString(fmt.Sprintf("%-*s  ", widths[i], h))
		}
		fmt.Println("  " + ui.SubtleStyle.Render(strings.TrimRight(hdr.String(), " ")))
		for _, row := range cells {
			var line strings.Builder
			for i, s := range row {
				if res.Headers[i] == "volume" {
					line.WriteString(fmt.Sprintf("%*s  ", widths[i], s))
				} else {
					line.WriteString(fmt.Sprintf("%-*s  ", widths[i], s))
				}
			}
			fmt.Println("  " + strings.TrimRight(line.String(), " "))
		}
		printPagination(res.Pagination)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyticsCmd)
	analyticsCmd.Flags().String("groupings", "", "Comma-separated groupings: date, zone_name")
	analyticsCmd.Flags().String("start-date", "", "Start date (YYYY-MM-DD)")
	analyticsCmd.Flags().String("end-date", "", "End date (YYYY-MM-DD)")
	analyticsCmd.Flags().String("sort", "", "Sort order (e.g. volume:desc, date:asc, zone_name:asc)")
	addPageFlags(analyticsCmd)
}
