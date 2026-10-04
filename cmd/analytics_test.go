package cmd

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestQueryDNSAnalytics(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":{"rows":[["2026-09-01","example.com",1200]],"headers":["date","zone_name","volume"]},"query":{"account_id":1010,"start_date":"2026-09-01","end_date":"2026-09-30","sort":"volume:desc","page":2,"per_page":50,"groupings":"date,zone_name"},"pagination":{"current_page":2,"per_page":50,"total_entries":51,"total_pages":2}}`)
	res, err := queryDNSAnalytics(context.Background(), c, "1010", analyticsParams{
		Groupings: "date,zone_name", StartDate: "2026-09-01", EndDate: "2026-09-30", Sort: "volume:desc", Page: 2, PerPage: 50,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodGet, "/v2/1010/dns_analytics")
	q, _ := url.ParseQuery(got.Query)
	for k, v := range map[string]string{"groupings": "date,zone_name", "start_date": "2026-09-01", "end_date": "2026-09-30", "sort": "volume:desc", "page": "2", "per_page": "50"} {
		if q.Get(k) != v {
			t.Errorf("query %s = %q, want %q", k, q.Get(k), v)
		}
	}
	if len(res.Rows) != 1 || res.Headers[2] != "volume" || formatAnalyticsCell(res.Rows[0][2]) != "1200" {
		t.Errorf("unexpected result %+v", res)
	}
	if res.Pagination == nil || res.Pagination.TotalPages != 2 {
		t.Errorf("unexpected pagination %+v", res.Pagination)
	}
}

func TestQueryDNSAnalyticsBadAccount(t *testing.T) {
	c, calls := recordingClient(t, 200, `{}`)
	if _, err := queryDNSAnalytics(context.Background(), c, "not-a-number", analyticsParams{}); err == nil {
		t.Fatal("expected error for non-numeric account")
	}
	if len(*calls) != 0 {
		t.Errorf("expected no API calls")
	}
}
